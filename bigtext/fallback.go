package bigtext

import (
	_ "embed"
	"image"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed fonts/SymbolsNerdFontMono-subset.ttf
var symbolsTTF []byte

// symbols is the Nerd Font icons (see fonts/README.md), for the icons
// the label font lacks.
var symbols = sync.OnceValue(func() *opentype.Font {
	f, err := opentype.Parse(symbolsTTF)
	if err != nil {
		return nil
	}
	return f
})

// fallbackFace draws each rune with the first face that has it: the
// label font, then the icons.
type fallbackFace struct{ faces []font.Face }

func (f fallbackFace) pick(r rune) font.Face {
	for _, fc := range f.faces {
		if _, ok := fc.GlyphAdvance(r); ok {
			return fc
		}
	}
	return f.faces[0]
}

func (f fallbackFace) Close() error {
	for _, fc := range f.faces {
		fc.Close()
	}
	return nil
}

func (f fallbackFace) Glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	return f.pick(r).Glyph(dot, r)
}

func (f fallbackFace) GlyphBounds(r rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) {
	return f.pick(r).GlyphBounds(r)
}

func (f fallbackFace) GlyphAdvance(r rune) (fixed.Int26_6, bool) {
	return f.pick(r).GlyphAdvance(r)
}

func (f fallbackFace) Kern(r0, r1 rune) fixed.Int26_6 {
	a, b := f.pick(r0), f.pick(r1)
	if a != b {
		return 0
	}
	return a.Kern(r0, r1)
}

func (f fallbackFace) Metrics() font.Metrics { return f.faces[0].Metrics() }
