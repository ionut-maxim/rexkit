package components

import (
	"github.com/ionut-maxim/markdown"

	"github.com/ionut-maxim/rexkit/icons"
	"github.com/ionut-maxim/rexkit/theme"
)

// MarkdownTheme gives the markdown library's renderer the current palette
// and icons. dark picks Chroma's dark or light code style.
func MarkdownTheme(dark bool) markdown.Theme {
	p := theme.Pal
	th := markdown.DefaultTheme(dark)
	th.Text, th.Subtle, th.Muted, th.Accent, th.Accent2 = p.Text, p.Subtle, p.Muted, p.Accent, p.Pink
	th.Rule, th.Surface, th.SurfaceHi = p.SurfaceHi, p.Surface, p.SurfaceHi
	th.Green, th.Red, th.Yellow, th.Blue = p.Green, p.Red, p.Yellow, p.Blue
	th.Bullets, th.CheckOn, th.CheckOff = icons.Current.Bullets, icons.Current.BoxOn, icons.Current.BoxOff
	return th
}
