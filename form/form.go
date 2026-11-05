// Package form is a shell screen that asks for a few lines of text: one
// labelled field per value, each a tap target, with a hint and a message
// when its value doesn't check out, and Submit and Cancel buttons.
//
// Keys and taps share one path: tab and shift+tab (or up and down) move
// between fields, enter submits, esc cancels. Tapping a field focuses it;
// tapping a button flashes it, then acts.
//
// A form doesn't do the work itself. When every field checks out, it shows
// itself busy and runs Options.Submit with the values; the app does the
// job and answers with a Done message, which closes the form or shows why
// it failed.
//
//	f := form.New(form.Options{
//		Title:  "New address",
//		Submit: "Create",
//		Fields: []form.Field{{Label: "Domain", Required: true}},
//		OnSubmit: func(v []string) tea.Cmd { return create(v[0]) },
//	})
//	return f, shell.PushCmd(f)
package form

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ionut-maxim/rexkit/components"
	"github.com/ionut-maxim/rexkit/icons"
	"github.com/ionut-maxim/rexkit/shell"
	"github.com/ionut-maxim/rexkit/theme"
	"github.com/ionut-maxim/rexkit/touch"
)

// Field is one line of text to ask for.
type Field struct {
	Label       string
	Placeholder string
	// Hint is a quiet line under the field.
	Hint string
	// Value is the starting text.
	Value string
	// Required refuses an empty value.
	Required bool
	// CharLimit caps the length (0: 256).
	CharLimit int
	// Check, if set, vets the trimmed value; its error is shown under the
	// field.
	Check func(string) error
}

// Options configure a form.
type Options struct {
	// Title is the breadcrumb, and the heading over the fields.
	Title string
	// Intro is an optional quiet line under the heading.
	Intro  string
	Fields []Field
	// Submit labels the submit button ("Submit" by default).
	Submit string
	// OnSubmit runs with every field's trimmed value, in order, once they
	// all check out. Its command does the job and ends with a Done.
	OnSubmit func(values []string) tea.Cmd
	// Tall says whether there's room for tall buttons (shell.Frame.Tall).
	Tall func() bool
}

// Done ends a submit: with no Err the form closes (and Status, if set, goes
// to the status line); with an Err, the form shows it and can be sent
// again.
type Done struct {
	Err    error
	Status string
}

// Form is the screen.
type Form struct {
	o      Options
	inputs []textinput.Model
	errs   []string // per field
	focus  int
	busy   bool
	err    string // from the last Done

	zones   []touch.Zone
	pressed tea.Msg
	width   int

	keys struct{ Submit, Next, Prev, Cancel key.Binding }
}

// Messages the form's own taps send.
type (
	focusMsg   struct{ field int }
	submitMsg  struct{}
	cancelMsg  struct{}
	releaseMsg struct{ target tea.Msg }
)

// New returns a form.
func New(o Options) *Form {
	if o.Submit == "" {
		o.Submit = "Submit"
	}
	f := &Form{o: o, errs: make([]string, len(o.Fields))}
	for _, fd := range o.Fields {
		in := textinput.New()
		in.Prompt = ""
		in.Placeholder = fd.Placeholder
		in.CharLimit = fd.CharLimit
		if in.CharLimit == 0 {
			in.CharLimit = 256
		}
		in.SetValue(fd.Value)
		f.inputs = append(f.inputs, in)
	}
	f.restyle()
	f.keys.Submit = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", strings.ToLower(o.Submit)))
	f.keys.Next = key.NewBinding(key.WithKeys("tab", "down"), key.WithHelp("tab", "next"))
	f.keys.Prev = key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("shift+tab", "previous"))
	f.keys.Cancel = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))
	return f
}

// restyle paints the inputs in the current palette.
func (f *Form) restyle() {
	for i := range f.inputs {
		st := f.inputs[i].Styles()
		text := lipgloss.NewStyle().Foreground(theme.Pal.Text)
		ph := lipgloss.NewStyle().Foreground(theme.Pal.Muted)
		st.Focused.Text, st.Blurred.Text = text, text
		st.Focused.Placeholder, st.Blurred.Placeholder = ph, ph
		st.Cursor.Color = theme.Pal.Accent
		st.Cursor.Blink = false // a steady cursor; blinking costs a timer per key
		f.inputs[i].SetStyles(st)
	}
}

// Values are the fields' trimmed values, in order.
func (f *Form) Values() []string {
	out := make([]string, len(f.inputs))
	for i, in := range f.inputs {
		out[i] = strings.TrimSpace(in.Value())
	}
	return out
}

