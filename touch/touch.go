// Package touch makes a terminal app easy to use with a finger or a
// mouse: tap zones, key hints drawn as buttons, and clicks and the wheel
// on Bubbles lists.
package touch

import (
	"bytes"
	"reflect"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ionut-maxim/rexkit/components"
	"github.com/ionut-maxim/rexkit/theme"
)

// Zone is a clickable span on one screen row; clicking it sends msg.
type Zone struct {
	Y, X0, X1 int // x1 exclusive
	H         int // rows from y; 0 means 1
	Msg       tea.Msg
}

// Hit is the message of the zone at x, y, if any.
func Hit(zones []Zone, x, y int) (tea.Msg, bool) {
	for _, z := range zones {
		if y >= z.Y && y < z.Y+max(z.H, 1) && x >= z.X0 && x < z.X1 {
			return z.Msg, true
		}
	}
	return nil, false
}

// KeyMsg builds the key press a binding's first key stands for, so a click
// on a key hint does exactly what the key does.
func KeyMsg(name string) tea.KeyPressMsg {
	switch name {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "pgup":
		return tea.KeyPressMsg{Code: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	}
	r := []rune(name)[0]
	return tea.KeyPressMsg{Code: r, Text: name}
}

// Hints renders key Hints as buttons, wrapped to width (one row unless
// all is set; then what doesn't fit becomes a "? more" button that opens
// the full help). Tall buttons take three lines a row.
func Hints(bindings []key.Binding, width int, all, tall bool, pressed tea.Msg) ([]string, []Zone) {
	const gap = 1
	moreBg := theme.Pal.SurfaceHi
	if pressed != nil && reflect.DeepEqual(pressed, tea.Msg(KeyMsg("?"))) {
		moreBg = theme.Pal.AccentSoft
	}
	more := components.Btn("?", "more", moreBg, tall)
	var shown []key.Binding
	for _, b := range bindings {
		if b.Enabled() && len(b.Keys()) > 0 {
			shown = append(shown, b)
		}
	}
	rowH := 1
	if tall {
		rowH = 3
	}
	var rows [][]components.Seg
	var zones []Zone
	var row []components.Seg
	x := 0
	for i, b := range shown {
		h := b.Help()
		bg := theme.Pal.SurfaceHi
		if pressed != nil && reflect.DeepEqual(pressed, tea.Msg(KeyMsg(b.Keys()[0]))) {
			bg = theme.Pal.AccentSoft
		}
		sg := components.Btn(h.Key, h.Desc, bg, tall)
		g := 0
		if x > 0 {
			g = gap
		}
		need := x + g + sg.W
		if !all && i < len(shown)-1 {
			need += gap + more.W
		}
		if need > width {
			if !all {
				if x+g+more.W <= width {
					if g > 0 {
						row = append(row, components.Plain(" "))
					}
					zones = append(zones, Zone{Y: len(rows) * rowH, H: rowH, X0: x + g, X1: x + g + more.W, Msg: KeyMsg("?")})
					row = append(row, more)
				}
				break
			}
			if x > 0 {
				rows = append(rows, row)
				row, x, g = nil, 0, 0
			}
		}
		if g > 0 {
			row = append(row, components.Plain(" "))
		}
		zones = append(zones, Zone{Y: len(rows) * rowH, H: rowH, X0: x + g, X1: x + g + sg.W, Msg: KeyMsg(b.Keys()[0])})
		row = append(row, sg)
		x += g + sg.W
	}
	if len(row) > 0 || len(rows) == 0 {
		rows = append(rows, row)
	}
	var lines []string
	for _, r := range rows {
		lines = append(lines, components.RenderSegs(r, tall)...)
	}
	return lines, zones
}

// ListRow returns the index (into VisibleItems) of the row at line y of
// l.View(), if y is on an item. It finds where the items start by drawing
// the first visible item and looking for it, so it follows whatever the
// list shows above the items (filter, status bar).
func ListRow(l list.Model, d list.ItemDelegate, y int) (int, bool) {
	items := l.VisibleItems()
	first := l.Paginator.Page * l.Paginator.PerPage
	if first >= len(items) {
		return 0, false
	}
	var buf bytes.Buffer
	d.Render(&buf, l, first, items[first])
	want := rowKey(strings.SplitN(buf.String(), "\n", 2)[0])
	top := -1
	for i, line := range strings.Split(l.View(), "\n") {
		if rowKey(line) == want {
			top = i
			break
		}
	}
	if top < 0 || y < top {
		return 0, false
	}
	step := d.Height() + d.Spacing()
	off := y - top
	if off%step >= d.Height() {
		return 0, false // the gap between rows
	}
	idx := first + off/step
	if idx >= min(first+l.Paginator.PerPage, len(items)) {
		return 0, false
	}
	return idx, true
}

// rowKey is a row's text without styling or the selection bar.
func rowKey(s string) string {
	return strings.TrimSpace(strings.TrimLeft(ansi.Strip(s), "│ "))
}

// RowClicks turns clicks on a list into selecting and opening: Click
// selects the clicked row; a second click on the selected row (or a quick
// double click) reports open.
type RowClicks struct {
	last    int
	lastAt  time.Time
	hasLast bool
}

func (r *RowClicks) Click(l *list.Model, idx int) (open bool) {
	cur := l.Index()
	now := time.Now()
	open = idx == cur || (r.hasLast && r.last == idx && now.Sub(r.lastAt) < 400*time.Millisecond)
	l.Select(idx)
	r.last, r.lastAt, r.hasLast = idx, now, true
	return open
}

// Wheel moves a list's cursor with the mouse Wheel.
func Wheel(l *list.Model, msg tea.MouseWheelMsg) {
	switch msg.Mouse().Button {
	case tea.MouseWheelUp:
		l.CursorUp()
	case tea.MouseWheelDown:
		l.CursorDown()
	}
}
