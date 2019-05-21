package edwoodtest

import (
	"image"

	"github.com/rjkroege/edwood/internal/draw"
)

type mockDisplay struct{}

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
func (d *mockDisplay) OpenFont(name string) (draw.Font, error) { return newFont(), nil }
func (d *mockDisplay) AllocImage(r image.Rectangle, pix draw.Pix, repl bool, val draw.Color) (draw.Image, error) {
	return &mockImage{r: r}, nil
}
func (d *mockDisplay) AllocImageMix(color1, color3 draw.Color) draw.Image {
	return NewImage(image.Rectangle{})
}
func (d *mockDisplay) Attach(ref int) error                   { return nil }
func (d *mockDisplay) Flush() error                           { return nil }
func (d *mockDisplay) ScaleSize(n int) int                    { return 0 }
func (d *mockDisplay) ReadSnarf(buf []byte) (int, int, error) { return 0, 0, nil }
func (d *mockDisplay) WriteSnarf(data []byte) error           { return nil }
func (d *mockDisplay) MoveTo(pt image.Point) error            { return nil }
func (d *mockDisplay) SetCursor(c *draw.Cursor) error         { return nil }

type mockImage struct {
	r image.Rectangle
}

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

type mockFont struct{}

func newFont() draw.Font { return &mockFont{} }

func (f *mockFont) Name() string             { return "" }
func (f *mockFont) Height() int              { return 16 }
func (f *mockFont) BytesWidth(b []byte) int  { return f.RunesWidth([]rune(string(b))) }
func (f *mockFont) RunesWidth(r []rune) int  { return 10 * len(r) }
func (f *mockFont) StringWidth(s string) int { return f.RunesWidth([]rune(s)) }
