// Package bigtext draws tall, touch-friendly buttons as images, shown
// with kitty Unicode placeholders. Big headings live in the markdown
// library (github.com/ionut-maxim/markdown/bigtext), whose font this uses.
//
// Rex reports a nominal 8×16 cell whatever the client's real one, so
// strips are drawn at 48 pixels a cell or more and the terminal scales
// them into place.
package bigtext

import (
	"image"
	"image/color"
	"math"
	"os"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	mdbigtext "github.com/ionut-maxim/markdown/bigtext"
	"github.com/ionut-maxim/markdown/kitty"
)

// FontFile, when set before the first drawing, names a TTF or OTF
// file to draw with instead of the markdown library's JetBrains Mono Bold.
var FontFile string

// Font is the font for buttons: FontFile when set and readable, else
// the markdown library's JetBrains Mono Bold.
var Font = sync.OnceValue(func() *opentype.Font {
	if p := FontFile; p != "" {
		if data, err := os.ReadFile(p); err == nil {
			if f, err := opentype.Parse(data); err == nil {
				return f
			}
		}
	}
	return mdbigtext.Font()
})

func hasGlyphs(face font.Face, s string) bool {
	for _, r := range s {
		if _, ok := face.GlyphAdvance(r); !ok {
			return false
		}
	}
	return true
}

// stripFace is the font at the size for button labels: about one row.
func stripFace(ch int) (font.Face, error) { return stripFaceAt(float64(ch)) }

// stripFaceAt is the strip's font for text h pixels tall (a cell's
// height gives the terminal's own text size).
func stripFaceAt(h float64) (font.Face, error) {
	return opentype.NewFace(Font(), &opentype.FaceOptions{Size: h * 0.62, DPI: 72, Hinting: font.HintingFull})
}

// roundRect fills a rounded rectangle, anti-aliased at the corners.
func roundRect(img *image.RGBA, x0, y0, x1, y1, r float64, c color.Color) {
	cr, cg, cb, _ := c.RGBA()
	for y := int(y0); y < int(math.Ceil(y1)); y++ {
		for x := int(x0); x < int(math.Ceil(x1)); x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			// Distance outside the rounded shape (negative inside).
			dx := math.Max(math.Max(x0+r-px, px-(x1-r)), 0)
			dy := math.Max(math.Max(y0+r-py, py-(y1-r)), 0)
			d := math.Hypot(dx, dy) - r
			a := math.Max(0, math.Min(1, 0.5-d))
			if a <= 0 {
				continue
			}
			img.SetRGBA(x, y, color.RGBA{uint8(float64(cr>>8) * a), uint8(float64(cg>>8) * a), uint8(float64(cb>>8) * a), uint8(255 * a)})
		}
	}
}

// HasGlyphs is true when the button font draws every rune of s (Nerd Font
// icons aren't in it).
func HasGlyphs(s string) bool {
	face, err := stripFace(48)
	if err != nil {
		return false
	}
	defer face.Close()
	return hasGlyphs(face, s)
}

// Button is one button of a strip.
type Button struct {
	Key     string // identifies the button in the spans
	Text    string // what's drawn: an icon (see HasGlyphs) and a label
	Primary bool   // drawn in the accent colour
	Pressed bool   // drawn in the bright accent, while tapped
}

// Span is where a button is in a strip, in cells from its left edge.
type Span struct {
	X0, X1 int // X1 exclusive
	Key    string
}

// measureStrip lays the buttons out in cells: each is its text plus
// padding, with a one-cell gap between.
func measureStrip(face font.Face, buttons []Button, cw int) (cols int, spans []Span) {
	x := 0
	for i, b := range buttons {
		if i > 0 {
			x++
		}
		w := (font.MeasureString(face, b.Text).Ceil() + 2*cw + cw - 1) / cw
		w = max(w, 4) + 2 // room for the rounded ends
		spans = append(spans, Span{X0: x, X1: x + w, Key: b.Key})
		x += w
	}
	return x, spans
}

// Strip draws buttons as rounded, filled shapes rows cells tall, in one
// image, for a terminal whose cells are cell (Term.Cell): its width in
// cells and where each button is come back with it.
func Strip(buttons []Button, rows int, cell kitty.Cell) (image.Image, int, []Span) {
	cw, ch := mdbigtext.Scale(cell)
	face, err := stripFace(ch)
	if err != nil {
		return nil, 0, nil
	}
	defer face.Close()
	cols, spans := measureStrip(face, buttons, cw)
	w, h := cols*cw, rows*ch
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	// Leave a little air above and below so stacked cards don't touch.
	inset := float64(ch) * 0.18
	if rows >= 3 {
		inset = float64(ch) * 0.35
	}
	m := face.Metrics()
	for i, sp := range spans {
		b := buttons[i]
		bg, fg := buttonColors(b.Primary, b.Pressed)
		x0, x1 := float64(sp.X0*cw)+1, float64(sp.X1*cw)-1
		y0, y1 := inset, float64(h)-inset
		roundRect(img, x0, y0, x1, y1, math.Min((y1-y0)/2, float64(ch)*0.3), bg)
		tw := font.MeasureString(face, b.Text).Ceil()
		base := int((y0+y1)/2) + (m.Ascent.Ceil()-m.Descent.Ceil())/2
		d := font.Drawer{Dst: img, Src: image.NewUniform(fg), Face: face,
			Dot: fixed.P(int((x0+x1)/2)-tw/2, base)}
		d.DrawString(b.Text)
	}
	return img, cols, spans
}
