package shell

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ionut-maxim/rexkit/icons"
	"github.com/ionut-maxim/rexkit/theme"
	"github.com/ionut-maxim/rexkit/touch"
)

// With the action bar, the header is split in two jobs:
//
//   - getting around: a big Back button when there's somewhere to go
//     back to, then the views as equal tabs, the current one tinted, each
//     with its badge; buttons like the action bar's (images in kitty);
//   - where you are: the path (root › … › this screen) on one line, the
//     earlier steps tappable, this screen in bold, and the context (a
//     team) on the right when there's room.
//
// In portrait the path is a line under the tabs; in landscape, where
// height is scarce, the buttons are two lines tall and the path sits to
// their right.

// navRows is the nav buttons' height.
func navRows(mode barMode) int {
	if mode == barRail {
		return 2
	}
	return 3
}

// navHeight is the header's height with the action bar.
func (f *Frame) navHeight(mode barMode) int {
	if mode == barBottom {
		return navRows(mode) + 1
	}
	return navRows(mode)
}

// navButtons are Back (when there's a screen to go back to) and a tab
// per view.
func (f *Frame) navButtons(mode barMode) []barButton {
	if mode == barHints || len(f.o.Views) == 0 || !f.started {
		return nil
	}
	rows := navRows(mode)
	compact := f.width < 64
	var out []barButton
	x := 1
	if len(f.stack) > 1 {
		w, label := 10, "Back"
		if compact {
			w, label = 6, ""
		}
		out = append(out, barButton{act: barAction{key: "‹", label: label, msg: Pop{}}, x: x, w: w, h: rows})
		x += w + 1
	}
	n := len(f.o.Views)
	tabW := (f.width - 1 - x - (n - 1)) / n
	if mode == barRail {
		tabW = min(tabW, 18) // leave the path its room
	}
	tabW = max(tabW, 4)
	for v, view := range f.o.Views {
		label := view.Name
		if b := f.badge(v); b != "" {
			label += " " + b
		}
		key := view.Icon
		if compact && ansi.StringWidth(key+" "+label) > tabW-2 {
			key = ""
		}
		out = append(out, barButton{act: barAction{key: key, label: label, msg: SwitchView{v}}, x: x, w: tabW, h: rows, primary: v == f.cur})
		x += tabW + 1
	}
	return out
}

// badge is view v's badge, if its first screen has one.
func (f *Frame) badge(v int) string {
	view := f.o.Views[v]
	if view.Badge == nil {
		return ""
	}
	if root := f.Root(v); root != nil {
		return view.Badge(root)
	}
	return ""
}

// navLines draws the header with the action bar; its zones are from the
// top left of the screen.
func (f *Frame) navLines(mode barMode) []string {
	rows := navRows(mode)
	buttons := f.navButtons(mode)
	lines := make([]string, rows)
	x := 0
	for _, b := range buttons {
		pad := strings.Repeat(" ", max(b.x-x, 0))
		for i, l := range f.drawButton(b) {
			lines[i] += pad + l
		}
		x = b.x + b.w
		f.zones = append(f.zones, touch.Zone{Y: 0, H: rows, X0: b.x, X1: b.x + b.w, Msg: b.act.msg})
	}
	switch mode {
	case barBottom:
		path, zones := f.pathLine(f.width-2, 1, rows)
		f.zones = append(f.zones, zones...)
		lines = append(lines, " "+path)
	case barRail:
		// To the right of the tabs, on their first line.
		start := x + 2
		path, zones := f.pathLine(f.width-start-1, start, 0)
		f.zones = append(f.zones, zones...)
		lines[0] += strings.Repeat(" ", max(start-x, 0)) + path
	}
	if len(buttons) == 0 {
		// Before the views start: the app's name and the context.
		for i := range lines {
			lines[i] = ""
		}
	}
	return lines
}

// pathLine is where you are, at most width cells, starting at column x0
// on line y (for its tap zones): the views' first screen, then each
// screen opened from it, this one in bold. The oldest steps fold into
// "…" when it doesn't fit; the context goes on the right when there's
// room.
func (f *Frame) pathLine(width, x0, y int) (string, []touch.Zone) {
	type step struct {
		text  string
		depth int
	}
	var steps []step
	for i, s := range f.stack {
		steps = append(steps, step{s.Title(), i + 1})
	}
	if len(steps) == 0 {
		return "", nil
	}
	sep := " " + theme.Styles.Muted.Render(icons.Current.Sep) + " "
	sepW := ansi.StringWidth(sep)
	stepText := func(s step, last bool) string {
		if last {
			return theme.Styles.Title.Render(s.text)
		}
		st := theme.Styles.Subtle
		if f.isPressed(PopTo{s.depth}) {
			st = theme.Styles.Accent
		}
		return st.Render(s.text)
	}
	total := func(ss []step, folded bool) int {
		w := 0
		if folded {
			w += 1 + sepW
		}
		for i, s := range ss {
			if i > 0 {
				w += sepW
			}
			w += ansi.StringWidth(s.text)
		}
		return w
	}
	folded := false
	for len(steps) > 1 && total(steps, folded) > width {
		steps, folded = steps[1:], true
	}
	var out strings.Builder
	add := func(text string, _ tea.Msg) { out.WriteString(text) }
	if folded {
		add(theme.Styles.Muted.Render("…"), PopTo{max(steps[0].depth-1, 1)})
		add(sep, nil)
	}
	for i, s := range steps {
		if i > 0 {
			add(sep, nil)
		}
		last := i == len(steps)-1
		var msg tea.Msg
		if !last {
			msg = PopTo{s.depth}
		}
		add(stepText(s, last), msg)
	}
	// The whole line is one target (a single line is hard to hit word by
	// word): ▾ says it opens.
	var zones []touch.Zone
	if len(f.stack) > 1 {
		out.WriteString(" " + theme.Styles.Muted.Render("▾"))
	}
	text := ansi.Truncate(out.String(), max(width, 1), "…")
	if len(f.stack) > 1 {
		zones = append(zones, touch.Zone{Y: y, H: 1, X0: x0, X1: x0 + max(width, ansi.StringWidth(text)), Msg: pathMsg{}})
	}
	if f.o.Context != nil {
		ctx := theme.Styles.Muted.Render(icons.Current.Team + " " + f.o.Context())
		if gap := width - ansi.StringWidth(text) - ansi.StringWidth(ctx); gap >= 3 {
			text += strings.Repeat(" ", gap) + ctx
		}
	}
	return text, zones
}

// pathMsg is a tap on the path: back one screen, or, with more to go
// back to, the Go to sheet.
type pathMsg struct{}

// goTo is the tap on the path.
func (f *Frame) goTo() tea.Cmd {
	if len(f.stack) <= 2 {
		return func() tea.Msg { return Pop{} }
	}
	var acts []barAction
	for i := len(f.stack) - 2; i >= 0; i-- {
		acts = append(acts, barAction{label: f.stack[i].Title(), msg: PopTo{i + 1}})
	}
	sheet := newMoreSheet(acts)
	sheet.title = "Go to"
	return func() tea.Msg { return Push{sheet} }
}
