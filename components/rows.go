package components

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ionut-maxim/rexkit/icons"
	"github.com/ionut-maxim/rexkit/theme"
)

// Row is a list item drawn by rowDelegate: a coloured marker, a plain
// title (filter matches are highlighted in it) and a plain description.
// Keeping colour out of the title matters: the list highlights matches by
// rune index, which breaks escape codes inside the string.
type Row interface {
	list.Item
	Marker() string
	Title() string
	Description() string
}

// RowDelegate draws Rows for a Bubbles list: two lines and a gap, with a
// selection bar.
type RowDelegate struct{}

var rowStyles = struct {
	title, selTitle, match lipgloss.Style
}{}

func (RowDelegate) Height() int                         { return 2 }
func (RowDelegate) Spacing() int                        { return 1 }
func (RowDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

// Render draws a row as a card you can tap: an accent edge when selected,
// the marker and title, a chevron at the right (tap to open), and the
// details below.
func (RowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	r, ok := item.(Row)
	if !ok {
		return
	}
	var matches []int
	if m.IsFiltered() {
		matches = m.MatchesForItem(index)
	}
	fmt.Fprint(w, RenderRow(r, index == m.Index(), m.IsFiltered(), matches, m.Width()))
}

// RenderRow draws a Row as RowDelegate does, two lines: the marker and
// title (filter matches, rune indexes into the title, underlined when
// filtered), a chevron at the right, and the details below.
func RenderRow(r Row, selected, filtered bool, matches []int, width int) string {
	edge := "  "
	titleStyle := lipgloss.NewStyle().Foreground(theme.Pal.Text)
	chev := theme.Styles.Muted.Render(icons.Current.Chevron)
	if selected {
		edge = theme.Styles.Accent.Render("▌ ")
		titleStyle = titleStyle.Bold(true)
		chev = theme.Styles.Accent.Render(icons.Current.Chevron)
	}
	title := r.Title()
	if filtered {
		match := titleStyle.Underline(true).Foreground(theme.Pal.Accent)
		title = lipgloss.StyleRunes(title, matches, match, titleStyle)
	} else {
		title = titleStyle.Render(title)
	}
	line1 := Spread(edge+r.Marker()+" "+title, chev+" ", width)
	line2 := ansi.Truncate(edge+"  "+r.Description(), width, "…")
	return line1 + "\n" + line2
}
