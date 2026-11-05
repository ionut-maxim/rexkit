// Package markdown renders GitHub-flavoured markdown for the terminal. Images are
// split out of the text so the UI can draw them itself (kitty graphics) or
// show them as links.
package markdown

import (
	"regexp"
	"strings"
)

// Image is a markdown image: ![Alt](URL).
type Image struct {
	Alt string
	URL string
}

// Segment is a run of markdown text or one image.
type Segment struct {
	Text  string
	Image *Image
}

var imageRE = regexp.MustCompile(`!\[([^\]]*)\]\(\s*<?([^)\s>]+)>?(?:\s+"[^"]*")?\s*\)`)

// Split cuts md into text and image segments, in order. Image syntax inside
// fenced code blocks is left as text.
func Split(md string) []Segment {
	var out []Segment
	var text strings.Builder
	flush := func() {
		if strings.TrimSpace(text.String()) != "" {
			out = append(out, Segment{Text: text.String()})
		}
		text.Reset()
	}

	fence := ""
	for _, line := range strings.SplitAfter(md, "\n") {
		trimmed := strings.TrimSpace(line)
		if fence != "" {
			text.WriteString(line)
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fence = trimmed[:3]
			text.WriteString(line)
			continue
		}
		rest := line
		for {
			loc := imageRE.FindStringSubmatchIndex(rest)
			if loc == nil {
				text.WriteString(rest)
				break
			}
			text.WriteString(rest[:loc[0]])
			flush()
			out = append(out, Segment{Image: &Image{Alt: rest[loc[2]:loc[3]], URL: rest[loc[4]:loc[5]]}})
			rest = rest[loc[1]:]
		}
	}
	flush()
	return out
}

// Images returns the images in md, in order.
func Images(md string) []Image {
	var out []Image
	for _, s := range Split(md) {
		if s.Image != nil {
			out = append(out, *s.Image)
		}
	}
	return out
}
