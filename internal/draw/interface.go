package draw

import "image"

type Display interface {
	ScreenImage() Image
	White() Image
	Black() Image
	Opaque() Image
	Transparent() Image

	InitKeyboard() *Keyboardctl
	InitMouse() *Mousectl
	OpenFont(name string) (Font, error)
	AllocImage(r image.Rectangle, pix Pix, repl bool, val Color) (Image, error)
	AllocImageMix(color1, color3 Color) Image
	Attach(ref int) error
	Flush() error
	ScaleSize(n int) int
	ReadSnarf(buf []byte) (int, int, error)
	WriteSnarf(data []byte) error
	MoveTo(pt image.Point) error
	SetCursor(c *Cursor) error
}

type Image interface {
	Display() Display
	Pix() Pix
	R() image.Rectangle

	Draw(r image.Rectangle, src, mask Image, p1 image.Point)
	Border(r image.Rectangle, n int, color Image, sp image.Point)
	Bytes(pt image.Point, src Image, sp image.Point, f Font, b []byte) image.Point
	Free() error
}

type Font interface {
	Name() string
	Height() int
	BytesWidth(b []byte) int
	RunesWidth(r []rune) int
	StringWidth(s string) int
}

// displayImpl implements the Display interface.
type displayImpl struct {
	*drawDisplay
}

var _ = Display((*displayImpl)(nil))

func (d *displayImpl) ScreenImage() Image { return &imageImpl{d.drawDisplay.ScreenImage} }
func (d *displayImpl) White() Image       { return &imageImpl{d.drawDisplay.White} }
func (d *displayImpl) Black() Image       { return &imageImpl{d.drawDisplay.Black} }
func (d *displayImpl) Opaque() Image      { return &imageImpl{d.drawDisplay.Opaque} }
func (d *displayImpl) Transparent() Image { return &imageImpl{d.drawDisplay.Transparent} }

func (d *displayImpl) OpenFont(name string) (Font, error) {
	f, err := d.drawDisplay.OpenFont(name)
	if err != nil {
		return nil, err
	}
	return &fontImpl{f}, nil
}

func (d *displayImpl) AllocImage(r image.Rectangle, pix Pix, repl bool, val Color) (Image, error) {
	i, err := d.drawDisplay.AllocImage(r, pix, repl, val)
	if err != nil {
		return nil, err
	}
	return &imageImpl{i}, nil
}

func (d *displayImpl) AllocImageMix(color1, color3 Color) Image {
	return &imageImpl{d.drawDisplay.AllocImageMix(color1, color3)}
}

// imageImpl implements the Image interface.
type imageImpl struct {
	*drawImage
}

var _ = Image((*imageImpl)(nil))

func (dst *imageImpl) Display() Display   { return &displayImpl{dst.drawImage.Display} }
func (dst *imageImpl) Pix() Pix           { return dst.drawImage.Pix }
func (dst *imageImpl) R() image.Rectangle { return dst.drawImage.R }

func (dst *imageImpl) Draw(r image.Rectangle, src, mask Image, p1 image.Point) {
	dst.drawImage.Draw(r, toDrawImage(src), toDrawImage(mask), p1)
}

func (dst *imageImpl) Border(r image.Rectangle, n int, color Image, sp image.Point) {
	dst.drawImage.Border(r, n, toDrawImage(color), sp)
}

func (dst *imageImpl) Bytes(pt image.Point, src Image, sp image.Point, f Font, b []byte) image.Point {
	return dst.drawImage.Bytes(pt, toDrawImage(src), sp, f.(*fontImpl).drawFont, b)
}

func toDrawImage(i Image) *drawImage {
	if i == nil {
		return nil
	}
	return i.(*imageImpl).drawImage
}

type fontImpl struct {
	*drawFont
}

func (f *fontImpl) Name() string { return f.drawFont.Name }
func (f *fontImpl) Height() int  { return f.drawFont.Height }
