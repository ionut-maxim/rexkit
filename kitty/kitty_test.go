package kitty

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

func TestFit(t *testing.T) {
	cases := []struct {
		w, h, maxC, maxR int
		cols, rows       int
	}{
		{640, 360, 200, 100, 64, 18}, // natural size: 640/10 cols, 360/20 rows
		{640, 360, 32, 100, 32, 9},   // narrower box halves both
		{640, 360, 200, 9, 32, 9},    // row cap shrinks the width to match
		{2880, 1800, 80, 30, 80, 25}, // big screenshot fits the width
		{5, 5, 80, 30, 1, 1},         // tiny images still get a cell
		{0, 10, 80, 30, 0, 0},
	}
	for _, c := range cases {
		cols, rows := Fit(c.w, c.h, c.maxC, c.maxR)
		if cols != c.cols || rows != c.rows {
			t.Errorf("Fit(%d,%d,%d,%d) = %d,%d; want %d,%d", c.w, c.h, c.maxC, c.maxR, cols, rows, c.cols, c.rows)
		}
	}
}

func TestPlaceholders(t *testing.T) {
	out := Placeholders(0x123456, 3, 2)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines", len(lines))
	}
	if !strings.HasPrefix(lines[0], "\x1b[38;2;18;52;86m") {
		t.Errorf("id not encoded in the colour: %q", lines[0][:20])
	}
	for r, line := range lines {
		if w := ansi.StringWidth(line); w != 3 {
			t.Errorf("line %d width = %d, want 3", r, w)
		}
		runes := []rune(ansi.Strip(line))
		if len(runes) != 9 || runes[0] != kitty.Placeholder || runes[1] != kitty.Diacritic(r) || runes[5] != kitty.Diacritic(1) {
			t.Errorf("line %d cells = %U", r, runes)
		}
	}
}

func TestDecodeAndTransmit(t *testing.T) {
	data, err := os.ReadFile("testdata/testcard.png")
	if err != nil {
		t.Fatal(err)
	}
	im, err := Decode(4242, data)
	if err != nil {
		t.Fatal(err)
	}
	if im.Bounds.Dx() != 640 || im.Bounds.Dy() != 360 {
		t.Fatalf("bounds = %v", im.Bounds)
	}
	seq, err := im.Transmit(32, 9)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"i=4242", "s=320", "v=180", "o=z", "q=2", "a=p", "U=1", "c=32", "r=9"} {
		if !strings.Contains(seq, want) {
			t.Errorf("sequence lacks %s", want)
		}
	}
	if !strings.HasPrefix(seq, "\x1b_G") {
		t.Errorf("not an APC kitty sequence: %q", seq[:10])
	}
	if _, err := Decode(1, []byte("not an image")); err == nil {
		t.Error("want an error for garbage")
	}
}

func TestQueryAndDelete(t *testing.T) {
	if !strings.Contains(Query, "a=q") || !strings.HasSuffix(Query, "\x1b[c") {
		t.Errorf("Query = %q", Query)
	}
	if d := Delete(7); !strings.Contains(d, "a=d") || !strings.Contains(d, "d=I") || !strings.Contains(d, "i=7") {
		t.Errorf("Delete = %q", d)
	}
}
