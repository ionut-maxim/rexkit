package shell

import (
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/ionut-maxim/markdown/kitty"

	"github.com/ionut-maxim/rexkit/bigtext"
	"github.com/ionut-maxim/rexkit/theme"
	"github.com/ionut-maxim/rexkit/touch"
)

// The action bar (Options.ActionBar) puts the screen's main actions in
// buttons big enough for a finger, and adapts to the terminal's shape,
// which on a phone changes with every turn:
//
//   - barBottom: a row of equal buttons three lines tall under the body
//     (portrait);
//   - barRail: buttons stacked on the right of the body (landscape: wide
//     and short, so the body keeps its height);
//   - barHints: the one-line key hints, when neither fits.
//
// What doesn't fit, and the frame's own keys (back, refresh, help,
// quit), are in the More sheet. Inside a terminal with kitty graphics
// each button is an image (bigtext.DrawPill); elsewhere a rounded
// outline.
type barMode int

const (
	barHints barMode = iota
	barBottom
	barRail
)

const (
	barRows  = 3  // a button's height
	railCols = 22 // the rail's width, gap included
)

func (m barMode) String() string { return [...]string{"hints", "bottom", "rail"}[m] }

// barMode is the bar's layout for the current size.
func (f *Frame) barMode() barMode {
	if !f.o.ActionBar || f.o.StartErr != nil || f.width == 0 {
		return barHints
	}
	switch {
	case f.width >= 110 && f.height < 40 && f.height >= 14:
		return barRail
	case f.height >= 18 && f.width >= 36:
		return barBottom
	}
	return barHints
}

// bodyWidth is the screens' width: all of it, but the rail's.
func (f *Frame) bodyWidth() int {
	if f.barMode() == barRail {
		return max(f.width-railCols, 1)
	}
	return f.width
}

// barAction is a button: a binding's first key, sent when tapped.
type barAction struct {
	key   string // as the hint shows it
	label string
	msg   tea.Msg
}

// moreMsg opens the More sheet.
type moreMsg struct{}

func actionOf(b key.Binding) barAction {
	h := b.Help()
	return barAction{key: keyGlyph(h.Key), label: capital(h.Desc), msg: touch.KeyMsg(b.Keys()[0])}
}

// keyGlyph shortens a key's name for a button.
func keyGlyph(k string) string {
	switch k {
	case "enter":
		return "⏎"
	case "esc":
		return "⎋"
	case "up", "↑":
		return "↑"
	case "down", "↓":
		return "↓"
	case "tab":
		return "⇥"
	case "space":
		return "␣"
	}
	if utf8.RuneCountInString(k) > 3 {
		if i := strings.IndexAny(k, " /"); i > 0 {
			return k[:i]
		}
	}
	return k
}

