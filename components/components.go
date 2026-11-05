// Package components is the building blocks of a touch-friendly terminal
// app: buttons with rounded ends that can be tall, pills, chips, section
// headers, progress bars, card surfaces, and the layout classes for the
// three widths of the Rex iPhone client.
package components

import (
	"image/color"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ionut-maxim/rexkit/icons"
	"github.com/ionut-maxim/rexkit/theme"
)

// Size is the layout class for a width. The iPhone's Rex client has three
// layouts (folded, unfolded, landscape); the breakpoints follow them.
type Size int

const (
	Compact Size = iota // < 64 columns: one thing per line
	Regular             // 64–119: the unfolded phone
	Wide                // ≥ 120: landscape and desktops
)

// SizeOf is the layout class for a width.
func SizeOf(width int) Size {
	switch {
	case width < 64:
		return Compact
	case width < 120:
		return Regular
	}
	return Wide
}

// Pill is text on a coloured background with rounded ends (square ones
// without a Nerd Font).
func Pill(text string, fg, bg color.Color) string {
	body := lipgloss.NewStyle().Foreground(fg).Background(bg).Render(text)
	if icons.Current.PillL == "" {
		return lipgloss.NewStyle().Foreground(fg).Background(bg).Render(" " + text + " ")
	}
	cap := lipgloss.NewStyle().Foreground(bg)
	return cap.Render(icons.Current.PillL) + body + cap.Render(icons.Current.PillR)
}

// PillWidth is the cell width pill(text) takes.
func PillWidth(text string) int { return ansi.StringWidth(text) + 2 }

// Button is something to tap: a rounded shape on the surface colour with
// an optional key in the accent colour, then the label.
func Button(key, label string) string { return Btn(key, label, theme.Pal.SurfaceHi, false).Text }

// ButtonWidth is the cell width of button(key, label).
func ButtonWidth(key, label string) int { return Btn(key, label, theme.Pal.SurfaceHi, false).W }

// Seg is a piece of a row that may hold buttons: plain text, or a button
// with a background colour. A row draws on one line; when tall, it gets an
// empty line above and below that still belongs to its buttons' tap areas,
// a bigger target for a finger without drawing anything odd.
type Seg struct {
	Text string // rendered
	W    int
	Bg   color.Color // nil for plain text
}

// Plain is a Seg of plain text, not tappable.
func Plain(text string) Seg { return Seg{Text: text, W: ansi.StringWidth(text)} }

// Btn is a button: the key in the accent colour, then the label, padded,
// with rounded ends (square without a Nerd Font). Tall ones get more
// padding.
func Btn(key, label string, bg color.Color, tall bool) Seg {
	inner := lipgloss.NewStyle().Background(bg)
	pad := " "
	if tall {
		pad = "  "
	}
	body := inner.Render(pad)
	if key != "" {
		body += inner.Foreground(theme.Pal.Accent).Bold(true).Render(key)
		if label != "" {
			body += inner.Render(" ")
		}
	}
	if label != "" {
		body += inner.Foreground(theme.Pal.Text).Render(label)
	}
	body += inner.Render(pad)
	if icons.Current.PillL != "" {
		cap := lipgloss.NewStyle().Foreground(bg)
		body = cap.Render(icons.Current.PillL) + body + cap.Render(icons.Current.PillR)
	}
	return Seg{Text: body, W: ansi.StringWidth(body), Bg: bg}
}

// RenderSegs draws a row of segs: one line, or three when tall (the
// button line between two empty ones).
func RenderSegs(segs []Seg, tall bool) []string {
	var mid strings.Builder
	w := 0
	for _, sg := range segs {
		mid.WriteString(sg.Text)
		w += sg.W
	}
	if !tall {
		return []string{mid.String()}
	}
	return []string{"", mid.String(), ""}
}

// Chip is an icon in a colour and a quiet label: the unit of metadata.
func Chip(icon string, c color.Color, text string) string {
	if text == "" {
		return lipgloss.NewStyle().Foreground(c).Render(icon)
	}
	return lipgloss.NewStyle().Foreground(c).Render(icon) + " " + theme.Styles.Subtle.Render(text)
}

// JoinChips lays chips out with a gap, wrapping onto new lines at width.
// indent is put before every line.
func JoinChips(chips []string, width int, indent string) string {
	var lines []string
	line, w := indent, ansi.StringWidth(indent)
	first := true
	for _, c := range chips {
		if c == "" {
			continue
		}
		cw := ansi.StringWidth(c)
		gap := 3
		if first {
			gap = 0
		}
		if !first && w+gap+cw > width {
			lines = append(lines, line)
			line, w, gap = indent, ansi.StringWidth(indent), 0
		}
		line += strings.Repeat(" ", gap) + c
		w += gap + cw
		first = false
	}
	return strings.Join(append(lines, line), "\n")
}

