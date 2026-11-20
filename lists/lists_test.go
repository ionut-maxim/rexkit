package lists

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ionut-maxim/rexkit/components"
	"github.com/ionut-maxim/rexkit/icons"
)

func init() { icons.Current = icons.Unicode }

type row struct{ id, title string }

func (r row) FilterValue() string { return r.id + " " + r.title }
func (r row) Marker() string      { return "•" }
func (r row) Title() string       { return r.title }
func (r row) Description() string { return r.id }

func rows(n int) []components.Row {
	out := make([]components.Row, n)
	for i := range out {
		out[i] = row{fmt.Sprintf("NIN-%d", i+1), fmt.Sprintf("Issue number %d", i+1)}
	}
	return out
}

func key(s string) tea.KeyPressMsg {
	switch s {
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	}
	r := []rune(s)[0]
	return tea.KeyPressMsg{Code: r, Text: s}
}

func wheel(down bool) tea.MouseWheelMsg {
	if down {
		return tea.MouseWheelMsg{Button: tea.MouseWheelDown}
	}
	return tea.MouseWheelMsg{Button: tea.MouseWheelUp}
}

func view(r *Rows) string { return ansi.Strip(r.View()) }

func TestScrollPlaceFollowsUnlessFree(t *testing.T) {
	var s Scroll
	if off := s.Place(30, 2, 10, 60); off != 22 {
		t.Fatalf("offset = %d, want 22 (selection at the bottom)", off)
	}
	s.Wheel(wheel(false))
	if s.Offset() != 22-WheelLines || !s.Free() {
		t.Fatalf("offset = %d free = %v", s.Offset(), s.Free())
	}
	if off := s.Place(30, 2, 10, 60); off != 22-WheelLines {
		t.Errorf("a free scroll was pulled back: %d", off)
	}
	s.Follow()
	if off := s.Place(30, 2, 10, 60); off != 22 {
		t.Errorf("Follow didn't bring it back: %d", off)
	}
	for range 30 {
		s.Wheel(wheel(true))
	}
	if s.Offset() != 50 {
		t.Errorf("scrolled past the end: %d", s.Offset())
	}
	for range 30 {
		s.Wheel(wheel(false))
	}
	if s.Offset() != 0 {
		t.Errorf("scrolled past the top: %d", s.Offset())
	}
}

func TestRowsWheelScrollsWithoutSelecting(t *testing.T) {
	r := NewRows("issue", "issues")
	r.SetItems(rows(20))
	r.SetSize(60, 14)
	if out := view(r); !strings.Contains(out, "20 issues") || !strings.Contains(out, "Issue number 1") {
		t.Fatalf("first view:\n%s", out)
	}
	for range 3 {
		r.Update(wheel(true))
	}
	out := view(r)
	if r.Index() != 0 {
		t.Errorf("index = %d after scrolling", r.Index())
	}
	if strings.Contains(out, "Issue number 1\n") || !strings.Contains(out, "Issue number 4") {
		t.Errorf("didn't scroll:\n%s", out)
	}
	if !strings.Contains(out, "20 issues") {
		t.Errorf("the status line scrolled away:\n%s", out)
	}
	r.Update(key("j"))
	if r.Index() != 1 || !strings.Contains(view(r), "Issue number 2") {
		t.Errorf("a key didn't bring the selection back: index %d\n%s", r.Index(), view(r))
	}
}

func TestRowsClick(t *testing.T) {
	r := NewRows("issue", "issues")
	r.SetItems(rows(20))
	r.SetSize(60, 14)
	_ = view(r)
	for range 2 {
		r.Update(wheel(true))
	}
	_ = view(r) // scrolled by 6 lines: row 3 (NIN-3) starts at line 2
	if _, ok := r.RowAt(0); ok {
		t.Error("the status line is a row")
	}
	open, ok := r.Click(2)
	if !ok || open || r.Index() != 2 {
		t.Fatalf("click: open=%v ok=%v index=%d", open, ok, r.Index())
	}
	if _, ok := r.Click(4); ok {
		t.Error("the gap between rows took a click")
	}
	before := r.scroll.Offset()
	_ = view(r)
	if r.scroll.Offset() != before {
		t.Errorf("a tap moved the page: %d → %d", before, r.scroll.Offset())
	}
	if open, ok := r.Click(3); !ok || !open {
		t.Errorf("a tap on the selected row: open=%v ok=%v", open, ok)
	}
}

