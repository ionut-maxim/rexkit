// Package shell is the app frame for a terminal app made of screens: a
// header with a tab for each view and breadcrumbs, a body that shows the
// screen on top, and a footer with the status line and key hints as
// buttons. Each view keeps its own stack of screens, so switching away
// and back returns to where you were.
//
// It is built for touch (the Rex iOS client) as much as for keys: every
// tab, crumb and hint is a button with a tap zone, taps flash before they
// act, and from 40 rows up (tall) buttons get a three-row target. It also
// probes the terminal (background colour, kitty graphics, cell size) and
// reports the app's state to Rex's tab (rexstatus).
//
// An app plugs in through Options: its views, a Hook that sees every
// message first, global keys, and how to load what the views need.
package shell

import (
	"image/color"
	"reflect"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/ionut-maxim/rexkit/bigtext"
	"github.com/ionut-maxim/rexkit/components"
	"github.com/ionut-maxim/rexkit/icons"
	"github.com/ionut-maxim/rexkit/kitty"
	"github.com/ionut-maxim/rexkit/rexstatus"
	"github.com/ionut-maxim/rexkit/theme"
	"github.com/ionut-maxim/rexkit/touch"
)

// Screen is one level of a view's stack.
type Screen interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (Screen, tea.Cmd)
	// View renders the body in the given size.
	View(width, height int) string
	// Title is the breadcrumb for this screen.
	Title() string
	// Keys are the screen's own bindings, for the hints.
	Keys() []key.Binding
	// Capturing is true while the screen takes text input, so q and esc
	// go to the screen instead of quitting or going back.
	Capturing() bool
	// Refresh reloads the screen's data.
	Refresh() tea.Cmd
}

// Closer is a screen that frees terminal resources (images) when it
// closes.
type Closer interface{ Close() tea.Cmd }

// Hoverer is a screen that wants pointer motion (a desktop's hover), with
// body-relative coordinates. Others don't get it.
type Hoverer interface{ Hover() }

// Filterer is a screen whose esc first clears an applied filter.
type Filterer interface{ ClearFilter() bool }

// Term is what screens need to know about the terminal. The frame keeps
// Dark and Kitty current; share the pointer with the screens.
type Term struct {
	Dark  bool
	Kitty bool // draws kitty graphics
	Rex   bool // runs inside Rex, so tabs can be opened
}

// Images says whether to use kitty graphics.
type Images int

const (
	// ImagesAuto asks the terminal at start.
	ImagesAuto Images = iota
	ImagesOn
	ImagesOff
)

// Messages screens and hooks send to the frame.
type (
	// Push puts a screen on the current view's stack, tells it the size,
	// then runs its Init.
	Push struct{ Screen Screen }
	// Pop closes the top screen (never the view's first).
	Pop struct{}
	// PopTo goes back until Depth screens are left.
	PopTo struct{ Depth int }
	// Status shows a short notice in the status line.
	Status string
	// Err shows an error in the status line (r retries); it's also passed
	// to the screen on top.
	Err struct{ Err error }
	// Refresh reloads the screen on top.
	Refresh struct{}
	// SwitchView makes a view current (a tab tap).
	SwitchView struct{ View int }
	// ThemeChanged tells the screen on top that the colours, kitty support
	// or cell size changed.
	ThemeChanged struct{}
	// Progress reports the app's state to Rex's tab; see Report.
	Progress struct {
		State rexstatus.State
		Msg   string
	}
)

// PushCmd is a command that pushes s.
func PushCmd(s Screen) tea.Cmd { return func() tea.Msg { return Push{s} } }

// Report asks the frame to report a state to Rex's tab: Working while a
// job runs (the frame counts them), Done or Error when it ends. msg is one
// short line. A key press turns Done or Error back to Idle.
func Report(state rexstatus.State, msg string) tea.Cmd {
	return func() tea.Msg { return Progress{state, msg} }
}

// View is one of the app's views, shown as a tab.
type View struct {
	Name, Icon string
	// Root makes the view's first screen, the first time it's shown.
	Root func() Screen
	// Badge, if set, is shown after the name (a count); it gets the view's
	// first screen, once there is one.
	Badge func(root Screen) string
}

// GlobalKey is a key that works on any screen not taking input, once the
// views have started.
type GlobalKey struct {
	Binding key.Binding
	Run     func() tea.Cmd
}