// SetValue sets field i's text.
func (f *Form) SetValue(i int, v string) { f.inputs[i].SetValue(v) }

// Busy says whether a submit is running.
func (f *Form) Busy() bool { return f.busy }

// Focused is the field with the cursor.
func (f *Form) Focused() int { return f.focus }

// Init focuses the first field.
func (f *Form) Init() tea.Cmd {
	if len(f.inputs) == 0 {
		return nil
	}
	return f.inputs[f.focus].Focus()
}

// Title implements shell.Screen.
func (f *Form) Title() string { return f.o.Title }

// Capturing implements shell.Screen: the form takes text.
func (f *Form) Capturing() bool { return true }

// Refresh implements shell.Screen.
func (f *Form) Refresh() tea.Cmd { return nil }

// Keys implements shell.Screen.
func (f *Form) Keys() []key.Binding {
	if f.busy {
		return nil
	}
	return []key.Binding{f.keys.Submit, f.keys.Next, f.keys.Cancel}
}

func (f *Form) setFocus(i int) tea.Cmd {
	if len(f.inputs) == 0 {
		return nil
	}
	f.inputs[f.focus].Blur()
	f.focus = (i%len(f.inputs) + len(f.inputs)) % len(f.inputs)
	return f.inputs[f.focus].Focus()
}

// check vets every field, and focuses the first that fails.
func (f *Form) check() bool {
	ok := true
	for i, v := range f.Values() {
		f.errs[i] = ""
		fd := f.o.Fields[i]
		switch {
		case fd.Required && v == "":
			f.errs[i] = fd.Label + " is needed"
		case fd.Check != nil && v != "":
			if err := fd.Check(v); err != nil {
				f.errs[i] = err.Error()
			}
		}
		if f.errs[i] != "" && ok {
			ok = false
			f.setFocus(i)
		}
	}
	return ok
}

func (f *Form) submit() tea.Cmd {
	if f.busy {
		return nil
	}
	f.err = ""
	if !f.check() || f.o.OnSubmit == nil {
		return nil
	}
	f.busy = true
	return f.o.OnSubmit(f.Values())
}

// Update implements shell.Screen.
func (f *Form) Update(msg tea.Msg) (shell.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width = msg.Width
		return f, nil
	case shell.ThemeChanged:
		f.restyle()
		return f, nil
	case Done:
		f.busy = false
		if msg.Err != nil {
			f.err = msg.Err.Error()
			return f, nil
		}
		cmds := []tea.Cmd{func() tea.Msg { return shell.Pop{} }}
		if msg.Status != "" {
			cmds = append(cmds, func() tea.Msg { return shell.Status(msg.Status) })
		}
		return f, tea.Sequence(cmds...)
	case tea.MouseClickMsg:
		m := msg.Mouse()
		if m.Button != tea.MouseLeft || f.busy {
			return f, nil
		}
		if target, ok := touch.Hit(f.zones, m.X, m.Y); ok {
			if fm, ok := target.(focusMsg); ok {
				return f, f.setFocus(fm.field)
			}
			f.pressed = target
			return f, tea.Tick(components.PressDelay, func(time.Time) tea.Msg { return releaseMsg{target} })
		}
		return f, nil
	case releaseMsg:
		f.pressed = nil
		switch msg.target.(type) {
		case submitMsg:
			return f, f.submit()
		case cancelMsg:
			return f, func() tea.Msg { return shell.Pop{} }
		}
		return f, nil
	case tea.KeyPressMsg:
		if f.busy {
			return f, nil
		}
		switch {
		case key.Matches(msg, f.keys.Submit):
			return f, f.submit()
		case key.Matches(msg, f.keys.Cancel):
			return f, func() tea.Msg { return shell.Pop{} }
		case key.Matches(msg, f.keys.Next):
			return f, f.setFocus(f.focus + 1)
		case key.Matches(msg, f.keys.Prev):
			return f, f.setFocus(f.focus - 1)
		}
	}
	if len(f.inputs) == 0 || f.busy {
		return f, nil
	}
	before := f.inputs[f.focus].Value()
	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	if f.inputs[f.focus].Value() != before {
		f.errs[f.focus], f.err = "", ""
	}
	return f, cmd
}

func (f *Form) tall() bool { return f.o.Tall != nil && f.o.Tall() }