// PriorityBars draws a priority on Linear's scale: 1 urgent as a red mark,
// 2 high, 3 medium and 4 low as three, two and one lit bars, 0 as nothing.
// It is always three cells wide so titles line up.
func PriorityBars(p int) string {
	if p == 1 {
		return lipgloss.NewStyle().Foreground(theme.Pal.Red).Render(PadRight(icons.Current.Urgent, 3))
	}
	if p < 2 || p > 4 {
		return "   "
	}
	on := 5 - p // high 3, medium 2, low 1
	bars := []string{"▂", "▄", "▆"}
	var b strings.Builder
	for i, bar := range bars {
		c := theme.Pal.Muted
		if i < on {
			c = theme.Pal.Subtle
			if p == 2 {
				c = theme.Pal.Orange
			}
		}
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render(bar))
	}
	return b.String()
}

// ProgressBar is a thin bar w cells wide, filled to frac.
func ProgressBar(frac float64, w int, c color.Color) string {
	frac = max(0, min(1, frac))
	full := int(frac*float64(w) + 0.5)
	return lipgloss.NewStyle().Foreground(c).Render(strings.Repeat("━", full)) +
		lipgloss.NewStyle().Foreground(theme.Pal.SurfaceHi).Render(strings.Repeat("━", w-full))
}

// SectionHeader is a group title with its count and a rule to the edge.
func SectionHeader(icon string, c color.Color, title string, count, width int) string {
	return SectionTitle(icon, c, title+" "+theme.Styles.Muted.Render(strconv.Itoa(count)), width)
}

// SectionTitle is a title with a rule to the edge, without a count.
func SectionTitle(icon string, c color.Color, title string, width int) string {
	head := lipgloss.NewStyle().Foreground(c).Render(icon) + " " +
		lipgloss.NewStyle().Foreground(c).Bold(true).Render(title)
	rest := width - 2 - ansi.StringWidth(head) - 1
	if rest < 3 {
		return "  " + head
	}
	return "  " + head + " " + lipgloss.NewStyle().Foreground(theme.Pal.SurfaceHi).Render(strings.Repeat("─", rest))
}

// Spread puts left and right on one line of width, right-aligned, cutting
// left when they don't fit.
func Spread(left, right string, width int) string {
	rw := ansi.StringWidth(right)
	lw := width - rw - 1
	if lw < 8 {
		return ansi.Truncate(left, width, "…")
	}
	left = ansi.Truncate(left, lw, "…")
	return left + strings.Repeat(" ", max(width-ansi.StringWidth(left)-rw, 1)) + right
}

// LinkButton is a button that is also an OSC 8 hyperlink: tapping it
// opens url on the device that was tapped (on iOS, the app that claims
// the link, else Safari; the Mac's browser), even with mouse reporting on.
func LinkButton(url, label string, pressed bool) string {
	bg := theme.Pal.SurfaceHi
	if pressed {
		bg = theme.Pal.AccentSoft
	}
	return ansi.SetHyperlink(url) + Btn("", label, bg, false).Text + ansi.ResetHyperlink()
}

// PressDelay is how long a tapped button shows as pressed before its
// action runs, so the flash is seen even when the button goes away.
const PressDelay = 120 * time.Millisecond

// PadRight pads s with spaces to w cells.
func PadRight(s string, w int) string { return s + strings.Repeat(" ", max(w-ansi.StringWidth(s), 0)) }

// PadLeft right-aligns s in w cells.
func PadLeft(s string, w int) string { return strings.Repeat(" ", max(w-ansi.StringWidth(s), 0)) + s }

// FillBg paints a rendered line's background to width, through the
// resets of the styles inside it, so a card reads as one surface.
func FillBg(line string, width int, bg color.Color) string {
	seq := ansi.Style{}.BackgroundColor(bg).String()
	line += strings.Repeat(" ", max(width-ansi.StringWidth(line), 0))
	return seq + strings.ReplaceAll(line, "\x1b[m", "\x1b[m"+seq) + "\x1b[m"
}

// Dot is a coloured bullet for statuses and labels.
func Dot(c string) string {
	return lipgloss.NewStyle().Foreground(theme.Hex(c)).Render(icons.Current.Dot)
}