// Options configure a Frame.
type Options struct {
	// App names the app: the window title and Rex's tab status.
	App   string
	Views []View
	// Start is the view to open on.
	Start int
	// Images: on, off, or ask the terminal.
	Images Images
	// StartErr, when set, is shown instead of anything else (q quits).
	StartErr error
	// Term is shared with the screens; the frame makes one when nil.
	Term *Term
	// Load runs at start, and on r while no screen is open; it ends by
	// calling Start, typically from Hook when what it loaded arrives.
	// Without Load, the views start at once.
	Load func() tea.Cmd
	// Hook sees every message before the frame; true means handled.
	Hook func(msg tea.Msg) (tea.Cmd, bool)
	// Keys are the app's global keys.
	Keys []GlobalKey
	// Context is shown quietly at the right of the header (a team, an
	// account), when there's room.
	Context func() string
	// Loading is shown while no screen is open.
	Loading func() string
	// ErrText words an error for the status line.
	ErrText func(error) string
	// OnFocus, when set, turns on the terminal's focus reports and runs
	// each time the terminal gains focus once the views have started
	// (a Rex tab coming back to the front), for example to reload.
	OnFocus func() tea.Cmd
}

// Frame is the root model.
type Frame struct {
	o    Options
	term *Term

	// pressed is the zone message of the button being tapped: it shows in
	// the accent colour until its action runs.
	pressed tea.Msg
	// prog is the state last reported to Rex; working counts jobs still
	// running.
	prog    rexstatus.State
	working int

	probing bool     // waiting for the kitty graphics answer
	started bool     // the views have started
	stack   []Screen // the current view's screens
	views   [][]Screen
	cur     int
	zones   []touch.Zone // tap zones from the last render
	bodyTop int
	bodyH   int
	width   int
	height  int

	status   string
	err      error
	showHelp bool
	keys     frameKeys
}

type frameKeys struct {
	Quit, ForceQuit, Back, Help, Refresh, NextView key.Binding
}

// releaseMsg ends a button press and runs its action.
type releaseMsg struct{ target tea.Msg }

// kittyTimeoutMsg ends detection when the terminal never answers.
type kittyTimeoutMsg struct{}

// New returns a frame.
func New(o Options) *Frame {
	t := o.Term
	if t == nil {
		t = &Term{Dark: true, Kitty: o.Images == ImagesOn}
	}
	if o.ErrText == nil {
		o.ErrText = func(err error) string { return "error: " + err.Error() }
	}
	return &Frame{o: o, term: t, cur: o.Start, views: make([][]Screen, len(o.Views)), keys: frameKeys{
		Quit:      key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		ForceQuit: key.NewBinding(key.WithKeys("ctrl+c")),
		Back:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Refresh:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		NextView:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next view")),
	}}
}

// Term is the terminal state shared with the screens.
func (f *Frame) Term() *Term { return f.term }

// Start opens the start view; until then, view switches and global keys
// wait. It does nothing once started.
func (f *Frame) Start() tea.Cmd {
	if f.started {
		return nil
	}
	f.started = true
	return f.SwitchTo(f.o.Start)
}

// Started says whether Start has run.
func (f *Frame) Started() bool { return f.started }

// Current is the current view.
func (f *Frame) Current() int { return f.cur }

// SwitchTo makes view v current, making its first screen when it's new.
// The current view again goes back to its first screen.
func (f *Frame) SwitchTo(v int) tea.Cmd {
	if !f.started {
		return nil
	}
	if v == f.cur && len(f.stack) > 0 {
		var cmds []tea.Cmd
		for len(f.stack) > 1 {
			cmds = append(cmds, f.Pop())
		}
		return tea.Batch(cmds...)
	}
	f.views[f.cur] = f.stack
	f.cur, f.stack = v, f.views[v]
	f.err, f.status = nil, ""
	if len(f.stack) == 0 {
		f.stack = nil
		_, cmd := f.Update(Push{f.o.Views[v].Root()})
		return cmd
	}
	// Tell it the size, which may have changed while it was away.
	return f.Forward(tea.WindowSizeMsg{Width: f.width, Height: f.height})
}

// Screens is every screen in every view.
func (f *Frame) Screens() []Screen {
	f.views[f.cur] = f.stack
	var out []Screen
	for _, st := range f.views {
		out = append(out, st...)
	}
	return out
}

// Root is view v's first screen, or nil.
func (f *Frame) Root(v int) Screen {
	st := f.views[v]
	if v == f.cur {
		st = f.stack
	}
	if len(st) == 0 {
		return nil
	}
	return st[0]
}

