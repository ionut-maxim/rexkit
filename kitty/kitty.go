// Package kitty draws images with the kitty graphics protocol, using
// Unicode placeholders: the image is sent once with a virtual placement,
// then drawn by printing placeholder cells (U+10EEEE with row and column
// diacritics, coloured with the image id). The cells are ordinary text, so
// they scroll and clip with the rest of a Bubble Tea view.
//
// See https://sw.kovidgoyal.net/kitty/graphics-protocol/#unicode-placeholders
package kitty

import (
	"bytes"
	"fmt"
	"image"
	"math/rand/v2"
	"strings"
	"sync/atomic"

	// Decoders for the common screenshot formats.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// QueryID is the image id used by the support query.
const QueryID = 31

// Query asks the terminal whether it supports kitty graphics, followed by a
// primary device attributes request: a terminal that answers DA1 without an
// OK first has no support.
var Query = ansi.KittyGraphics([]byte("AAAA"), "i=31", "s=1", "v=1", "a=q", "t=d", "f=24") +
	ansi.RequestPrimaryDeviceAttributes

// Cell is the assumed size of a terminal cell in pixels. Only the ratio
// matters: it keeps images from looking stretched.
var Cell = struct{ W, H int }{W: 10, H: 20}

// Fit returns how many columns and rows an image of w×h pixels takes when
// shown at most maxCols wide and maxRows high, never scaled up.
func Fit(w, h, maxCols, maxRows int) (cols, rows int) {
	if w <= 0 || h <= 0 || maxCols <= 0 || maxRows <= 0 {
		return 0, 0
	}
	cols = min(maxCols, (w+Cell.W-1)/Cell.W)
	rows = (cols*Cell.W*h/w + Cell.H - 1) / Cell.H
	if rows > maxRows {
		rows = maxRows
		cols = max(1, rows*Cell.H*w/h/Cell.W)
	}
	return max(cols, 1), max(rows, 1)
}

// Image is a decoded image ready to send.
type Image struct {
	ID     int
	Bounds image.Rectangle
	img    image.Image
}

// Decode decodes PNG, JPEG, GIF or WebP bytes.
func Decode(id int, data []byte) (*Image, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return &Image{ID: id, Bounds: img.Bounds(), img: img}, nil
}

// FromImage wraps an image that's already decoded, such as one drawn by
// the program.
func FromImage(id int, img image.Image) *Image {
	return &Image{ID: id, Bounds: img.Bounds(), img: img}
}

// Transmit returns the escape sequences that send the image, scaled down
// to cols×rows cells, and create its virtual placement. Sending it again
// with a new size replaces both.
func (im *Image) Transmit(cols, rows int) (string, error) { return im.transmit(cols, rows, true) }

// TransmitSharp is Transmit without scaling the pixels down: the terminal
// fits them into the cells itself, so an image drawn at a higher
// resolution than the reported cell size stays sharp on a denser screen.
func (im *Image) TransmitSharp(cols, rows int) (string, error) { return im.transmit(cols, rows, false) }

func (im *Image) transmit(cols, rows int, scale bool) (string, error) {
	src := im.img
	b := src.Bounds()
	tw, th := cols*Cell.W, rows*Cell.H
	if scale && (b.Dx() > tw || b.Dy() > th) {
		// Keep the aspect ratio inside the cell box.
		scale := min(float64(tw)/float64(b.Dx()), float64(th)/float64(b.Dy()))
		dst := image.NewRGBA(image.Rect(0, 0, max(1, int(float64(b.Dx())*scale)), max(1, int(float64(b.Dy())*scale))))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
		src = dst
	}
	var out strings.Builder
	err := kitty.EncodeGraphics(&out, src, &kitty.Options{
		Action: kitty.Transmit,
		ID:     im.ID,
		Format: kitty.RGBA,
		// Raw pixels need their size; the encoder doesn't add it.
		ImageWidth:   src.Bounds().Dx(),
		ImageHeight:  src.Bounds().Dy(),
		Compression:  kitty.Zlib,
		Transmission: kitty.Direct,
		Chunk:        true,
		Quiet:        2,
	})
	if err != nil {
		return "", fmt.Errorf("encode image: %w", err)
	}
	out.WriteString(ansi.KittyGraphics(nil,
		"a=p", fmt.Sprintf("i=%d", im.ID), "U=1", fmt.Sprintf("c=%d", cols), fmt.Sprintf("r=%d", rows), "q=2"))
	return out.String(), nil
}

// Delete returns the sequence that frees the image in the terminal.
func Delete(id int) string {
	return ansi.KittyGraphics(nil, "a=d", "d=I", fmt.Sprintf("i=%d", id), "q=2")
}

// Placeholders returns rows lines of cols placeholder cells for image id.
// The id travels in the 24-bit foreground colour.
func Placeholders(id, cols, rows int) string {
	fg := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", (id>>16)&0xff, (id>>8)&0xff, id&0xff)
	var b strings.Builder
	for r := range rows {
		if r > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(fg)
		for c := range cols {
			b.WriteRune(kitty.Placeholder)
			b.WriteRune(kitty.Diacritic(r))
			b.WriteRune(kitty.Diacritic(c))
		}
		b.WriteString("\x1b[39m")
	}
	return b.String()
}

// Image ids start at a random point so two copies of the TUI, or other
// programs, in the same terminal don't overwrite each other's images. The
// id is carried in a 24-bit colour, so it stays below 1<<24.
var nextID atomic.Int64

func init() { nextID.Store(int64(1<<20 + rand.IntN(1<<22))) }

// NewID is a fresh image id, unlikely to clash with another program's.
func NewID() int { return int(nextID.Add(1)) }
