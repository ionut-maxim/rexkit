package bigtext

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"testing"

	"github.com/ionut-maxim/markdown/kitty"

	"github.com/ionut-maxim/rexkit/theme"
)

// PILL_PREVIEW=<file> writes a sheet of buttons to look at.
func TestPillPreview(t *testing.T) {
	out := os.Getenv("PILL_PREVIEW")
	if out == "" {
		t.Skip("set PILL_PREVIEW")
	}
	theme.Set(true)
	cell := kitty.Cell{W: 8, H: 16}
	pills := []struct {
		p    Pill
		cols int
	}{
		{Pill{Key: "↵", Label: "Open", Primary: true}, 15},
		{Pill{Key: "w", Label: "Work on it"}, 15},
		{Pill{Key: "s", Label: "Status", Pressed: true}, 15},
		{Pill{Key: "⋯", Label: "More"}, 12},
		{Pill{Key: "a", Label: "Assignee"}, 21},
	}
	sheet := image.NewRGBA(image.Rect(0, 0, 80*48, 3*48))
	draw.Draw(sheet, sheet.Bounds(), image.NewUniform(theme.Pal.Surface), image.Point{}, draw.Src)
	x := 0
	for _, p := range pills {
		img := DrawPill(p.p, p.cols, 3, cell)
		draw.Draw(sheet, img.Bounds().Add(image.Pt(x, 0)), img, image.Point{}, draw.Over)
		x += img.Bounds().Dx() + 48
	}
	f, _ := os.Create(out)
	defer f.Close()
	png.Encode(f, sheet)
}

// PILL_PREVIEW_STRIP=<file> writes the Inbox card strip to look at.
func TestStripPreview(t *testing.T) {
	out := os.Getenv("PILL_PREVIEW_STRIP")
	if out == "" {
		t.Skip("set PILL_PREVIEW_STRIP")
	}
	theme.Set(true)
	img, _, _ := Strip([]Button{{Key: "d", Text: "✓ Done", Primary: true}, {Key: "D", Text: "▶ Check in a tab"}, {Key: "a", Text: "↳ Answer", Pressed: true}}, 2, kitty.Cell{W: 8, H: 16})
	sheet := image.NewRGBA(img.Bounds())
	draw.Draw(sheet, sheet.Bounds(), image.NewUniform(theme.Pal.Surface), image.Point{}, draw.Src)
	draw.Draw(sheet, sheet.Bounds(), img, image.Point{}, draw.Over)
	f, _ := os.Create(out)
	defer f.Close()
	png.Encode(f, sheet)
}