// View implements shell.Screen. Compact puts each label over its field;
// wider screens put labels in a column on the left. The form is never
// wider than 96 columns.
func (f *Form) View(width, _ int) string {
	f.zones = f.zones[:0]
	const pad = 2
	w := min(width, 96)
	inner := max(w-2*pad, 10)
	sz := components.SizeOf(width)
	tall := f.tall()
	padS := strings.Repeat(" ", pad)

	var lines []string
	add := func(s string) { lines = append(lines, ansi.Truncate(s, width, "…")) }
	add("")
	add(padS + theme.Styles.Title.Render(f.o.Title))
	if f.o.Intro != "" {
		for _, l := range strings.Split(lipgloss.NewStyle().Width(inner).Render(f.o.Intro), "\n") {
			add(padS + theme.Styles.Subtle.Render(strings.TrimRight(l, " ")))
		}
	}
	add("")

	labelW := 0
	for _, fd := range f.o.Fields {
		labelW = max(labelW, ansi.StringWidth(fd.Label)+1)
	}
	stacked := sz == components.Compact
	boxW := inner - 2
	if !stacked {
		boxW = inner - labelW - 3
	}

	for i, fd := range f.o.Fields {
		focused := i == f.focus
		label := fd.Label
		if fd.Required {
			label += theme.Styles.Accent.Render("*")
		}
		lstyle := theme.Styles.Subtle
		edge := theme.Styles.Muted.Render("│")
		if focused {
			lstyle = theme.Styles.Title
			edge = theme.Styles.Accent.Render("▌")
		}
		in := f.inputs[i]
		in.SetWidth(max(boxW-1, 4))
		box := components.FillBg(" "+in.View(), boxW, theme.Pal.Surface)
		if focused {
			box = components.FillBg(" "+in.View(), boxW, theme.Pal.SurfaceHi)
		}
		top := len(lines)
		var row []string
		if stacked {
			row = append(row, padS+edge+" "+lstyle.Render(label), padS+edge+" "+box)
		} else {
			l := lstyle.Render(label) + strings.Repeat(" ", max(labelW-ansi.StringWidth(fd.Label)-boolInt(fd.Required), 0))
			row = append(row, padS+edge+" "+l+" "+box)
		}
		switch {
		case f.errs[i] != "":
			row = append(row, padS+edge+" "+subIndent(stacked, labelW)+theme.Styles.Err.Render(icons.Current.Warning+" "+f.errs[i]))
		case fd.Hint != "":
			row = append(row, padS+edge+" "+subIndent(stacked, labelW)+theme.Styles.Faint.Render(fd.Hint))
		}
		for _, r := range row {
			add(r)
		}
		f.zones = append(f.zones, touch.Zone{Y: top, H: len(row), X0: 0, X1: w, Msg: focusMsg{i}})
		add("")
		if tall {
			add("")
		}
	}

	if f.err != "" {
		for _, l := range strings.Split(lipgloss.NewStyle().Width(inner-2).Render(f.err), "\n") {
			add(padS + theme.Styles.Err.Render(strings.TrimRight(l, " ")))
		}
		add("")
	}

	// The buttons, on the right on wider screens.
	if f.busy {
		add(padS + theme.Styles.Status.Render(icons.Current.Spinner+" "+f.o.Submit+"…"))
		return strings.Join(lines, "\n")
	}
	submitBg, cancelBg := theme.Pal.AccentSoft, theme.Pal.SurfaceHi
	if _, ok := f.pressed.(submitMsg); ok {
		submitBg = theme.Pal.Accent
	}
	if _, ok := f.pressed.(cancelMsg); ok {
		cancelBg = theme.Pal.Accent
	}
	submit := components.Btn("", icons.Current.Check+" "+f.o.Submit, submitBg, tall)
	cancel := components.Btn("", "Cancel", cancelBg, tall)
	x := pad
	if !stacked {
		x = pad + inner - submit.W - 1 - cancel.W
	}
	segs := []components.Seg{components.Plain(strings.Repeat(" ", x)), cancel, components.Plain(" "), submit}
	top, h := len(lines), 1
	if tall {
		h = 3
	}
	f.zones = append(f.zones,
		touch.Zone{Y: top, H: h, X0: x, X1: x + cancel.W, Msg: cancelMsg{}},
		touch.Zone{Y: top, H: h, X0: x + cancel.W + 1, X1: x + cancel.W + 1 + submit.W, Msg: submitMsg{}})
	for _, l := range components.RenderSegs(segs, tall) {
		add(l)
	}
	return strings.Join(lines, "\n")
}

// subIndent lines a hint or message up with the field's text, which
// starts one cell into its box.
func subIndent(stacked bool, labelW int) string {
	if stacked {
		return " "
	}
	return strings.Repeat(" ", labelW+2)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
