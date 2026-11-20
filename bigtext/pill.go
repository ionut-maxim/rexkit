package bigtext

import (
	"image"
	"image/color"

	mdbigtext "github.com/ionut-maxim/markdown/bigtext"
	"github.com/ionut-maxim/markdown/kitty"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/ionut-maxim/rexkit/theme"
)

// Pill is one button of an action bar: its key in the accent colour,
// then its label, centred on a rounded shape that fills cols by rows
// cells, with a little air around it so neighbours don't touch.
type Pill struct {
	Key, Label string
	Primary    bool // the screen's main action: filled in the accent
	Pressed    bool // being tapped: the bright accent
}

// DrawPill draws p for a terminal whose cells are cell (Term.Cell). The
// text shrinks to fit, and drops the label, then the key, when even that
// doesn't.
func DrawPill(p Pill, cols, rows int, cell kitty.Cell) image.Image {
	cw, ch := mdbigtext.Scale(cell)
	w, h := cols*cw, rows*ch
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	// Quiet colours: buttons sit on the surface, the main action is only
	// tinted with the accent, and a tap deepens the tint.
	bg, fg, keyC := theme.Pal.SurfaceHi, theme.Pal.Text, theme.Pal.Subtle
	if p.Primary {
		bg, keyC = mix(theme.Pal.SurfaceHi, theme.Pal.AccentSoft, 0.35), theme.Pal.Accent
	}
	if p.Pressed {
		bg, keyC = mix(theme.Pal.SurfaceHi, theme.Pal.AccentSoft, 0.7), theme.Pal.Text
	}
	padX, padY := float64(cw)*0.35, float64(ch)*0.3
	if rows >= 3 {
		padY = float64(ch) * 0.4
	}
	x0, y0, x1, y1 := padX, padY, float64(w)-padX, float64(h)-padY
	// Softly rounded corners, not a capsule.
	roundRect(img, x0, y0, x1, y1, float64(ch)*0.3, bg)

	// The text: as big as a cell and a bit, smaller when it doesn't fit.
	room := int(x1-x0) - 2*cw
	for _, size := range []float64{1.15, 1.0, 0.85, 0.7} {
		face, err := stripFaceAt(float64(ch) * size)
		if err != nil {
			return img
		}
		key, label := p.Key, p.Label
		if !hasGlyphs(face, key) {
			key = "" // the font lacks it: an empty box would be worse
		}
		gap := " "
		kw := font.MeasureString(face, key).Ceil()
		lw := font.MeasureString(face, gap+label).Ceil()
		if key == "" {
			lw = font.MeasureString(face, label).Ceil()
		}
		total := kw + lw
		if total > room && size == 0.7 {
			// Still too wide: the label alone, then the key alone.
			if lw <= room && label != "" {
				key, kw, total = "", 0, font.MeasureString(face, label).Ceil()
				lw = total
				gap = ""
			} else {
				label, lw, total = "", 0, kw
			}
		}
		if total > room && size != 0.7 {
			face.Close()
			continue
		}
		m := face.Metrics()
		base := int((y0+y1)/2) + (m.Ascent.Ceil()-m.Descent.Ceil())/2
		x := int((x0+x1)/2) - total/2
		if key != "" {
			d := font.Drawer{Dst: img, Src: image.NewUniform(keyC), Face: face, Dot: fixed.P(x, base)}
			d.DrawString(key)
			x += kw
		} else {
			gap = ""
		}
		if label != "" {
			d := font.Drawer{Dst: img, Src: image.NewUniform(fg), Face: face, Dot: fixed.P(x, base)}
			d.DrawString(gap + label)
		}
		face.Close()
		break
	}
	return img
}

// mix is a blended t of the way from a to b.
func mix(a, b color.Color, t float64) color.Color {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	f := func(x, y uint32) uint8 { return uint8((float64(x>>8)*(1-t) + float64(y>>8)*t) + 0.5) }
	return color.RGBA{f(ar, br), f(ag, bg), f(ab, bb), 255}
}
