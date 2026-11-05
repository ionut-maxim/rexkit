package markdown

import (
	"charm.land/lipgloss/v2"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestSplit(t *testing.T) {
	md := "Before\n\n![shot.png](https://images.example.com/a/b/c) after on the same line\n\n" +
		"```\n![not an image](https://example.com/x)\n```\n\n" +
		"![](<https://images.example.com/d> \"title\")"
	got := Split(md)
	if len(got) != 4 {
		for i, s := range got {
			t.Logf("%d: %+v %+v", i, s.Text, s.Image)
		}
		t.Fatalf("got %d segments, want 4", len(got))
	}
	if got[0].Text != "Before\n\n" {
		t.Errorf("seg 0 = %q", got[0].Text)
	}
	if im := got[1].Image; im == nil || im.Alt != "shot.png" || im.URL != "https://images.example.com/a/b/c" {
		t.Errorf("seg 1 = %+v", got[1].Image)
	}
	if !strings.Contains(got[2].Text, "after on the same line") || !strings.Contains(got[2].Text, "![not an image]") {
		t.Errorf("seg 2 = %q", got[2].Text)
	}
	if im := got[3].Image; im == nil || im.URL != "https://images.example.com/d" || im.Alt != "" {
		t.Errorf("seg 3 = %+v", got[3].Image)
	}
}

func TestSplitNoImages(t *testing.T) {
	got := Split("just *text*")
	if len(got) != 1 || got[0].Image != nil {
		t.Fatalf("got %+v", got)
	}
	if Split("") != nil {
		t.Fatal("empty markdown should give no segments")
	}
}

func TestImages(t *testing.T) {
	im := Images("a ![x](https://u/1) b ![y](https://u/2)")
	if len(im) != 2 || im[1].URL != "https://u/2" {
		t.Fatalf("images = %+v", im)
	}
}

func theme() Theme {
	c := lipgloss.Color("#888888")
	return Theme{Dark: true, Text: c, Subtle: c, Muted: c, Accent: c, Accent2: c, Rule: c, Surface: c, SurfaceHi: c,
		Green: c, Red: c, Yellow: c, Blue: c, Bullets: []string{"•", "◦"}, CheckOn: "☑", CheckOff: "☐", Quote: "▎"}
}

func plain(src string, width int) string { return ansi.Strip(RenderTheme(src, width, theme())) }

func TestRenderWraps(t *testing.T) {
	out := plain("# Title\n\nsome **bold** text that is long enough to wrap around at a narrow width", 30)
	if !strings.Contains(out, "Title") || strings.Contains(out, "#") || !strings.Contains(out, "bold") || !strings.Contains(out, "━━━") {
		t.Fatalf("render:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if w := ansi.StringWidth(line); w > 30 {
			t.Fatalf("line wider than 30 (%d): %q", w, line)
		}
	}
}

func TestRenderBlocks(t *testing.T) {
	src := "1. one\n2. two\n   - nested\n\n- [x] done\n- [ ] todo\n\n> quoted\n\n```go\nx := 1\n```\n\n---"
	out := plain(src, 40)
	for _, want := range []string{"1. one", "2. two", "   ◦ nested", "☑ done", "☐ todo", "▎ quoted", "go", "x := 1", strings.Repeat("─", 40)} {
		if !strings.Contains(out, want) {
			t.Errorf("render lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "[x]") || strings.Contains(out, "```") {
		t.Errorf("markdown syntax left in:\n%s", out)
	}
}

func TestRenderHighlightsCode(t *testing.T) {
	out := RenderTheme("```go\nfunc main() {}\n```", 40, Theme{Dark: true, Text: lipgloss.Color("#ffffff"), Muted: lipgloss.Color("#777777"),
		Surface: lipgloss.Color("#222222"), Bullets: []string{"•"}})
	// Keywords get their own colour: more than one foreground in the line.
	line := strings.Split(out, "\n")[1]
	if strings.Count(line, "38;2;") < 2 {
		t.Fatalf("no highlighting: %q", line)
	}
}

func TestRenderTables(t *testing.T) {
	src := "| Key | Action |\n|---|--:|\n| d | done |\n| a | answer |"
	out := plain(src, 40)
	if !strings.Contains(out, "╭") || !strings.Contains(out, "│ d   │   done │") {
		t.Fatalf("grid:\n%s", out)
	}
	// Too narrow for the grid: cards.
	out = plain("| A long header | Another long header |\n|---|---|\n| some value | other value |", 20)
	if strings.Contains(out, "╭") || !strings.Contains(out, "▎ A long header\n▎   some value") {
		t.Fatalf("cards:\n%s", out)
	}
}

func TestRenderLinks(t *testing.T) {
	out := RenderTheme("see [docs](https://e.example/x)", 40, theme())
	if !strings.Contains(out, "\x1b]8;;https://e.example/x") || !strings.Contains(ansi.Strip(out), "see docs") {
		t.Fatalf("link: %q", out)
	}
}

func TestRenderEscapes(t *testing.T) {
	out := plain(`Food (\~7k/month now) and Gaming (\~1.5k/month now) \*not bold\* Tom &amp; Jerry &#169; `+"`a\\~b`", 80)
	for _, want := range []string{"Food (~7k/month now) and Gaming (~1.5k/month now)", "*not bold*", "Tom & Jerry ©", `a\~b`} {
		if !strings.Contains(out, want) {
			t.Errorf("render lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, `(\~`) {
		t.Errorf("escape left in:\n%s", out)
	}
}
