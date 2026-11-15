package lists

import (
	"fmt"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"

	"github.com/ionut-maxim/rexkit/components"
	"github.com/ionut-maxim/rexkit/icons"
	"github.com/ionut-maxim/rexkit/theme"
)

// Rows is a scrolling list of components.Row, drawn as RowDelegate draws
// them: two lines and a gap each. Over the rows, a status line counts them
// and shows the filter. Keys move the selection (up/down, j/k, pgup/pgdn,
// home/end, g/G); / starts a filter, enter keeps it, esc drops it; the
// wheel scrolls without selecting; Click maps a tap to a row.
type Rows struct {
	items   []components.Row
	visible []int         // indexes into items, in display order
	matches map[int][]int // rune matches in each item's title, by item
	cursor  int           // into visible
	scroll  Scroll

	one, many string // what the rows are called, for the status line
	filterOn  bool
	filter    []rune
	editing   bool // typing the filter

	width, height int
}

// rowStep is a row's two lines and its gap.
const rowStep = 3

// head is the status line and the blank line under it.
const head = 2

// NewRows makes an empty list; one and many name the rows ("issue",
// "issues") on the status line.
func NewRows(one, many string) *Rows {
	return &Rows{one: one, many: many, filterOn: true}
}

// SetFilterEnabled turns the / filter on or off (on by default).
func (r *Rows) SetFilterEnabled(on bool) {
	r.filterOn = on
	if !on {
		r.ResetFilter()
	}
}

// SetItems replaces the rows, keeping the selection on the same row (by
// FilterValue) when it's still there, and the filter.
func (r *Rows) SetItems(items []components.Row) {
	var keep string
	if it, ok := r.Selected(); ok {
		keep = it.FilterValue()
	}
	r.items = items
	r.apply()
	r.cursor = 0
	for i, idx := range r.visible {
		if keep != "" && r.items[idx].FilterValue() == keep {
			r.cursor = i
			break
		}
	}
}

// Items are all the rows, filtered or not.
func (r *Rows) Items() []components.Row { return r.items }

// Visible are the rows shown, in order.
func (r *Rows) Visible() []components.Row {
	out := make([]components.Row, len(r.visible))
	for i, idx := range r.visible {
		out[i] = r.items[idx]
	}
	return out
}

// Index is the selected row's place among the visible rows.
func (r *Rows) Index() int { return r.cursor }

// Selected is the selected row, if any is visible.
func (r *Rows) Selected() (components.Row, bool) {
	if r.cursor < 0 || r.cursor >= len(r.visible) {
		return nil, false
	}
	return r.items[r.visible[r.cursor]], true
}

// Select selects the i-th visible row and brings it into view.
func (r *Rows) Select(i int) {
	r.cursor = max(0, min(i, len(r.visible)-1))
	r.scroll.Follow()
}

// Filtering says the filter is being typed: the list takes text input.
func (r *Rows) Filtering() bool { return r.editing }

// IsFiltered says a filter is applied (being typed or kept).
func (r *Rows) IsFiltered() bool { return len(r.filter) > 0 || r.editing }

// FilterText is the filter as typed.
func (r *Rows) FilterText() string { return string(r.filter) }

// ResetFilter drops the filter and shows every row.
func (r *Rows) ResetFilter() {
	r.filter, r.editing = nil, false
	r.apply()
	r.cursor = 0
	r.scroll.Reset()
}

// SetSize sets the view's size.
func (r *Rows) SetSize(width, height int) { r.width, r.height = width, height }

// apply works out the visible rows for the filter: every row in order, or
// the fuzzy matches on FilterValue, best first.
func (r *Rows) apply() {
	r.matches = nil
	r.visible = r.visible[:0]
	if len(r.filter) == 0 {
		for i := range r.items {
			r.visible = append(r.visible, i)
		}
		return
	}
	targets := make([]string, len(r.items))
	for i, it := range r.items {
		targets[i] = it.FilterValue()
	}
	r.matches = map[int][]int{}
	for _, m := range fuzzy.Find(string(r.filter), targets) {
		r.visible = append(r.visible, m.Index)
		r.matches[m.Index] = titleMatches(r.items[m.Index].Title(), targets[m.Index], m.MatchedIndexes)
	}
	r.cursor = max(0, min(r.cursor, len(r.visible)-1))
}

// titleMatches maps matched byte offsets in the filter value to rune
// indexes in the title, when the title is the value or ends it (an id
// before a title, say). Otherwise nothing is underlined.
func titleMatches(title, value string, matched []int) []int {
	start := strings.LastIndex(value, title)
	if start < 0 || start+len(title) != len(value) {
		return nil
	}
	var out []int
	for _, b := range matched {
		if b < start {
			continue
		}
		out = append(out, len([]rune(value[start:b])))
	}
	return out
}

