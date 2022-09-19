// Package edwoodtest contains utility functions that help with testing Edwood.
package edwoodtest

import (
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/rjkroege/edwood/draw"
)

var _ = draw.Display((*mockDisplay)(nil))

const (
	fwidth  = 13
	fheight = 10
)

// GettableDrawOps display implementations can provide a list of the
// executed draw ops.
type GettableDrawOps interface {
	DrawOps() []string
	Clear()

	// SVGDrawOps writes the accumulated SVG format drawops to w where rect
	// is the area of interest for the drawops.
	SVGDrawOps(w io.Writer) error
}

// mockDisplay implements draw.Display.
type mockDisplay struct {
	snarfbuf []byte
	mu       sync.Mutex
	drawops  []string

	// TODO(rjk): This is essentially the same as drawops above. Except that
	// I have pruned the drawops array at various points. And that would mean
	// that it's not the same length as svgdrawops. This would be
	// unfortunate. So save extra stuff. Later, I can merge this and clean it
	// up once I have finished implementing the validation of the saved
	// testdata SVG out.
	annotations []string
	svgdrawops  []string
	screenimage draw.Image

	// roi is the rectangle of interest.
	rectofi image.Rectangle
}

// NewDisplay returns a mock draw.Display where visulizations of the output are w.r.t. rectangle rectofi.
// Set rectofi to control SVG output.
func NewDisplay(rectofi image.Rectangle) draw.Display {
	md := &mockDisplay{
		rectofi: rectofi,
	}
	md.screenimage = newimageimpl(md, "screen-800x600", image.Rect(0, 0, 800, 600))
	md.svgdrawops = append(md.svgdrawops, boundingboxsvg(0, rectofi))
	md.annotations = append(md.annotations, fmt.Sprintf("target rect %v", rectofi))
	return md
}

func (d *mockDisplay) ScreenImage() draw.Image {
	return d.screenimage
}

func (d *mockDisplay) White() draw.Image  { return newimageimpl(d, "white", image.Rectangle{}) }
func (d *mockDisplay) Black() draw.Image  { return newimageimpl(d, "black", image.Rectangle{}) }
func (d *mockDisplay) Opaque() draw.Image { return newimageimpl(d, "opaque", image.Rectangle{}) }
func (d *mockDisplay) Transparent() draw.Image {
	return newimageimpl(d, "transparent", image.Rectangle{})
}
func (d *mockDisplay) InitKeyboard() *draw.Keyboardctl { return &draw.Keyboardctl{} }
func (d *mockDisplay) InitMouse() *draw.Mousectl       { return &draw.Mousectl{} }

// TODO(rjk): Support a richer variety of fonts with better metrics.
// NB: to make the recorded ops easier to read, I provide them in
// character multiples based on the fixed font metrics here.
func (d *mockDisplay) OpenFont(name string) (draw.Font, error) { return NewFont(fwidth, fheight), nil }

func (d *mockDisplay) AllocImage(r image.Rectangle, pix draw.Pix, repl bool, val draw.Color) (draw.Image, error) {
	name := fmt.Sprintf("%s-%v", NiceColourName(val), r)
	if repl {
		name += ",tiled"
	}

	return &mockImage{
		d: d,
		r: r,
		n: name,
	}, nil
}

func (d *mockDisplay) AllocImageMix(color1, color3 draw.Color) draw.Image {
	name := fmt.Sprintf("mix(%s,%s)", NiceColourName(color1), NiceColourName(color3))
	return &mockImage{
		d: d,
		n: name,
	}
}

func (d *mockDisplay) Attach(ref int) error { return nil }
func (d *mockDisplay) Flush() error         { return nil }
func (d *mockDisplay) ScaleSize(n int) int  { return 1 }

// ReadSnarf reads the snarf buffer into buf, returning the number of bytes read,
// the total size of the snarf buffer (useful if buf is too short), and any
// error. No error is returned if there is no problem except for buf being too
// short.
func (d *mockDisplay) ReadSnarf(buf []byte) (int, int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	n := copy(buf, d.snarfbuf)
	if n < len(d.snarfbuf) {
		return n, len(d.snarfbuf), errors.New("short read")
	}
	return n, n, nil
}

