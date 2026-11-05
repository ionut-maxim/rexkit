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
	selected := index == m.Index()
	edge := "  "
	titleStyle := lipgloss.NewStyle().Foreground(theme.Pal.Text)
	chev := theme.Styles.Muted.Render(icons.Current.Chevron)
	if selected {
		edge = theme.Styles.Accent.Render("▌ ")
		titleStyle = titleStyle.Bold(true)
		chev = theme.Styles.Accent.Render(icons.Current.Chevron)
	}
	title := r.Title()
	if m.IsFiltered() {
		match := titleStyle.Underline(true).Foreground(theme.Pal.Accent)
		title = lipgloss.StyleRunes(title, m.MatchesForItem(index), match, titleStyle)
	} else {
		title = titleStyle.Render(title)
	}
	width := m.Width()
	line1 := Spread(edge+r.Marker()+" "+title, chev+" ", width)
	line2 := ansi.Truncate(edge+"  "+r.Description(), width, "…")
	fmt.Fprintf(w, "%s\n%s", line1, line2)
}