func capital(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if n == 0 {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}

// actions splits the bindings into the screen's (in its order: the
// first is its main action) and the frame's.
func (f *Frame) actions() (screen, frame []barAction) {
	top := f.Top()
	capturing := false
	if top != nil {
		capturing = top.Capturing()
		for _, b := range top.Keys() {
			if b.Enabled() && len(b.Keys()) > 0 {
				screen = append(screen, actionOf(b))
			}
		}
	}
	if capturing {
		return screen, nil
	}
	// The More sheet has its own Run and Close: no More on More.
	if _, ok := top.(*moreSheet); ok {
		return screen, nil
	}
	if len(f.stack) > 1 {
		frame = append(frame, actionOf(f.keys.Back))
	}
	frame = append(frame, actionOf(f.keys.Refresh), actionOf(f.keys.Quit))
	return screen, frame
}

// barButton is a button placed in the bar: in cells, from the bar's (or
// the rail's) top left.
type barButton struct {
	act     barAction
	x, y, w int
	h       int // lines; barRows when 0
	primary bool
}

func (b barButton) rows() int {
	if b.h > 0 {
		return b.h
	}
	return barRows
}

// barButtons lays out the buttons for mode: as many of the screen's
// actions as fit, then More.
func (f *Frame) barButtons(mode barMode, bodyH int) []barButton {
	screen, frame := f.actions()
	var slots int
	switch mode {
	case barBottom:
		minW := 15
		if f.width < 64 {
			minW = 12
		}
		slots = max((f.width-1)/(minW+1), 2)
	case barRail:
		slots = max(bodyH/barRows, 2)
	default:
		return nil
	}
	shown := screen
	more := len(frame) > 0
	if !more && len(shown) <= slots {
		slots = len(shown) + 1 // everything fits: no More
	}
	if len(shown)+len(frame) > slots || (more && len(shown) >= slots) {
		shown = shown[:min(len(shown), slots-1)]
	}
	if len(screen) > len(shown) {
		more = true
	}
	acts := append([]barAction(nil), shown...)
	if more {
		acts = append(acts, barAction{key: "⋯", label: "More", msg: moreMsg{}})
	}
	var out []barButton
	switch mode {
	case barBottom:
		// Equal widths across the screen, a cell between, and a cell's
		// margin at each end.
		n := len(acts)
		room := f.width - 2 - (n - 1)
		x := 1
		for i, a := range acts {
			w := room / n
			if i < room%n {
				w++
			}
			out = append(out, barButton{act: a, x: x, y: 0, w: w, primary: i == 0 && len(shown) > 0})
			x += w + 1
		}
	case barRail:
		for i, a := range acts {
			out = append(out, barButton{act: a, x: 0, y: i * barRows, w: railCols - 1, primary: i == 0 && len(shown) > 0})
		}
	}
	return out
}

// pillKey is a drawn button image: what it shows, its size and look.
type pillKey struct {
	key, label       string
	cols, rows       int
	primary, pressed bool
	dark             bool
	cellW, cellH     int
}

func (f *Frame) pillKey(b barButton) pillKey {
	return pillKey{key: b.act.key, label: b.act.label, cols: b.w, rows: b.rows(),
		primary: b.primary, pressed: f.isPressed(b.act.msg), dark: f.term.Dark,
		cellW: f.term.Cell.W, cellH: f.term.Cell.H}
}

func (f *Frame) isPressed(msg tea.Msg) bool {
	return f.pressed != nil && reflect.DeepEqual(f.pressed, msg)
}

// imagesOn says the bar's buttons are drawn as images.
func (f *Frame) imagesOn() bool { return f.term.Kitty && f.term.Cell.W > 0 && f.term.Cell.H > 0 }

// prepareBar draws and sends the images the bar shows now, after every
// update (never in View), and frees old ones.
func (f *Frame) prepareBar() tea.Cmd {
	mode := f.barMode()
	if mode == barHints || !f.imagesOn() {
		return nil
	}
	if f.pills == nil {
		f.pills = map[pillKey]int{}
	}
	_, bodyH := f.layoutHeights(mode)
	var seq strings.Builder
	want := map[pillKey]bool{}
	buttons := append(f.barButtons(mode, bodyH), f.navButtons(mode)...)
	for _, b := range buttons {
		k := f.pillKey(b)
		want[k] = true
		if _, ok := f.pills[k]; ok {
			continue
		}
		img := bigtext.DrawPill(bigtext.Pill{Key: k.key, Label: k.label, Primary: k.primary, Pressed: k.pressed}, k.cols, k.rows, f.term.Cell)
		id := kitty.NewID()
		t, err := kitty.FromImage(id, img).TransmitSharp(k.cols, k.rows)
		if err != nil {
			continue
		}
		seq.WriteString(t)
		f.pills[k] = id
	}
	// Keep the last screens' buttons for going back and forth.
	if len(f.pills) > 48 {
		for k, id := range f.pills {
			if !want[k] {
				seq.WriteString(kitty.Delete(id))
				delete(f.pills, k)
			}
		}
	}
	if seq.Len() == 0 {
		return nil
	}
	return tea.Raw(seq.String())
}

// layoutHeights is the header's and the body's height for mode.
func (f *Frame) layoutHeights(mode barMode) (headerH, bodyH int) {
	headerH = f.navHeight(mode)
	footerH := f.statusLines()
	if mode == barBottom {
		footerH += barRows
	}
	return headerH, max(f.height-headerH-footerH, 1)
}

// statusLines is how many lines the error or status takes (0 or 1).
func (f *Frame) statusLines() int {
	if f.err != nil || f.status != "" {
		return 1
	}
	return 0
}

// drawButton is a button as lines: its image's placeholders, or a
// rounded outline when there's no image.
func (f *Frame) drawButton(b barButton) []string {
	if f.imagesOn() {
		if id, ok := f.pills[f.pillKey(b)]; ok {
			return strings.Split(kitty.Placeholders(id, b.w, b.rows()), "\n")
		}
	}
	return textButton(b.act, b.w, b.rows(), b.primary, f.isPressed(b.act.msg))
}

// textButton draws a button in text: a block of the button colour three
// lines tall, the key and the label centred on the middle line. The same
// colours as everywhere else; square, since text can't round corners
// without seams.
func textButton(a barAction, w, rows int, primary, pressed bool) []string {
	w = max(w, 4)
	bg := theme.Pal.Button
	keyC := theme.Pal.Subtle
	if primary {
		bg, keyC = theme.Pal.ButtonOn, theme.Pal.Accent
	}
	if pressed {
		bg, keyC = theme.Pal.ButtonPress, theme.Pal.Text
	}
	fill := lipgloss.NewStyle().Background(bg)
	text := a.key + " " + a.label
	if a.key == "" {
		text = a.label
	}
	if ansi.StringWidth(text) > w-2 {
		text = a.label
	}
	if ansi.StringWidth(text) > w-2 {
		text = ansi.Truncate(text, max(w-2, 1), "…")
	}
	tw := ansi.StringWidth(text)
	left := (w - tw) / 2
	right := w - tw - left
	labelSt := fill.Foreground(theme.Pal.Text)
	var mid string
	if a.key != "" && strings.HasPrefix(text, a.key+" ") {
		mid = fill.Foreground(keyC).Bold(true).Render(a.key) + labelSt.Render(" "+a.label)
	} else {
		mid = labelSt.Render(text)
	}
	blank := fill.Render(strings.Repeat(" ", w))
	out := make([]string, max(rows, 1))
	for i := range out {
		out[i] = blank
	}
	out[(len(out)-1)/2] = fill.Render(strings.Repeat(" ", left)) + mid + fill.Render(strings.Repeat(" ", right))
	return out
}

// bottomBar draws the bar under the body; its zones are from its top.
func (f *Frame) bottomBar(buttons []barButton) ([]string, []touch.Zone) {
	lines := make([]string, barRows)
	var zones []touch.Zone
	x := 0
	for _, b := range buttons {
		pad := strings.Repeat(" ", max(b.x-x, 0))
		for i, l := range f.drawButton(b) {
			lines[i] += pad + l
		}
		x = b.x + b.w
		zones = append(zones, touch.Zone{Y: 0, H: barRows, X0: b.x, X1: b.x + b.w, Msg: b.act.msg})
	}
	return lines, zones
}

// railLines draws the rail, height lines; its zones are from its top
// left.
func (f *Frame) railLines(buttons []barButton, height int) ([]string, []touch.Zone) {
	lines := make([]string, height)
	var zones []touch.Zone
	for _, b := range buttons {
		for i, l := range f.drawButton(b) {
			if b.y+i < height {
				lines[b.y+i] = l
			}
		}
		zones = append(zones, touch.Zone{Y: b.y, H: barRows, X0: 0, X1: b.w, Msg: b.act.msg})
	}
	return lines, zones
}

// joinRail puts the rail to the right of the body, line by line.
func joinRail(body string, bodyW int, rail []string) string {
	lines := strings.Split(body, "\n")
	for i := range lines {
		w := ansi.StringWidth(lines[i])
		if w > bodyW {
			lines[i] = ansi.Truncate(lines[i], bodyW, "")
			w = bodyW
		}
		if i < len(rail) {
			lines[i] += strings.Repeat(" ", bodyW-w+1) + rail[i]
		}
	}
	return strings.Join(lines, "\n")
}

// moreSheet lists every action: the screen's, then the frame's, one big
// entry each. A tap or enter closes it and runs the action on the screen
// it was opened from.
type moreSheet struct {
	title  string // "More" when empty
	acts   []barAction
	cursor int
	cells  []sheetCell // where each entry is, in the last View
}

// sheetCell is where an entry was drawn: its first line and columns.
type sheetCell struct{ y, x0, x1 int }

func newMoreSheet(acts []barAction) *moreSheet { return &moreSheet{acts: acts} }

func (m *moreSheet) Init() tea.Cmd { return nil }
func (m *moreSheet) Title() string {
	if m.title != "" {
		return m.title
	}
	return "More"
}
func (m *moreSheet) Capturing() bool  { return false }
func (m *moreSheet) Refresh() tea.Cmd { return nil }
func (m *moreSheet) Keys() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
	}
}

