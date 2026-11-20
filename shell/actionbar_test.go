package shell

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ionut-maxim/rexkit/touch"
)

// actScreen has three actions and records the keys it gets.
type actScreen struct {
	keys  []string
	width int
}

func (s *actScreen) Init() tea.Cmd { return nil }
func (s *actScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyPressMsg:
		s.keys = append(s.keys, m.String())
	case tea.WindowSizeMsg:
		s.width = m.Width
	}
	return s, nil
}
func (s *actScreen) View(w, h int) string { return "the body" }
func (s *actScreen) Title() string        { return "list" }
func (s *actScreen) Capturing() bool      { return false }
func (s *actScreen) Refresh() tea.Cmd     { return nil }
func (s *actScreen) Keys() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "work on it")),
		key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "status")),
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "assignee")),
		key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "label")),
	}
}

func barFrame(t *testing.T, w, h int) (*Frame, *actScreen) {
	t.Helper()
	s := &actScreen{}
	f := New(Options{ActionBar: true, Views: []View{{Name: "List", Root: func() Screen { return s }}}})
	run(f, tea.WindowSizeMsg{Width: w, Height: h})
	run(f, f.Start())
	return f, s
}

// tap presses and releases the zone whose message is msg.
func tap(t *testing.T, f *Frame, want tea.Msg) {
	t.Helper()
	_ = f.View()
	for _, z := range f.zones {
		if z.Msg == want {
			run(f, tea.MouseClickMsg{X: z.X0 + 1, Y: z.Y + z.H/2, Button: tea.MouseLeft})
			run(f, releaseMsg{f.pressed})
			return
		}
	}
	t.Fatalf("no zone for %v in %+v", want, f.zones)
}

func screen(f *Frame) string { return ansi.Strip(f.View().Content) }

func TestActionBarFollowsTheShape(t *testing.T) {
	for _, tc := range []struct {
		w, h int
		mode barMode
		want []string
	}{
		{50, 40, barBottom, []string{"⏎ Open", "Work on it", "⋯ More"}},
		{83, 40, barBottom, []string{"⏎ Open", "w Work on it", "s Status", "a Assignee", "⋯ More"}},
		{150, 26, barRail, []string{"⏎ Open", "l Label", "⋯ More"}},
		{50, 14, barHints, []string{"enter open"}},
	} {
		f, s := barFrame(t, tc.w, tc.h)
		if got := f.barMode(); got != tc.mode {
			t.Errorf("%dx%d: mode %v, want %v", tc.w, tc.h, got, tc.mode)
		}
		out := screen(f)
		for _, w := range tc.want {
			if !strings.Contains(out, w) {
				t.Errorf("%dx%d lacks %q:\n%s", tc.w, tc.h, w, out)
			}
		}
		if n := len(strings.Split(f.View().Content, "\n")); n != tc.h {
			t.Errorf("%dx%d: %d lines", tc.w, tc.h, n)
		}
		// The screens get the body's width: the rail takes its share.
		if want := tc.w; tc.mode == barRail && s.width != tc.w-railCols || tc.mode != barRail && s.width != want {
			t.Errorf("%dx%d: screen width %d", tc.w, tc.h, s.width)
		}
	}
}

func TestActionBarTaps(t *testing.T) {
	for _, size := range [][2]int{{83, 40}, {150, 26}} {
		f, s := barFrame(t, size[0], size[1])
		tap(t, f, touch.KeyMsg("w"))
		if strings.Join(s.keys, ",") != "w" {
			t.Fatalf("%v: keys = %v", size, s.keys)
		}
		// More opens the sheet with the rest; a tap there runs it on the
		// screen below and closes the sheet.
		tap(t, f, tea.Msg(moreMsg{}))
		sheet, ok := f.Top().(*moreSheet)
		if !ok {
			t.Fatalf("%v: top = %T", size, f.Top())
		}
		if out := screen(f); !strings.Contains(out, "Label") || !strings.Contains(out, "Refresh") || !strings.Contains(out, "Quit") {
			t.Errorf("%v: sheet:\n%s", size, out)
		}
		var label int
		for i, a := range sheet.acts {
			if a.label == "Label" {
				label = i
			}
		}
		c := sheet.cells[label]
		run(f, tea.MouseClickMsg{X: c.x0 + 1, Y: f.bodyTop + c.y + 1, Button: tea.MouseLeft})
		if f.Top() != Screen(s) || s.keys[len(s.keys)-1] != "l" {
			t.Errorf("%v: after the sheet: top %T, keys %v", size, f.Top(), s.keys)
		}
	}
}

func TestActionBarQuestionMarkOpensMore(t *testing.T) {
	f, _ := barFrame(t, 83, 40)
	run(f, tea.KeyPressMsg{Code: '?', Text: "?"})
	if _, ok := f.Top().(*moreSheet); !ok {
		t.Fatalf("? opened %T", f.Top())
	}
}

// With kitty graphics, each button is an image: sent after the update,
// drawn with placeholders.
func TestActionBarImages(t *testing.T) {
	f, _ := barFrame(t, 83, 40)
	f.term.Kitty, f.term.Cell.W, f.term.Cell.H = true, 8, 16
	_, cmd := f.Update(tea.WindowSizeMsg{Width: 83, Height: 40})
	sent := ""
	var walk func(tea.Cmd)
	walk = func(c tea.Cmd) {
		if c == nil {
			return
		}
		switch m := c().(type) {
		case tea.BatchMsg:
			for _, c := range m {
				walk(c)
			}
		case tea.RawMsg:
			sent += m.Msg.(string)
		}
	}
	walk(cmd)
	if !strings.Contains(sent, "\x1b_G") || len(f.pills) != 5 {
		t.Fatalf("pills = %d, sent %d bytes", len(f.pills), len(sent))
	}
	if !strings.ContainsRune(f.View().Content, 0x10EEEE) {
		t.Error("the bar isn't drawn with placeholders")
	}
	// Nothing is sent again for the same bar.
	if _, cmd := f.Update(tea.KeyPressMsg{Code: 'x', Text: "x"}); cmd != nil {
		sent = ""
		walk(cmd)
		if strings.Contains(sent, "\x1b_G") {
			t.Error("the same buttons were sent again")
		}
	}
}
