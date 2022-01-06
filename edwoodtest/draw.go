// Package edwoodtest contains utility functions that help with testing Edwood.
package edwoodtest

import (
	"errors"
	"fmt"
	"image"
	"sync"
	"unicode/utf8"

	"github.com/rjkroege/edwood/draw"
)

var _ = draw.Display((*mockDisplay)(nil))

// GettableDrawOps display implementations can provide a list of the
// executed draw ops.
type GettableDrawOps interface {
	DrawOps() []string
	Clear()
}

// mockDisplay implements draw.Display.
type mockDisplay struct {
	snarfbuf []byte
	mu       sync.Mutex
	drawops  []string
}

// NewDisplay returns a mock draw.Display.
func NewDisplay() draw.Display {
	return &mockDisplay{}
}

func (d *mockDisplay) ScreenImage() draw.Image {
	return newimageimpl(d, "screen-800x600", image.Rect(0, 0, 800, 600))
}

func (d *mockDisplay) White() draw.Image  { return newimageimpl(d, "white", image.Rectangle{}) }
func (d *mockDisplay) Black() draw.Image  { return newimageimpl(d, "black", image.Rectangle{}) }
func (d *mockDisplay) Opaque() draw.Image { return newimageimpl(d, "opaque", image.Rectangle{}) }
func (d *mockDisplay) Transparent() draw.Image {
	return newimageimpl(d, "transparent", image.Rectangle{})
}
func (d *mockDisplay) InitKeyboard() *draw.Keyboardctl { return nil }
func (d *mockDisplay) InitMouse() *draw.Mousectl       { return nil }

// TODO(rjk): Need to increase fidelity here.
func (d *mockDisplay) OpenFont(name string) (draw.Font, error) { return NewFont(13, 10), nil }

func (d *mockDisplay) AllocImage(r image.Rectangle, pix draw.Pix, repl bool, val draw.Color) (draw.Image, error) {
	name := NiceColourName(val)
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
func (d *mockDisplay) ScaleSize(n int) int  { return 0 }

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

	op := fmt.Sprintf("%s <- draw-chars %q atpoint: %v font: %s fill: %s sp: %v",
		i.n,
		string(b),
		pt,
		f.Name(),
		srcname,
		sp,
	)
	i.d.drawops = append(i.d.drawops, op)

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

func (f *mockFont) Name() string             { return "/lib/font/edwood.font" }
func (f *mockFont) Height() int              { return f.height }
func (f *mockFont) BytesWidth(b []byte) int  { return f.width * utf8.RuneCount(b) }
func (f *mockFont) RunesWidth(r []rune) int  { return f.width * len(r) }
func (f *mockFont) StringWidth(s string) int { return f.width * utf8.RuneCountInString(s) }
