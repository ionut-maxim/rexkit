// Package theme is the colours and shared styles of a terminal app: a
// palette for dark backgrounds and one for light, with a purple accent as
// the accent. Set picks one when the terminal reports its background.
package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Palette is the app's colours. There's one for dark backgrounds and one
// for light ones; setTheme picks it when the terminal reports its
// background. The Accent is purple.
type Palette struct {
	Accent, AccentSoft, Text, Subtle, Muted, Surface, SurfaceHi color.Color
	Green, Yellow, Orange, Red, Blue, Cyan, Pink                color.Color
}

// Dark is the palette for dark backgrounds.
var Dark = Palette{
	Accent:     lipgloss.Color("#8B93FF"),
	AccentSoft: lipgloss.Color("#5E6AD2"),
	Text:       lipgloss.Color("#E6E6EB"),
	Subtle:     lipgloss.Color("#9BA0AB"),
	Muted:      lipgloss.Color("#5F6470"),
	Surface:    lipgloss.Color("#262836"),
	SurfaceHi:  lipgloss.Color("#34374A"),
	Green:      lipgloss.Color("#4CB782"),
	Yellow:     lipgloss.Color("#F2C94C"),
	Orange:     lipgloss.Color("#F2994A"),
	Red:        lipgloss.Color("#EB5757"),
	Blue:       lipgloss.Color("#4EA7FC"),
	Cyan:       lipgloss.Color("#26B5CE"),
	Pink:       lipgloss.Color("#E879B9"),
}

// Light is the palette for light backgrounds.
var Light = Palette{
	Accent:     lipgloss.Color("#5E6AD2"),
	AccentSoft: lipgloss.Color("#8B93FF"),
	Text:       lipgloss.Color("#1F2023"),
	Subtle:     lipgloss.Color("#5E6169"),
	Muted:      lipgloss.Color("#9A9DA6"),
	Surface:    lipgloss.Color("#ECEDF5"),
	SurfaceHi:  lipgloss.Color("#DCDEEC"),
	Green:      lipgloss.Color("#2E9A64"),
	Yellow:     lipgloss.Color("#B98900"),
	Orange:     lipgloss.Color("#D9730D"),
	Red:        lipgloss.Color("#D93F3F"),
	Blue:       lipgloss.Color("#1F7AE0"),
	Cyan:       lipgloss.Color("#0E8FA6"),
	Pink:       lipgloss.Color("#C2408C"),
}

// Pal is the palette in use.
var Pal = Dark

// StyleSet is the shared styles, built from the palette.
type StyleSet struct {
	Brand, Crumb, Faint, Status, Err, ErrTitle, ErrBox lipgloss.Style
	Title, Subtle, Muted, Accent, Ok, Warn             lipgloss.Style
	KeyCap, KeyDesc                                    lipgloss.Style
}

// Styles are the shared styles, built from Pal.
var Styles StyleSet

func init() { Set(true) }

// Set switches the palette for a dark or light background.
func Set(dark bool) {
	if dark {
		Pal = Dark
	} else {
		Pal = Light
	}
	fg := func(c color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }
	Styles = StyleSet{
		Brand:    fg(Pal.Accent).Bold(true),
		Crumb:    fg(Pal.Text).Bold(true),
		Faint:    fg(Pal.Muted),
		Status:   fg(Pal.Cyan),
		Err:      fg(Pal.Red),
		ErrTitle: fg(Pal.Red).Bold(true),
		ErrBox:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Pal.Red).Padding(1, 2),
		Title:    fg(Pal.Text).Bold(true),
		Subtle:   fg(Pal.Subtle),
		Muted:    fg(Pal.Muted),
		Accent:   fg(Pal.Accent),
		Ok:       fg(Pal.Green),
		Warn:     fg(Pal.Yellow),
		KeyCap:   lipgloss.NewStyle().Foreground(Pal.Text).Background(Pal.SurfaceHi).Bold(true),
		KeyDesc:  fg(Pal.Subtle),
	}
}

// Hex turns a colour (#rrggbb) into a colour, falling back to the
// palette's subtle grey when it is missing.
func Hex(c string) color.Color {
	if len(c) == 7 && c[0] == '#' {
		return lipgloss.Color(c)
	}
	return Pal.Subtle
}