// Top is the screen on top of the current view, or nil.
func (f *Frame) Top() Screen {
	if len(f.stack) == 0 {
		return nil
	}
	return f.stack[len(f.stack)-1]
}

// Stack is the current view's screens, first to top.
func (f *Frame) Stack() []Screen { return f.stack }

// Append puts s on top as is: no size, no Init (see Push for both).
func (f *Frame) Append(s Screen) { f.stack = append(f.stack, s) }

// Pop closes and removes the top screen.
func (f *Frame) Pop() tea.Cmd {
	top := f.Top()
	f.stack = f.stack[:len(f.stack)-1]
	if c, ok := top.(Closer); ok {
		return c.Close()
	}
	return nil
}

// Forward passes msg to the screen on top.
func (f *Frame) Forward(msg tea.Msg) tea.Cmd {
	top := f.Top()
	if top == nil {
		return nil
	}
	s, cmd := top.Update(msg)
	f.stack[len(f.stack)-1] = s
	return cmd
}

// SetStatus shows a notice in the status line ("" clears it).
func (f *Frame) SetStatus(s string) { f.status = s }

// Status is the notice in the status line.
func (f *Frame) Status() string { return f.status }

// SetErr shows an error in the status line (nil clears it).
func (f *Frame) SetErr(err error) { f.err = err }

// Err is the error in the status line.
func (f *Frame) Err() error { return f.err }

// BodyTop is the row the body starts on, from the last render.
func (f *Frame) BodyTop() int { return f.bodyTop }

// Pressed is the tap target showing pressed, if any.
func (f *Frame) Pressed() tea.Msg { return f.pressed }

// ShowingHelp says whether every key hint is shown (? toggles it).
func (f *Frame) ShowingHelp() bool { return f.showHelp }

// Probing says whether the frame still waits for the terminal's kitty
// graphics answer.
func (f *Frame) Probing() bool { return f.probing }

// Size is the terminal's size.
func (f *Frame) Size() (width, height int) { return f.width, f.height }

// Init starts the probes and the app's Load.
func (f *Frame) Init() tea.Cmd {
	if f.o.StartErr != nil {
		return nil
	}
	f.prog = rexstatus.Idle
	cmds := []tea.Cmd{tea.RequestBackgroundColor}
	if f.o.Load != nil {
		cmds = append(cmds, f.o.Load())
	} else {
		// Nothing to load first: the views start right away.
		cmds = append(cmds, f.Start())
	}
	cmds = append(cmds, tea.Raw(f.report(rexstatus.Idle, "")))
	if f.o.Images != ImagesOff {
		// The cell size in pixels, for drawing images sharp.
		cmds = append(cmds, tea.Raw(ansi.WindowOp(16)))
	}
	if f.o.Images == ImagesAuto {
		f.probing = true
		cmds = append(cmds, tea.Raw(kitty.Query),
			tea.Tick(time.Second, func(time.Time) tea.Msg { return kittyTimeoutMsg{} }))
	}
	return tea.Batch(cmds...)
}

func (f *Frame) report(state rexstatus.State, msg string) string {
	return rexstatus.Report(f.o.App, state, msg)
}