// WriteSnarf writes the data to the snarf buffer.
func (d *mockDisplay) WriteSnarf(data []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.snarfbuf = make([]byte, len(data))
	copy(d.snarfbuf, data)
	return nil
}

func (d *mockDisplay) MoveTo(pt image.Point) error    { return nil }
func (d *mockDisplay) SetCursor(c *draw.Cursor) error { return nil }
func (d *mockDisplay) DrawOps() []string              { return d.drawops }
func (d *mockDisplay) Clear()                         { d.drawops = nil }

func (d *mockDisplay) SVGDrawOps(w io.Writer) error {
	return singlesvgfile(w, d.svgdrawops, d.annotations, d.rectofi)
}

var _ = draw.Image((*mockImage)(nil))

// mockImage implements draw.Image.
type mockImage struct {
	r image.Rectangle
	d *mockDisplay
	n string
}

func newimageimpl(d *mockDisplay, name string, r image.Rectangle) draw.Image {
	return &mockImage{
		r: r,
		n: name,
		d: d,
	}
}

// NewImage returns a mock draw.Image with the given bounds.
func NewImage(display draw.Display, name string, r image.Rectangle) draw.Image {
	d := display.(*mockDisplay)
	return newimageimpl(d, name, r)
}

func (i *mockImage) Display() draw.Display { return i.d }
func (i *mockImage) Pix() draw.Pix         { return 0 }
func (i *mockImage) R() image.Rectangle    { return i.r }

// rectochars returns positions in character units where that's possible.
func rectochars(r image.Rectangle) string {
	var sb strings.Builder

	sb.WriteString(pointochars(r.Min))
	sb.WriteRune(',')

	sb.WriteRune('[')

	if r.Dx()%fwidth == 0 {
		fmt.Fprintf(&sb, "%d", r.Dx()/fwidth)
	} else {
		sb.WriteRune('-')
	}
	sb.WriteRune(',')

	if r.Dy()%fheight == 0 {
		fmt.Fprintf(&sb, "%d", r.Dy()/fheight)
	} else {
		sb.WriteRune('-')
	}
	sb.WriteRune(']')
	return sb.String()
}

func pointochars(p image.Point) string {
	var sb strings.Builder

	sb.WriteRune('[')

	if (p.X-20)%fwidth == 0 {
		fmt.Fprintf(&sb, "%d", (p.X-20)/fwidth)
	} else {
		sb.WriteRune('-')
	}
	sb.WriteRune(',')

	if (p.Y-10)%fheight == 0 {
		fmt.Fprintf(&sb, "%d", (p.Y-10)/fheight)
	} else {
		sb.WriteRune('-')
	}
	sb.WriteRune(']')

	return sb.String()
}

func (i *mockImage) Draw(r image.Rectangle, src, mask draw.Image, p1 image.Point) {
	srcname := "nil"
	if msrc, ok := src.(*mockImage); ok {
		srcname = msrc.n
	}
	maskname := "nil"
	if mmask, ok := src.(*mockImage); ok {
		maskname = mmask.n
	}

	op := fmt.Sprintf("%s <- draw r: %v src: %s mask %s p1: %v",
		i.n,
		r,
		srcname,
		maskname,
		p1,
	)
	switch {
	case i.r.Dx() > 0 && i.r.Dy() > 0 && maskname == srcname && srcname == i.n:
		sr := r.Sub(r.Min).Add(p1)
		op = fmt.Sprintf("blit %v %s, to %v %s",
			sr, rectochars(sr),
			r, rectochars(r),
		)

		// TODO(rjk): If this is right, fold it out and make an improved case statement.
		if i.d.screenimage == i {
			// TODO(rjk): Why am I failing to filter out the unnecessary draws?
			// I'm getting a bunch of fills that are setting up colour?
			i.d.svgdrawops = append(i.d.svgdrawops, blitsvg(
				len(i.d.svgdrawops),
				sr,
				r.Min,
				blitspace+i.d.rectofi.Dx(),
			))
			i.d.annotations = append(i.d.annotations, op)
		}
	case src != nil && i.r.Dx() > 0 && i.r.Dy() > 0 && maskname == srcname && src.R().Dx() == 0 && src.R().Dy() == 0:
		op = fmt.Sprintf("fill %v %s",
			r, rectochars(r),
		)

		// TODO(rjk): If this is right, fold it out and make an improved case statement.
		if i.d.screenimage == i {
			// TODO(rjk): Why am I failing to filter out the unnecessary draws?
			// I'm getting a bunch of fills that are setting up colour?
			i.d.svgdrawops = append(i.d.svgdrawops, fillsvg(
				len(i.d.svgdrawops),
				r,
				i.d.rectofi,
			))
			i.d.annotations = append(i.d.annotations, op)
		}
	}
	i.d.drawops = append(i.d.drawops, op)
}

