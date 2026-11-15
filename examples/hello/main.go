// Command hello is the smallest rexkit app, to start new apps from: two
// views as tabs, a list of cards you can tap, a detail screen with
// markdown, a form (n adds a note), and the frame's header, key hints and
// Rex tab status.
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
	"github.com/ionut-maxim/markdown"
	"github.com/ionut-maxim/markdown/teamd"

	"github.com/ionut-maxim/rexkit/components"
	"github.com/ionut-maxim/rexkit/form"
	"github.com/ionut-maxim/rexkit/icons"
	"github.com/ionut-maxim/rexkit/shell"
	"github.com/ionut-maxim/rexkit/theme"
)

type note struct{ title, sub, body string }

var notes = []note{
	{"Tap a card", "Focus, then open", "Tap a card to **focus** it, and tap it again to open it. Keys do the same: `j`/`k` and `enter`."},
	{"Tabs keep their place", "Each view has its own history", "Each view keeps its own history.\n\n- Open this note\n- Switch to *About* and back\n- You're still here"},
	{"Code is highlighted", "Chroma, in the app's palette", "```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```"},
	{"Pictures in markdown", "Headings, alerts and diagrams", "# Big heading\n\n> [!TIP]\n> In Rex, headings, alert icons and diagrams are drawn as pictures.\n\n```mermaid\nflowchart LR\n    A[List] --> B[Detail]\n    B --> A\n```"},
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

// detailScreen renders a note's markdown in the app's palette, with the
// markdown library. teamd sends its pictures (headings, alert icons,
// diagrams); the frame has already asked the terminal what it can do.
type detailScreen struct {
	term  *shell.Term
	n     note
	r     *markdown.Renderer
	p     *teamd.Pictures
	doc   *markdown.Doc
	width int
}

func newDetail(t *shell.Term, n note) shell.Screen {
	r := markdown.New(markdown.Options{Theme: components.MarkdownTheme(t.Dark), Caps: t.Caps()})
	return &detailScreen{term: t, n: n, r: r, p: teamd.New(r, teamd.Options{NoProbe: true}), doc: r.Parse(n.body)}
}

func (s *detailScreen) Init() tea.Cmd       { return nil }
func (s *detailScreen) Refresh() tea.Cmd    { return nil }
func (s *detailScreen) Title() string       { return s.n.title }
func (s *detailScreen) Capturing() bool     { return false }
func (s *detailScreen) Keys() []key.Binding { return nil }

// Close deletes the pictures when the screen goes.
func (s *detailScreen) Close() tea.Cmd { return s.p.Close() }

func (s *detailScreen) Update(msg tea.Msg) (shell.Screen, tea.Cmd) {
	cmd := s.p.Update(msg)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width - 4
		return s, tea.Batch(cmd, s.p.Show(s.width, s.doc))
	case shell.ThemeChanged:
		s.r.SetTheme(components.MarkdownTheme(s.term.Dark))
		s.r.SetCaps(s.term.Caps())
		return s, tea.Batch(cmd, s.p.Show(s.width, s.doc))
	}
	return s, cmd
}

func (s *detailScreen) View(width, _ int) string {
	md := s.r.Render(s.doc, width-4)
	return "\n" + lipgloss.NewStyle().PaddingLeft(2).Render(md)
}

// newNote is a form; submitting adds the note to the list.
func newNote(tall func() bool) shell.Screen {
	return form.New(form.Options{
		Title:  "New note",
		Submit: "Add",
		Fields: []form.Field{
			{Label: "Title", Required: true, Placeholder: "What it's about"},
			{Label: "Subtitle", Hint: "Optional, one line"},
		},
		OnSubmit: func(v []string) tea.Cmd {
			// A real app runs its job here and answers with form.Done.
			notes = append(notes, note{v[0], v[1], v[0]})
			return func() tea.Msg { return form.Done{Status: "Added " + v[0]} }
		},
		Tall: tall,
	})
}

func main() {
	term := &shell.Term{Dark: true}
	var frame *shell.Frame
	about := note{"About", "", "This is **rexkit**'s example app. Read `examples/hello/main.go`, then copy it to start a new app."}
	frame = shell.New(shell.Options{
		App:  "hello",
		Term: term,
		Views: []shell.View{
			{Name: "Notes", Icon: icons.Current.Issues, Root: func() shell.Screen { return newList(term) },
				Badge: func(root shell.Screen) string { return root.(*listScreen).badge() }},
			{Name: "About", Icon: icons.Current.Info, Root: func() shell.Screen { return newDetail(term, about) }},
		},
		Context: func() string { return "rexkit" },
		Keys: []shell.GlobalKey{{
			Binding: key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new note")),
			Run:     func() tea.Cmd { return shell.PushCmd(newNote(frame.Tall)) },
		}},
	})
	if _, err := tea.NewProgram(frame).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
