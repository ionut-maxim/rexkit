// Command hello is the smallest rexkit app, to start new apps from: two
// views as tabs, a list of cards you can tap, a detail screen with
// markdown, and the frame's header, key hints and Rex tab status.
//
//	go run github.com/ionut-maxim/rexkit/examples/hello
package main

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ionut-maxim/rexkit/components"
	"github.com/ionut-maxim/rexkit/icons"
	"github.com/ionut-maxim/rexkit/markdown"
	"github.com/ionut-maxim/rexkit/shell"
	"github.com/ionut-maxim/rexkit/theme"
)

type note struct{ title, sub, body string }

var notes = []note{
	{"Tap a card", "Focus, then open", "Tap a card to **focus** it, and tap it again to open it. Keys do the same: `j`/`k` and `enter`."},
	{"Tabs keep their place", "Each view has its own history", "Each view keeps its own history.\n\n- Open this note\n- Switch to *About* and back\n- You're still here"},
	{"Code is highlighted", "Chroma, in the app's palette", "```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```"},
}

// listScreen shows the notes as cards. Taps arrive with body-relative
// coordinates; rows maps a line to its note.
type listScreen struct {
	term   *shell.Term
	cursor int
	rows   map[int]int
	open   key.Binding
}

func newList(t *shell.Term) shell.Screen {
	return &listScreen{term: t, open: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open"))}
}

func (s *listScreen) Init() tea.Cmd           { return nil }
func (s *listScreen) Refresh() tea.Cmd        { return nil }
func (s *listScreen) Title() string           { return "Notes" }
func (s *listScreen) Capturing() bool         { return false }
func (s *listScreen) Keys() []key.Binding     { return []key.Binding{s.open} }
func (s *listScreen) openCmd() tea.Cmd        { return shell.PushCmd(newDetail(s.term, notes[s.cursor])) }
func (s *listScreen) move(d int)              { s.cursor = max(0, min(len(notes)-1, s.cursor+d)) }
func (s *listScreen) selected(i int) bool     { return i == s.cursor }
func (s *listScreen) badge() string           { return fmt.Sprint(len(notes)) }
func (s *listScreen) rowOf(y int) (int, bool) { i, ok := s.rows[y]; return i, ok }

func (s *listScreen) Update(msg tea.Msg) (shell.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, s.open):
			return s, s.openCmd()
		case msg.String() == "j" || msg.String() == "down":
			s.move(1)
		case msg.String() == "k" || msg.String() == "up":
			s.move(-1)
		}
	case tea.MouseClickMsg:
		if i, ok := s.rowOf(msg.Mouse().Y); ok {
			if s.selected(i) {
				return s, s.openCmd()
			}
			s.cursor = i
		}
	}
	return s, nil
}

func (s *listScreen) View(width, height int) string {
	s.rows = map[int]int{}
	lines := []string{"", components.SectionTitle(icons.Current.Issues, theme.Pal.Accent, "Notes", width)}
	for i, n := range notes {
		edge, title := theme.Styles.Muted.Render("│ "), theme.Styles.Subtle.Render(n.title)
		if s.selected(i) {
			edge, title = theme.Styles.Accent.Render("▌ "), theme.Styles.Title.Render(n.title)
		}
		card := []string{
			components.Spread(edge+title, theme.Styles.Muted.Render(icons.Current.Chevron)+" ", width),
			ansi.Truncate(edge+theme.Styles.Muted.Render(n.sub), width, "…"),
		}
		for _, l := range card {
			if s.selected(i) {
				l = components.FillBg(l, width, theme.Pal.Surface)
			}
			s.rows[len(lines)] = i
			lines = append(lines, l)
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// detailScreen renders a note's markdown in the app's palette.
type detailScreen struct {
	term *shell.Term
	n    note
}

func newDetail(t *shell.Term, n note) shell.Screen { return &detailScreen{term: t, n: n} }

func (s *detailScreen) Init() tea.Cmd                          { return nil }
func (s *detailScreen) Refresh() tea.Cmd                       { return nil }
func (s *detailScreen) Title() string                          { return s.n.title }
func (s *detailScreen) Capturing() bool                        { return false }
func (s *detailScreen) Keys() []key.Binding                    { return nil }
func (s *detailScreen) Update(tea.Msg) (shell.Screen, tea.Cmd) { return s, nil }

func (s *detailScreen) View(width, _ int) string {
	md := markdown.RenderTheme(s.n.body, width-4, components.MarkdownTheme(s.term.Dark))
	return "\n" + lipgloss.NewStyle().PaddingLeft(2).Render(md)
}

func main() {
	term := &shell.Term{Dark: true}
	about := note{"About", "", "This is **rexkit**'s example app. Read `examples/hello/main.go`, then copy it to start a new app."}
	frame := shell.New(shell.Options{
		App:  "hello",
		Term: term,
		Views: []shell.View{
			{Name: "Notes", Icon: icons.Current.Issues, Root: func() shell.Screen { return newList(term) },
				Badge: func(root shell.Screen) string { return root.(*listScreen).badge() }},
			{Name: "About", Icon: icons.Current.Info, Root: func() shell.Screen { return newDetail(term, about) }},
		},
		Context: func() string { return "rexkit" },
	})
	if _, err := tea.NewProgram(frame).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
