package form

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ionut-maxim/rexkit/icons"
	"github.com/ionut-maxim/rexkit/shell"
	"github.com/ionut-maxim/rexkit/touch"
)

type submitted struct{ values []string }

func newForm(tall bool) (*Form, *[]string) {
	var got []string
	f := New(Options{
		Title:  "New address",
		Intro:  "For one site.",
		Submit: "Create",
		Fields: []Field{
			{Label: "Domain", Required: true, Placeholder: "example.com"},
			{Label: "Description", Hint: "Optional"},
			{Label: "Prefix", Check: func(v string) error {
				if strings.ToLower(v) != v {
					return errors.New("lower case only")
				}
				return nil
			}},
		},
		OnSubmit: func(v []string) tea.Cmd {
			got = v
			return func() tea.Msg { return submitted{v} }
		},
		Tall: func() bool { return tall },
	})
	f.Init()
	return f, &got
}

func typeText(f *Form, s string) {
	for _, r := range s {
		f.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func press(f *Form, name string) tea.Cmd {
	_, cmd := f.Update(touch.KeyMsg(name))
	return cmd
}

func msgOf(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func TestSubmitChecksEveryField(t *testing.T) {
	f, got := newForm(false)
	if cmd := press(f, "enter"); cmd != nil || *got != nil {
		t.Fatal("an empty required field was submitted")
	}
	if !strings.Contains(ansi.Strip(f.View(83, 30)), "Domain is needed") {
		t.Fatalf("no message:\n%s", ansi.Strip(f.View(83, 30)))
	}
	typeText(f, " example.com ")
	f.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	f.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	typeText(f, "Shop")
	press(f, "enter")
	if *got != nil || f.Focused() != 2 || !strings.Contains(ansi.Strip(f.View(83, 30)), "lower case only") {
		t.Fatalf("a failing check was submitted; focus %d", f.Focused())
	}
	f.SetValue(2, "shop")
	cmd := press(f, "enter")
	if s, ok := msgOf(cmd).(submitted); !ok || strings.Join(s.values, "|") != "example.com||shop" {
		t.Fatalf("submitted %v", msgOf(cmd))
	}
	if !f.Busy() || f.Keys() != nil {
		t.Fatal("busy while the job runs, with no keys")
	}
	if press(f, "enter") != nil {
		t.Fatal("a second enter while busy did something")
	}
}

func TestDone(t *testing.T) {
	f, _ := newForm(false)
	typeText(f, "example.com")
	press(f, "enter")
	f.Update(Done{Err: errors.New("Fastmail said no")})
	if f.Busy() || !strings.Contains(ansi.Strip(f.View(83, 30)), "Fastmail said no") {
		t.Fatal("a failed job shows its error and lets you try again")
	}
	_, cmd := f.Update(Done{Status: "Created"})
	// tea.Sequence's message is an unexported []tea.Cmd.
	var msgs []tea.Msg
	seq := reflect.ValueOf(msgOf(cmd))
	for i := range seq.Len() {
		msgs = append(msgs, seq.Index(i).Interface().(tea.Cmd)())
	}
	if len(msgs) != 2 || msgs[0] != (shell.Pop{}) || msgs[1] != shell.Status("Created") {
		t.Fatalf("done sends %v", msgs)
	}
}

func TestEscCancels(t *testing.T) {
	f, _ := newForm(false)
	if msgOf(press(f, "esc")) != (shell.Pop{}) {
		t.Fatal("esc closes the form")
	}
}

// tap clicks the first cell whose text matches, then lets the flash end.
func tap(t *testing.T, f *Form, width int, text string) tea.Msg {
	t.Helper()
	view := strings.Split(ansi.Strip(f.View(width, 40)), "\n")
	for y, l := range view {
		if x := strings.Index(l, text); x >= 0 {
			x = ansi.StringWidth(l[:x])
			_, cmd := f.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
			m := msgOf(cmd)
			if r, ok := m.(releaseMsg); ok {
				_, cmd = f.Update(r)
				return msgOf(cmd)
			}
			return m
		}
	}
	t.Fatalf("no %q in:\n%s", text, strings.Join(view, "\n"))
	return nil
}

func TestTaps(t *testing.T) {
	icons.Current = icons.Unicode
	for _, width := range []int{50, 83, 165} {
		f, got := newForm(width == 83)
		tap(t, f, width, "Prefix")
		if f.Focused() != 2 {
			t.Fatalf("width %d: tapping a field focuses it (focus %d)", width, f.Focused())
		}
		tap(t, f, width, "Domain")
		typeText(f, "example.com")
		if m, ok := tap(t, f, width, "Create").(submitted); !ok || m.values[0] != "example.com" || (*got)[0] != "example.com" {
			t.Fatalf("width %d: tapping Create sends %v", width, m)
		}
		f.Update(Done{Err: errors.New("no")})
		if tap(t, f, width, "Cancel") != (shell.Pop{}) {
			t.Fatalf("width %d: tapping Cancel closes", width)
		}
	}
}

func TestLayouts(t *testing.T) {
	icons.Current = icons.Unicode
	f, _ := newForm(false)
	for _, width := range []int{50, 83, 165} {
		out := ansi.Strip(f.View(width, 30))
		for i, l := range strings.Split(out, "\n") {
			if w := ansi.StringWidth(l); w > width {
				t.Fatalf("width %d: line %d is %d wide:\n%s", width, i, w, out)
			}
		}
	}
	compact := ansi.Strip(f.View(50, 30))
	if !strings.Contains(compact, "▌ Domain*\n") {
		t.Fatalf("compact puts the label on its own line:\n%s", compact)
	}
	regular := ansi.Strip(f.View(83, 30))
	if !strings.Contains(regular, "Domain*") || strings.Contains(regular, "Domain*\n") {
		t.Fatalf("regular puts label and field on one line:\n%s", regular)
	}
	tallF, _ := newForm(true)
	if a, b := strings.Count(ansi.Strip(tallF.View(83, 50)), "\n"), strings.Count(regular, "\n"); a <= b+3 {
		t.Fatalf("tall adds room: %d vs %d lines", a, b)
	}
}