// sheetRun closes the More sheet, then runs msg on the screen below.
type sheetRun struct{ msg tea.Msg }

// run closes the sheet, then sends the action to the screen below.
func (m *moreSheet) run(i int) tea.Cmd {
	msg := m.acts[i].msg
	return func() tea.Msg { return sheetRun{msg} }
}

func (m *moreSheet) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			m.cursor = max(m.cursor-1, 0)
		case "down", "j":
			m.cursor = min(m.cursor+1, len(m.acts)-1)
		case "enter":
			return m, m.run(m.cursor)
		}
	case tea.MouseClickMsg:
		mo := msg.Mouse()
		for i, c := range m.cells {
			if mo.Y >= c.y && mo.Y < c.y+barRows && mo.X >= c.x0 && mo.X < c.x1 {
				m.cursor = i
				return m, m.run(i)
			}
		}
	}
	return m, nil
}

func (m *moreSheet) View(width, height int) string {
	// As wide as the bar under it, with the same margins, and a line
	// between entries so each reads as its own button. A wide screen
	// (landscape) gets columns, so every entry fits without scrolling.
	w := max(width-2, 8)
	cols := 1
	for cols < 3 && (w-(cols)*2)/(cols+1) >= 30 && (len(m.acts)+cols-1)/cols*(barRows+1)+1 > height {
		cols++
	}
	cw := (w - (cols-1)*2) / cols
	rowsPer := (len(m.acts) + cols - 1) / cols
	lines := make([]string, 1+rowsPer*(barRows+1))
	m.cells = m.cells[:0]
	for i, a := range m.acts {
		col, row := i/rowsPer, i%rowsPer
		y := 1 + row*(barRows+1)
		x := 1 + col*(cw+2)
		m.cells = append(m.cells, sheetCell{y: y, x0: x, x1: x + cw})
		for j, l := range sheetRow(a.key, a.label, cw, i == m.cursor) {
			line := lines[y+j]
			if pad := x - ansi.StringWidth(line); pad > 0 {
				line += strings.Repeat(" ", pad)
			}
			lines[y+j] = line + l
		}
	}
	return strings.Join(lines, "\n")
}

// sheetRow is a More entry: a button-coloured block three lines tall,
// the label on the left and the key on the right.
func sheetRow(k, label string, w int, sel bool) []string {
	bg := theme.Pal.Button
	if sel {
		bg = theme.Pal.ButtonOn
	}
	fill := lipgloss.NewStyle().Background(bg)
	w = max(w, 8)
	room := w - 4 - ansi.StringWidth(k) - 1
	if ansi.StringWidth(label) > room {
		label = ansi.Truncate(label, max(room, 1), "…")
	}
	gap := w - 4 - ansi.StringWidth(label) - ansi.StringWidth(k)
	mid := fill.Render("  ") + fill.Foreground(theme.Pal.Text).Render(label) + fill.Render(strings.Repeat(" ", max(gap, 1))) +
		fill.Foreground(theme.Pal.Subtle).Render(k) + fill.Render("  ")
	blank := fill.Render(strings.Repeat(" ", w))
	return []string{blank, mid, blank}
}

// openMore pushes the More sheet with every action of the top screen.
func (f *Frame) openMore() tea.Cmd {
	screen, frame := f.actions()
	all := append(append([]barAction(nil), screen...), frame...)
	return func() tea.Msg { return Push{newMoreSheet(all)} }
}
