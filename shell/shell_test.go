package shell

import (
	"errors"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ionut-maxim/rexkit/icons"
	"github.com/ionut-maxim/rexkit/rexstatus"
)

type page struct {
	title  string
	closed *int
}

func (p *page) Init() tea.Cmd                    { return nil }
func (p *page) Update(tea.Msg) (Screen, tea.Cmd) { return p, nil }
func (p *page) View(int, int) string             { return "body of " + p.title }
func (p *page) Title() string                    { return p.title }
func (p *page) Keys() []key.Binding              { return nil }
func (p *page) Capturing() bool                  { return false }
func (p *page) Refresh() tea.Cmd                 { return nil }
func (p *page) Close() tea.Cmd                   { *p.closed++; return nil }

// run feeds msg and every message its commands make back into the frame.
func run(f *Frame, msg tea.Msg) {
	queue := []tea.Msg{msg}
	for n := 0; len(queue) > 0 && n < 100; n++ {
		m := queue[0]
		queue = queue[1:]
		_, cmd := f.Update(m)
		queue = append(queue, exec(cmd)...)
	}
}

func exec(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	switch m := cmd().(type) {
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range m {
			out = append(out, exec(c)...)
		}
		return out
	case nil, tea.RawMsg:
		return nil
	default:
		return []tea.Msg{m}
	}
}

func TestViewsKeepTheirOwnStacks(t *testing.T) {
	icons.Current = icons.Unicode
	closed := 0
	f := New(Options{App: "demo", Images: ImagesOff, Views: []View{
		{Name: "Mail", Icon: "✉", Root: func() Screen { return &page{"mail", &closed} }, Badge: func(Screen) string { return "3" }},
		{Name: "Lists", Icon: "≡", Root: func() Screen { return &page{"lists", &closed} }},
	}, Context: func() string { return "acme" }, ErrText: func(err error) string { return "oops: " + err.Error() }})
	run(f, tea.WindowSizeMsg{Width: 80, Height: 20})
	if f.SwitchTo(1) != nil || f.Top() != nil {
		t.Fatal("nothing switches before Start")
	}
	exec(f.Start())
	if f.Top() == nil || f.Top().Title() != "mail" {
		t.Fatalf("top = %v", f.Top())
	}
	run(f, Push{&page{"one", &closed}})
	run(f, SwitchView{1})
	if f.Top().Title() != "lists" {
		t.Fatalf("top = %q", f.Top().Title())
	}
	out := ansi.Strip(f.View().Content)
	for _, want := range []string{"✉ Mail 3", "≡ Lists", "acme", "body of lists"} {
		if !strings.Contains(out, want) {
			t.Fatalf("lacks %q:\n%s", want, out)
		}
	}
	run(f, SwitchView{0})
	if f.Top().Title() != "one" {
		t.Fatalf("back on Mail, top = %q", f.Top().Title())
	}
	run(f, SwitchView{0}) // the current tab again: its first screen
	if f.Top().Title() != "mail" || closed != 1 {
		t.Fatalf("top = %q, closed = %d", f.Top().Title(), closed)
	}
	run(f, Err{errors.New("boom")})
	if out := ansi.Strip(f.View().Content); !strings.Contains(out, "oops: boom") {
		t.Fatalf("error line:\n%s", out)
	}
}

func TestProgressCountsJobs(t *testing.T) {
	f := New(Options{App: "demo", Images: ImagesOff})
	_, a := f.Update(Progress{rexstatus.Working, "one"})
	_, _ = f.Update(Progress{rexstatus.Working, "two"})
	if a == nil {
		t.Fatal("working is reported")
	}
	if _, cmd := f.Update(Progress{rexstatus.Done, "one done"}); cmd != nil {
		t.Fatal("done waits for the other job")
	}
	if _, cmd := f.Update(Progress{rexstatus.Done, "two done"}); cmd == nil {
		t.Fatal("the last job reports done")
	}
}