func TestRowsFilter(t *testing.T) {
	r := NewRows("issue", "issues")
	r.SetItems(rows(20))
	r.SetSize(60, 30)
	r.Update(key("/"))
	if !r.Filtering() {
		t.Fatal("/ didn't start the filter")
	}
	for _, c := range "number 12" {
		r.Update(key(string(c)))
	}
	out := view(r)
	if !strings.Contains(out, "number 12▏") || !strings.Contains(out, "of 20 issues") {
		t.Errorf("status line:\n%s", out)
	}
	if it, ok := r.Selected(); !ok || it.Title() != "Issue number 12" {
		t.Errorf("selected = %v", it)
	}
	// j is text while typing.
	r.Update(key("j"))
	if r.FilterText() != "number 12j" {
		t.Errorf("filter = %q", r.FilterText())
	}
	r.Update(key("backspace"))
	r.Update(key("enter"))
	if r.Filtering() || !r.IsFiltered() {
		t.Errorf("enter: filtering=%v filtered=%v", r.Filtering(), r.IsFiltered())
	}
	// Underlined matches in the title (the id isn't in it).
	if !strings.Contains(r.View(), "\x1b[") {
		t.Error("no styling")
	}
	if used := r.Update(key("esc")); !used || r.IsFiltered() || len(r.Visible()) != 20 {
		t.Errorf("esc: used=%v filtered=%v visible=%d", used, r.IsFiltered(), len(r.Visible()))
	}
	// Without a filter, esc is left for the app (back).
	if used := r.Update(key("esc")); used {
		t.Error("esc used with no filter")
	}
}

func TestRowsFilterOff(t *testing.T) {
	r := NewRows("repo", "repos")
	r.SetFilterEnabled(false)
	r.SetItems(rows(3))
	if used := r.Update(key("/")); used || r.Filtering() {
		t.Error("/ started a filter while it's off")
	}
}

func TestRowsSetItemsKeepsSelection(t *testing.T) {
	r := NewRows("issue", "issues")
	r.SetItems(rows(5))
	r.Select(3)
	items := rows(6)
	items[0], items[5] = items[5], items[0]
	r.SetItems(items)
	if it, _ := r.Selected(); it.Title() != "Issue number 4" {
		t.Errorf("selected = %q after SetItems", it.Title())
	}
	if len(r.Items()) != 6 {
		t.Errorf("items = %d", len(r.Items()))
	}
}

func TestRowsTinyAndEmpty(t *testing.T) {
	r := NewRows("issue", "issues")
	for _, size := range [][2]int{{0, 0}, {1, 1}, {10, 3}} {
		r.SetSize(size[0], size[1])
		_ = r.View()
		r.Update(key("j"))
		r.Update(wheel(true))
		if _, ok := r.Click(2); ok {
			t.Error("click on an empty list")
		}
	}
	r.SetItems(rows(1))
	if !strings.Contains(view(r), "1 issue") {
		t.Errorf("singular:\n%s", view(r))
	}
}

func TestTitleMatches(t *testing.T) {
	if got := titleMatches("Issue", "NIN-1 Issue", []int{0, 6, 7}); fmt.Sprint(got) != "[0 1]" {
		t.Errorf("got %v", got)
	}
	if got := titleMatches("Other", "NIN-1 Issue", []int{6}); got != nil {
		t.Errorf("got %v", got)
	}
}

// viewAll is View the old way: every row drawn, then the window cut out.
func viewAll(r *Rows) string {
	w, h := max(r.width, 1), max(r.height, head+2)
	var lines []string
	for i, idx := range r.visible {
		row := components.RenderRow(r.items[idx], i == r.cursor, len(r.filter) > 0, r.matches[idx], w)
		lines = append(lines, strings.Split(row, "\n")...)
		if i+1 < len(r.visible) {
			lines = append(lines, "")
		}
	}
	bodyH := h - head
	off := r.scroll.Offset()
	out := []string{ansi.Truncate(r.status(), w, "…"), ""}
	for y := 0; y < bodyH && off+y < len(lines); y++ {
		out = append(out, lines[off+y])
	}
	return strings.Join(out, "\n")
}

// Drawing only the rows in view gives the same screen as drawing all.
func TestRowsViewDrawsOnlyWhatFits(t *testing.T) {
	for _, h := range []int{3, 4, 5, 14, 15, 16, 61} {
		r := NewRows("issue", "issues")
		r.SetItems(rows(20))
		r.SetSize(60, h)
		steps := []tea.Msg{key("j"), key("j"), wheel(true), wheel(true), wheel(true), key("G"), wheel(false), key("k"), wheel(false), wheel(false), key("g")}
		for i, m := range steps {
			r.Update(m)
			got := r.View()
			if want := viewAll(r); got != want {
				t.Fatalf("h=%d step %d:\ngot:\n%s\nwant:\n%s", h, i, got, want)
			}
		}
	}
}

func BenchmarkRowsView(b *testing.B) {
	for _, n := range []int{50, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			r := NewRows("issue", "issues")
			r.SetItems(rows(n))
			r.SetSize(83, 45)
			b.ReportAllocs()
			for i := range b.N {
				if i%40 < 20 {
					r.Update(key("j"))
				} else {
					r.Update(key("k"))
				}
				_ = r.View()
			}
		})
	}
}