// Update runs the app's Hook, then handles navigation, taps, the probes
// and global keys, and passes the rest to the screen on top.
func (f *Frame) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if f.o.Hook != nil {
		if cmd, ok := f.o.Hook(msg); ok {
			return f, cmd
		}
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width, f.height = msg.Width, msg.Height
		return f, f.Forward(msg)

	case tea.MouseClickMsg:
		m := msg.Mouse()
		if m.Button != tea.MouseLeft || f.o.StartErr != nil {
			return f, nil
		}
		if target, ok := touch.Hit(f.zones, m.X, m.Y); ok {
			// Show the button pressed, then act.
			f.pressed = target
			return f, tea.Tick(components.PressDelay, func(time.Time) tea.Msg { return releaseMsg{target} })
		}
		if m.Y >= f.bodyTop && m.Y < f.bodyTop+f.bodyH {
			m.Y -= f.bodyTop
			return f, f.Forward(tea.MouseClickMsg(m))
		}
		return f, nil

	case tea.MouseWheelMsg:
		m := msg.Mouse()
		m.Y -= f.bodyTop
		return f, f.Forward(tea.MouseWheelMsg(m))

	case tea.MouseMotionMsg:
		m := msg.Mouse()
		if m.Button != tea.MouseNone || m.Y < f.bodyTop || m.Y >= f.bodyTop+f.bodyH {
			return f, nil
		}
		if _, ok := f.Top().(Hoverer); !ok {
			return f, nil
		}
		m.Y -= f.bodyTop
		return f, f.Forward(tea.MouseMotionMsg(m))

	case Progress:
		switch msg.State {
		case rexstatus.Working:
			f.working++
		case rexstatus.Done, rexstatus.Error:
			f.working = max(f.working-1, 0)
			if f.working > 0 {
				return f, nil // others still running
			}
		}
		f.prog = msg.State
		return f, tea.Raw(f.report(msg.State, msg.Msg))

	case releaseMsg:
		f.pressed = nil
		return f.Update(msg.target)

	case PopTo:
		var cmds []tea.Cmd
		for len(f.stack) > max(msg.Depth, 1) {
			cmds = append(cmds, f.Pop())
		}
		f.err, f.status = nil, ""
		return f, tea.Batch(cmds...)

	case uv.KittyGraphicsEvent:
		if f.probing && msg.Options.ID == kitty.QueryID {
			f.probing = false
			f.term.Kitty = string(msg.Payload) == "OK"
			return f, f.Forward(ThemeChanged{})
		}
		return f, nil

	case uv.CellSizeEvent:
		if msg.Width > 0 && msg.Height > 0 {
			kitty.Cell.W, kitty.Cell.H = msg.Width, msg.Height
			bigtext.CellKnown = true
			return f, f.Forward(ThemeChanged{})
		}
		return f, nil

	case uv.PrimaryDeviceAttributesEvent, kittyTimeoutMsg:
		// No kitty answer before DA1 (or at all): no kitty graphics.
		f.probing = false
		return f, nil

	case tea.KeyPressMsg:
		// Seen: a finished job's mark goes back to idle.
		if f.prog == rexstatus.Done || f.prog == rexstatus.Error {
			f.prog = rexstatus.Idle
			cmd := tea.Raw(f.report(rexstatus.Idle, ""))
			m, next := f.Update(msg)
			return m, tea.Batch(cmd, next)
		}
		if key.Matches(msg, f.keys.ForceQuit) {
			return f, f.quit()
		}
		top := f.Top()
		capturing := top != nil && top.Capturing()
		if f.o.StartErr != nil || (!capturing && key.Matches(msg, f.keys.Quit)) {
			return f, f.quit()
		}
		if !capturing {
			if key.Matches(msg, f.keys.Help) {
				f.showHelp = !f.showHelp
				return f, nil
			}
			for _, g := range f.o.Keys {
				if f.started && key.Matches(msg, g.Binding) {
					return f, g.Run()
				}
			}
			switch {
			case key.Matches(msg, f.keys.NextView) && f.started:
				return f, f.SwitchTo((f.cur + 1) % len(f.o.Views))
			case key.Matches(msg, f.keys.Refresh):
				f.err, f.status = nil, ""
				if top == nil {
					if f.o.Load == nil {
						return f, nil
					}
					return f, f.o.Load()
				}
				return f, top.Refresh()
			case key.Matches(msg, f.keys.Back):
				// A screen with an applied filter clears it first.
				if fl, ok := top.(Filterer); ok && fl.ClearFilter() {
					return f, nil
				}
				if len(f.stack) > 1 {
					f.err, f.status = nil, ""
					return f, f.Pop()
				}
				return f, nil
			}
		}
		// Any key clears a stale notice.
		f.status = ""

	case tea.BackgroundColorMsg:
		f.term.Dark = msg.IsDark()
		theme.Set(f.term.Dark)
		return f, f.Forward(ThemeChanged{})

	case SwitchView:
		return f, f.SwitchTo(msg.View)

	case tea.FocusMsg:
		if f.o.OnFocus != nil && f.started {
			return f, f.o.OnFocus()
		}
		return f, nil

	case Refresh:
		if top := f.Top(); top != nil {
			return f, top.Refresh()
		}
		return f, nil

	case Push:
		f.stack = append(f.stack, msg.Screen)
		f.err, f.status = nil, ""
		// Tell the new screen its size right away, before its data arrives.
		s, size := msg.Screen, tea.WindowSizeMsg{Width: f.width, Height: f.height}
		s2, cmd := s.Update(size)
		f.stack[len(f.stack)-1] = s2
		return f, tea.Batch(cmd, s2.Init())

	case Pop:
		if len(f.stack) > 1 {
			return f, f.Pop()
		}
		return f, nil

	case Err:
		f.err = msg.Err
		return f, f.Forward(msg)

	case Status:
		f.status = string(msg)
		return f, nil
	}
	return f, f.Forward(msg)
}

