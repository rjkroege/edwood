// Package edwoodtest contains utility functions that help with testing Edwood.
package edwoodtest

import (
	"errors"
	"image"
	"sync"
	"unicode/utf8"

	"github.com/rjkroege/edwood/draw"
)

var _ = draw.Display((*mockDisplay)(nil))

// mockDisplay implements draw.Display.
type mockDisplay struct {
	snarfbuf []byte
	mu       sync.Mutex
}

// NewDisplay returns a mock draw.Display.
func NewDisplay() draw.Display {
	return &mockDisplay{}
}
func (d *mockDisplay) ScreenImage() draw.Image                 { return NewImage(image.Rect(0, 0, 800, 600)) }
func (d *mockDisplay) White() draw.Image                       { return NewImage(image.Rectangle{}) }
func (d *mockDisplay) Black() draw.Image                       { return NewImage(image.Rectangle{}) }
func (d *mockDisplay) Opaque() draw.Image                      { return NewImage(image.Rectangle{}) }
func (d *mockDisplay) Transparent() draw.Image                 { return NewImage(image.Rectangle{}) }
func (d *mockDisplay) InitKeyboard() *draw.Keyboardctl         { return nil }
func (d *mockDisplay) InitMouse() *draw.Mousectl               { return nil }
func (d *mockDisplay) OpenFont(name string) (draw.Font, error) { return NewFont(13, 10), nil }
func (d *mockDisplay) AllocImage(r image.Rectangle, pix draw.Pix, repl bool, val draw.Color) (draw.Image, error) {
	return &mockImage{r: r}, nil
}
func (d *mockDisplay) AllocImageMix(color1, color3 draw.Color) draw.Image {
	return NewImage(image.Rectangle{})
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

var _ = draw.Image((*mockImage)(nil))

// mockImage implements draw.Image.
type mockImage struct {
	r image.Rectangle
}

// NewImage returns a mock draw.Image with the given bounds.
func NewImage(r image.Rectangle) draw.Image {
	return &mockImage{r: r}
}
func (i *mockImage) Display() draw.Display                                             { return NewDisplay() }
func (i *mockImage) Pix() draw.Pix                                                     { return 0 }
func (i *mockImage) R() image.Rectangle                                                { return i.r }
func (i *mockImage) Draw(r image.Rectangle, src, mask draw.Image, p1 image.Point)      {}
func (i *mockImage) Border(r image.Rectangle, n int, color draw.Image, sp image.Point) {}
func (i *mockImage) Bytes(pt image.Point, src draw.Image, sp image.Point, f draw.Font, b []byte) image.Point {
	return image.Point{}
}
func (i *mockImage) Free() error { return nil }

var _ = draw.Font((*mockFont)(nil))

// mockFont implements draw.Font and mocks as a fixed width font.
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