// move moves the selection by d rows and brings it into view.
func (r *Rows) move(d int) {
	r.Select(r.cursor + d)
}

// page is how many rows fit in the view.
func (r *Rows) page() int { return max((r.height-head)/rowStep, 1) }

// Update handles keys and the wheel. It returns true when it used the
// message.
func (r *Rows) Update(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		r.scroll.Wheel(msg)
		return true
	case tea.PasteMsg:
		if r.editing {
			r.typeFilter([]rune(msg.Content))
			return true
		}
	case tea.KeyPressMsg:
		if r.editing {
			return r.editKey(msg)
		}
		switch msg.String() {
		case "up", "k":
			r.move(-1)
		case "down", "j":
			r.move(1)
		case "pgup", "ctrl+u":
			r.move(-r.page())
		case "pgdown", "ctrl+d":
			r.move(r.page())
		case "home", "g":
			r.Select(0)
		case "end", "G":
			r.Select(len(r.visible) - 1)
		case "/":
			if !r.filterOn {
				return false
			}
			r.editing = true
		case "esc":
			if len(r.filter) == 0 {
				return false
			}
			r.ResetFilter()
		default:
			return false
		}
		return true
	}
	return false
}

// editKey handles a key while the filter is typed.
func (r *Rows) editKey(msg tea.KeyPressMsg) bool {
	switch msg.String() {
	case "esc":
		r.ResetFilter()
	case "enter":
		r.editing = false
	case "backspace":
		if len(r.filter) > 0 {
			r.filter = r.filter[:len(r.filter)-1]
			r.refilter()
		}
	case "up", "down":
		r.move(map[string]int{"up": -1, "down": 1}[msg.String()])
	default:
		if t := msg.Text; t != "" {
			r.typeFilter([]rune(t))
		}
	}
	return true
}

func (r *Rows) typeFilter(rs []rune) {
	for _, c := range rs {
		if unicode.IsPrint(c) {
			r.filter = append(r.filter, c)
		}
	}
	r.refilter()
}

func (r *Rows) refilter() {
	r.apply()
	r.cursor = 0
	r.scroll.Reset()
}

// RowAt is the visible row at line y of the last View, if y is on a row
// (not its gap or the status line).
func (r *Rows) RowAt(y int) (int, bool) {
	line := y + r.scroll.Offset() - head
	if y < head || line < 0 || line%rowStep >= 2 {
		return 0, false
	}
	i := line / rowStep
	if i >= len(r.visible) {
		return 0, false
	}
	return i, true
}

// Click handles a tap at line y of the last View: a row that isn't
// selected gets selected; the selected row reports open. The page stays
// where it is.
func (r *Rows) Click(y int) (open, ok bool) {
	i, ok := r.RowAt(y)
	if !ok {
		return false, false
	}
	if i == r.cursor {
		return true, true
	}
	r.cursor = i
	return false, true
}

// status is the line over the rows: the count, and the filter.
func (r *Rows) status() string {
	name := r.many
	if len(r.items) == 1 {
		name = r.one
	}
	count := fmt.Sprintf("%d %s", len(r.items), name)
	if !r.IsFiltered() {
		return theme.Styles.Muted.Render("  " + count)
	}
	cursor := ""
	if r.editing {
		cursor = theme.Styles.Accent.Render("▏")
	}
	return "  " + theme.Styles.Accent.Render(icons.Current.Search+" ") + string(r.filter) + cursor +
		theme.Styles.Muted.Render(fmt.Sprintf("  %d of %s", len(r.visible), count))
}

// View draws the status line and the rows that fit.
func (r *Rows) View() string {
	w, h := max(r.width, 1), max(r.height, head+2)
	var lines []string
	for i, idx := range r.visible {
		it := r.items[idx]
		row := components.RenderRow(it, i == r.cursor, len(r.filter) > 0, r.matches[idx], w)
		lines = append(lines, strings.Split(row, "\n")...)
		if i+1 < len(r.visible) {
			lines = append(lines, "")
		}
	}
	if len(r.visible) == 0 && r.IsFiltered() {
		lines = append(lines, theme.Styles.Faint.Render("  Nothing matches."))
	}
	bodyH := h - head
	off := r.scroll.Place(r.cursor*rowStep, 2, bodyH, len(lines))
	out := []string{ansi.Truncate(r.status(), w, "…"), ""}
	for y := 0; y < bodyH && off+y < len(lines); y++ {
		out = append(out, lines[off+y])
	}
	return strings.Join(out, "\n")
}