func (i *mockImage) Border(r image.Rectangle, n int, color draw.Image, sp image.Point) {
	colorname := "nil"
	if mcolor, ok := color.(*mockImage); ok {
		colorname = mcolor.n
	}

	op := fmt.Sprintf("%s <- border r: %v thick: %d color: %s sp: %v",
		i.n,
		r,
		n,
		colorname,
		sp,
	)
	i.d.drawops = append(i.d.drawops, op)
}

func (i *mockImage) Bytes(pt image.Point, src draw.Image, sp image.Point, f draw.Font, b []byte) image.Point {
	srcname := "nil"
	if msrc, ok := src.(*mockImage); ok {
		srcname = msrc.n
	}

	op := fmt.Sprintf("%s <- string %q atpoint: %v %s fill: %s",
		i.n,
		string(b),
		pt,
		pointochars(pt),
		srcname,
	)
	i.d.drawops = append(i.d.drawops, op)

	// TODO(rjk): Remove this duplication when I've switched to always using
	// the SVG path for baselines and such.
	shortop := fmt.Sprintf("string %q atpoint: %v %s fill: %s",
		string(b),
		pt,
		pointochars(pt),
		srcname,
	)

	i.d.svgdrawops = append(i.d.svgdrawops, bytessvg(len(i.d.svgdrawops), pt, b))
	i.d.annotations = append(i.d.annotations, shortop)

	// TODO(rjk): This assumes fixed width. Consider generalizing.
	return pt.Add(image.Pt(f.BytesWidth(b), 0))
}

func (i *mockImage) Free() error { return nil }

var _ = draw.Font((*mockFont)(nil))

// mockFont implements draw.Font and mocks as a fixed width font.
// TODO(rjk): Do we need to handle variable widths?
type mockFont struct {
	width, height int
}

// NewFont returns a draw.Font that mocks a fixed-width font.
func NewFont(width, height int) draw.Font {
	return &mockFont{
		width:  width,
		height: height,
	}
}

// const MockFontName = "/lib/font/bit/veryloverylongstringherengstringhere/euro.8.font"
const MockFontName = "/lib/font/bit/lucsans/euro.8.font"

func (f *mockFont) Name() string             { return Plan9FontPath(MockFontName) }
func (f *mockFont) Height() int              { return f.height }
func (f *mockFont) BytesWidth(b []byte) int  { return f.width * utf8.RuneCount(b) }
func (f *mockFont) RunesWidth(r []rune) int  { return f.width * len(r) }
func (f *mockFont) StringWidth(s string) int { return f.width * utf8.RuneCountInString(s) }

func Plan9FontPath(name string) string {
	const prefix = "/lib/font/bit"
	if strings.HasPrefix(name, prefix) {
		root := os.Getenv("PLAN9")
		if root == "" {
			root = "/usr/local/plan9"
		}
		return filepath.Join(root, "/font/", name[len(prefix):])
	}
	return name
}