// quit closes every screen, clears Rex's tab status, then quits.
func (f *Frame) quit() tea.Cmd {
	var cmds []tea.Cmd
	for _, s := range f.Screens() {
		if c, ok := s.(Closer); ok {
			if cmd := c.Close(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	cmds = append(cmds, tea.Raw(rexstatus.Clear))
	return tea.Sequence(append(cmds, tea.Quit)...)
}

// View draws header, body and footer, with mouse reporting for every
// motion (taps and hover).
func (f *Frame) View() tea.View {
	v := tea.NewView(f.render())
	v.AltScreen = true
	v.WindowTitle = f.o.App
	// Selecting text or following a link then needs shift+click (or the
	// terminal's own modifier).
	v.MouseMode = tea.MouseModeAllMotion
	v.ReportFocus = f.o.OnFocus != nil
	return v
}

func (f *Frame) render() string {
	f.zones = f.zones[:0]
	if f.width == 0 {
		return ""
	}
	if f.o.StartErr != nil {
		return theme.Styles.ErrBox.Width(min(f.width, 80)).Render(
			theme.Styles.ErrTitle.Render("Can't start") + "\n\n" + f.o.StartErr.Error() + "\n\n" + theme.Styles.Faint.Render("q quit"))
	}

	header := f.header()
	footer, footerZones := f.footer()
	headerH, footerH := lipgloss.Height(header), lipgloss.Height(footer)
	bodyH := max(f.height-headerH-footerH, 1)
	f.bodyTop, f.bodyH = headerH, bodyH
	for _, z := range footerZones {
		z.Y += headerH + bodyH
		f.zones = append(f.zones, z)
	}

	var body string
	if top := f.Top(); top != nil {
		body = top.View(f.width, bodyH)
	} else if f.err == nil && f.o.Loading != nil {
		body = theme.Styles.Faint.Render(f.o.Loading())
	}
	body = lipgloss.NewStyle().Width(f.width).Height(bodyH).MaxHeight(bodyH).Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

// Tall is true when there's room for tall buttons: the phone with the
// keyboard down, or a desktop.
func (f *Frame) Tall() bool { return f.height >= 40 }

// header draws the back button, the views' tabs and the breadcrumbs of
// the screens opened in the current view. Earlier screens are buttons that
// go back there. On a narrow screen the oldest crumbs fold into "…" and,
// with screens open, the tabs shrink to their icons.
func (f *Frame) header() string {
	type crumb struct {
		text  string
		depth int // screens left after tapping it
	}
	sz, tall := components.SizeOf(f.width), f.Tall()
	rowH := 1
	if tall {
		rowH = 3
	}
	var crumbs []crumb
	for i, s := range f.stack {
		if i == 0 {
			continue
		}
		crumbs = append(crumbs, crumb{s.Title(), i + 1})
	}
	sep := components.Plain(" " + theme.Styles.Muted.Render(icons.Current.Sep) + " ")
	isPressed := func(msg tea.Msg) bool { return msg != nil && reflect.DeepEqual(msg, f.pressed) }
	bgFor := func(msg tea.Msg, normal color.Color) color.Color {
		if isPressed(msg) {
			return theme.Pal.Accent
		}
		return normal
	}
	type tab struct {
		seg components.Seg
		msg tea.Msg
	}
	var tabs []tab
	tabsW := 0
	for v := range f.o.Views {
		msg := SwitchView{v}
		bg := theme.Pal.SurfaceHi
		if v == f.cur {
			bg = theme.Pal.AccentSoft
		}
		tsz := sz
		if sz == components.Compact && len(f.stack) <= 1 {
			tsz = components.Regular
		}
		t := tab{components.Btn("", f.tabLabel(v, tsz), bgFor(msg, bg), tall), msg}
		tabs = append(tabs, t)
		tabsW += t.seg.W + 1
	}
	var back components.Seg
	if len(f.stack) > 1 {
		label := icons.Current.Back + " back"
		if sz == components.Compact {
			label = icons.Current.Back
		}
		back = components.Btn("", label, bgFor(Pop{}, theme.Pal.SurfaceHi), tall)
	}
	crumbSeg := func(c crumb, last bool) components.Seg {
		if last {
			return components.Plain(theme.Styles.Title.Render(c.text))
		}
		return components.Btn("", c.text, bgFor(PopTo{c.depth}, theme.Pal.SurfaceHi), tall)
	}

	// Fold the oldest crumbs until the rest fit.
	width := func(cs []crumb, folded bool) int {
		w := tabsW
		if back.W > 0 {
			w += back.W + 1
		}
		if folded {
			w += sep.W + 1
		}
		for i, c := range cs {
			w += sep.W + crumbSeg(c, i == len(cs)-1).W
		}
		return w
	}
	folded := false
	var fold crumb
	for len(crumbs) > 1 && width(crumbs, folded) > f.width {
		fold, folded = crumbs[0], true
		crumbs = crumbs[1:]
	}

	var segs []components.Seg
	x := 0
	add := func(sg components.Seg, msg tea.Msg) {
		if msg != nil && x < f.width {
			f.zones = append(f.zones, touch.Zone{Y: 0, H: rowH, X0: x, X1: min(x+sg.W, f.width), Msg: msg})
		}
		segs = append(segs, sg)
		x += sg.W
	}
	if back.W > 0 {
		add(back, Pop{})
		add(components.Plain(" "), nil)
	}
	for i, t := range tabs {
		if i > 0 {
			add(components.Plain(" "), nil)
		}
		add(t.seg, t.msg)
	}
	if folded {
		add(sep, nil)
		add(components.Plain(theme.Styles.Muted.Render("…")), PopTo{fold.depth})
	}
	for i, c := range crumbs {
		add(sep, nil)
		last := i == len(crumbs)-1
		var msg tea.Msg
		if !last {
			msg = PopTo{c.depth}
		}
		add(crumbSeg(c, last), msg)
	}
	// The context, quietly on the right, when there's room.
	if f.o.Context != nil {
		if t := theme.Styles.Muted.Render(icons.Current.Team + " " + f.o.Context()); sz != components.Compact && x+ansi.StringWidth(t)+3 <= f.width {
			add(components.Plain(strings.Repeat(" ", f.width-x-ansi.StringWidth(t)-1)+t), nil)
		}
	}
	lines := components.RenderSegs(segs, tall)
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], f.width, "")
	}
	out := strings.Join(lines, "\n")
	if sz == components.Compact || tall {
		return out + "\n"
	}
	return out + "\n" + lipgloss.NewStyle().Foreground(theme.Pal.Surface).Render(strings.Repeat("─", f.width)) + "\n"
}

// tabLabel is a view's tab: its icon and name, with its badge; just the
// icon (and badge) when folded.
func (f *Frame) tabLabel(v int, sz components.Size) string {
	view := f.o.Views[v]
	badge := ""
	if view.Badge != nil {
		if root := f.Root(v); root != nil {
			if b := view.Badge(root); b != "" {
				badge = " " + b
			}
		}
	}
	if sz == components.Compact {
		return view.Icon + badge
	}
	return view.Icon + " " + view.Name + badge
}

// footer draws the status line and the key hints, which are buttons.
func (f *Frame) footer() (string, []touch.Zone) {
	var lines []string
	switch {
	case f.err != nil:
		lines = append(lines, ansi.Truncate(theme.Styles.Err.Render(icons.Current.Warning+" "+f.o.ErrText(f.err))+theme.Styles.Muted.Render("  r retry"), f.width, "…"))
	case f.status != "":
		lines = append(lines, ansi.Truncate(theme.Styles.Status.Render(icons.Current.Info+" "+f.status), f.width, "…"))
	}
	hintLines, zones := touch.Hints(f.Bindings(), f.width, f.showHelp, f.Tall(), f.pressed)
	for i := range zones {
		zones[i].Y += len(lines)
	}
	lines = append(lines, hintLines...)
	return lipgloss.NewStyle().MaxWidth(f.width).Render(strings.Join(lines, "\n")), zones
}

// Bindings are the key hints shown now: the top screen's keys, then back,
// refresh, help and quit (only the screen's while it takes input).
func (f *Frame) Bindings() []key.Binding {
	var b []key.Binding
	top := f.Top()
	if top != nil {
		b = append(b, top.Keys()...)
		// While a screen takes input, the frame's keys don't apply.
		if top.Capturing() {
			return b
		}
	}
	if len(f.stack) > 1 {
		b = append(b, f.keys.Back)
	}
	return append(b, f.keys.Refresh, f.keys.Help, f.keys.Quit)
}
