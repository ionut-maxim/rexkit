package components

import (
	"github.com/ionut-maxim/rexkit/icons"
	"github.com/ionut-maxim/rexkit/markdown"
	"github.com/ionut-maxim/rexkit/theme"
)

// MarkdownTheme gives the markdown renderer the current palette and icons.
// dark picks Chroma's dark or light code style.
func MarkdownTheme(dark bool) markdown.Theme {
	p := theme.Pal
	return markdown.Theme{
		Dark: dark, Text: p.Text, Subtle: p.Subtle, Muted: p.Muted, Accent: p.Accent, Accent2: p.Pink,
		Rule: p.SurfaceHi, Surface: p.Surface, SurfaceHi: p.SurfaceHi,
		Green: p.Green, Red: p.Red, Yellow: p.Yellow, Blue: p.Blue,
		Bullets: icons.Current.Bullets, CheckOn: icons.Current.BoxOn, CheckOff: icons.Current.BoxOff, Quote: "▎",
	}
}
