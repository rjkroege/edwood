// Package edwoodtest contains utility functions that help with testing Edwood.
package edwoodtest

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"image/png"
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

	// ScreenImageAsPNG writes the current pixel state of the screen image
	// as a PNG. Only meaningful when the display was created with
	// NewDisplayWithDPI; returns an error otherwise.
	ScreenImageAsPNG(w io.Writer) error
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

	// dpi is non-zero only for displays created with NewDisplayWithDPI.
	// A zero value means the non-rendering (string-recording only) path.
	dpi int

	// pixscreen is the backing pixel buffer for the screen image.
	// Non-nil only when dpi > 0.
	pixscreen *image.RGBA
}

// NewDisplay returns a mock draw.Display where visulizations of the output are w.r.t. rectangle rectofi.
// Set rectofi to control SVG output.
func NewDisplay(rectofi image.Rectangle) draw.Display {
	md := &mockDisplay{
		rectofi: rectofi,
	}
	md.screenimage = newimageimpl(md, "screen-800x600", draw.Notacolor, image.Rect(0, 0, 800, 600))
	md.svgdrawops = append(md.svgdrawops, boundingboxsvg(0, rectofi))
	md.annotations = append(md.annotations, fmt.Sprintf("target rect %v", rectofi))
	return md
}

// luridPink is the initial fill colour of a rendering display's pixel buffer.
// Any pixel that is never written by Draw or Bytes will remain this colour,
// making rendering gaps immediately obvious in PNG output.
var luridPink = color.RGBA{R: 0xFF, G: 0x00, B: 0xCC, A: 0xFF}

// NewDisplayWithDPI returns a mock draw.Display that renders to a real
// *image.RGBA in addition to recording draw operations as strings. dpi
// controls ScaleSize; 100 is "1:1 logical-to-physical". rectofi controls SVG
// output as in NewDisplay. Call ScreenImageAsPNG on the returned
// GettableDrawOps to obtain a PNG of the current screen state.
func NewDisplayWithDPI(rectofi image.Rectangle, dpi int) draw.Display {
	const w, h = 800, 600
	px := image.NewRGBA(image.Rect(0, 0, w, h))
	imagedraw.Draw(px, px.Bounds(), image.NewUniform(luridPink), image.Point{}, imagedraw.Src)

	md := &mockDisplay{
		rectofi:   rectofi,
		dpi:       dpi,
		pixscreen: px,
	}
	md.screenimage = &mockImage{
		d: md,
		r: image.Rect(0, 0, w, h),
		n: "screen-800x600",
		c: draw.Notacolor,
		m: px,
	}
	md.svgdrawops = append(md.svgdrawops, boundingboxsvg(0, rectofi))
	md.annotations = append(md.annotations, fmt.Sprintf("target rect %v", rectofi))
	return md
}

// drawColorToRGBA converts a draw.Color (0xRRGGBBAA) to color.RGBA.
func drawColorToRGBA(c draw.Color) color.RGBA {
	return color.RGBA{
		R: uint8(c >> 24),
		G: uint8(c >> 16),
		B: uint8(c >> 8),
		A: uint8(c),
	}
}

// solidImage returns an image.Uniform for the given draw.Color when the
// display is in rendering mode, or nil otherwise.
func (d *mockDisplay) solidImage(c draw.Color) image.Image {
	if d.pixscreen == nil {
		return nil
	}
	return image.NewUniform(drawColorToRGBA(c))
}

func (d *mockDisplay) ScreenImage() draw.Image {
	return d.screenimage
}

func (d *mockDisplay) White() draw.Image {
	return &mockImage{d: d, n: "white", c: draw.White, r: image.Rect(0, 0, 1, 1), m: d.solidImage(draw.White)}
}
func (d *mockDisplay) Black() draw.Image {
	return &mockImage{d: d, n: "black", c: draw.Black, r: image.Rect(0, 0, 1, 1), m: d.solidImage(draw.Black)}
}
func (d *mockDisplay) Opaque() draw.Image {
	return &mockImage{d: d, n: "opaque", c: draw.Opaque, r: image.Rect(0, 0, 1, 1), m: d.solidImage(draw.Opaque)}
}
func (d *mockDisplay) Transparent() draw.Image {
	return &mockImage{d: d, n: "transparent", c: draw.Transparent, r: image.Rect(0, 0, 1, 1), m: d.solidImage(draw.Transparent)}
}
func (d *mockDisplay) InitKeyboard() *draw.Keyboardctl { return &draw.Keyboardctl{} }
func (d *mockDisplay) InitMouse() *draw.Mousectl       { return &draw.Mousectl{} }

// TODO(rjk): Support a richer variety of fonts with better metrics.
// NB: to make the recorded ops easier to read, I provide them in
// character multiples based on the fixed font metrics here.
func (d *mockDisplay) OpenFont(name string) (draw.Font, error) { return NewFont(fwidth, fheight), nil }

func (d *mockDisplay) AllocImage(r image.Rectangle, pix draw.Pix, repl bool, val draw.Color) (draw.Image, error) {
	mi := &mockImage{
		d:    d,
		r:    r,
		c:    val,
		repl: repl,
	}
	if d.pixscreen != nil {
		c := drawColorToRGBA(val)
		if repl && r == image.Rect(0, 0, 1, 1) {
			mi.m = image.NewUniform(c)
		} else {
			px := image.NewRGBA(r)
			imagedraw.Draw(px, px.Bounds(), image.NewUniform(c), image.Point{}, imagedraw.Src)
			mi.m = px
		}
	}
	return mi, nil
}

func (d *mockDisplay) AllocImageMix(color1, color3 draw.Color) draw.Image {
	c1 := draw.WithAlpha(color1, 0x3f) >> 8
	c3 := draw.WithAlpha(color3, 0xbf) >> 8
	c := ((c1 + c3) << 8) | 0xff

	mi := &mockImage{
		d:    d,
		r:    image.Rect(0, 0, 1, 1),
		repl: true,
		c:    c,
	}
	if d.pixscreen != nil {
		mi.m = image.NewUniform(drawColorToRGBA(c))
	}
	return mi
}

func (d *mockDisplay) Attach(ref int) error { return nil }
func (d *mockDisplay) Flush() error         { return nil }

func (d *mockDisplay) ScaleSize(n int) int {
	if d.dpi == 0 {
		// Preserve existing non-rendering behaviour.
		return 1
	}
	if d.dpi <= 100 {
		return n
	}
	return (n*d.dpi + 50) / 100
}

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
func (d *mockDisplay) DrawOps() []string { return d.drawops }
func (d *mockDisplay) Clear()             { d.drawops = nil }

func (d *mockDisplay) SVGDrawOps(w io.Writer) error {
	return singlesvgfile(w, d.svgdrawops, d.annotations, d.rectofi)
}

func (d *mockDisplay) ScreenImageAsPNG(w io.Writer) error {
	if d.pixscreen == nil {
		return errors.New("ScreenImageAsPNG: display not created with NewDisplayWithDPI")
	}
	return png.Encode(w, d.pixscreen)
}

var _ = draw.Image((*mockImage)(nil))

// mockImage implements draw.Image.
type mockImage struct {
	r    image.Rectangle
	d    *mockDisplay
	n    string
	c    draw.Color
	repl bool

	// m is the pixel backing for this image. Non-nil only in displays created
	// with NewDisplayWithDPI. For the screen image it is *image.RGBA; for solid
	// colours it is *image.Uniform.
	m image.Image
}

// newimage creates a new mockImage. Use Notacolor for the situation
// where the name of the image takes precedence.
func newimageimpl(d *mockDisplay, name string, c draw.Color, r image.Rectangle) draw.Image {
	return &mockImage{
		r: r,
		d: d,
		c: c,
		n: name,
	}
}

// NewImage returns a mock draw.Image with the given bounds.
func NewImage(display draw.Display, name string, r image.Rectangle) draw.Image {
	d := display.(*mockDisplay)
	return newimageimpl(d, name, draw.Notacolor, r)
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
		srcname = msrc.N()
	}
	maskname := "nil"
	if mmask, ok := src.(*mockImage); ok {
		maskname = mmask.N()
	}

	op := fmt.Sprintf("%s <- draw r: %v src: %s mask %s p1: %v",
		i.n,
		r,
		srcname,
		maskname,
		p1,
	)

	// It's arguable that my logic to separate blit from fill is slightly
	// specious. The actual draw API deosn't (rightly) differentiate between
	// these and the distinction that I'm creating is only to make nicer test
	// output.
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
	case src != nil && i.r.Dx() > 0 && i.r.Dy() > 0 && maskname == srcname:
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
				src,
			))
			i.d.annotations = append(i.d.annotations, op)
		}
	}
	i.d.drawops = append(i.d.drawops, op)

	// Pixel-rendering path: only when this image has an RGBA backing.
	dstRGBA, dstOK := i.m.(*image.RGBA)
	if !dstOK {
		return
	}
	msrc, srcOK := src.(*mockImage)
	if !srcOK || msrc.m == nil {
		return
	}
	if mask == nil {
		imagedraw.Draw(dstRGBA, r, msrc.m, p1, imagedraw.Src)
	} else {
		mmask, maskOK := mask.(*mockImage)
		if maskOK && mmask.m != nil {
			imagedraw.DrawMask(dstRGBA, r, msrc.m, p1, mmask.m, p1, imagedraw.Over)
		} else {
			imagedraw.Draw(dstRGBA, r, msrc.m, p1, imagedraw.Src)
		}
	}
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

	// Pixel-rendering path: fill the bounding box of the glyph run with the
	// source colour. This is Option A from the proposal — no real font face
	// needed, platform-independent, deterministic. The rectangle geometry
	// matches what the fixed-width mockFont reports, so positions are correct.
	if dstRGBA, ok := i.m.(*image.RGBA); ok {
		if msrc, ok := src.(*mockImage); ok && msrc.m != nil {
			box := image.Rectangle{
				Min: pt,
				Max: pt.Add(image.Pt(f.BytesWidth(b), f.Height())),
			}
			imagedraw.Draw(dstRGBA, box, msrc.m, image.Point{}, imagedraw.Src)
		}
	}

	// TODO(rjk): This assumes fixed width. Consider generalizing.
	return pt.Add(image.Pt(f.BytesWidth(b), 0))
}

func (i *mockImage) Free() error { return nil }

// N returns a nicename for the image colour.
func (i *mockImage) N() string {
	name := i.n
	if i.c != draw.Notacolor {
		name = fmt.Sprintf("%s-%v", NiceColourName(i.c), i.r)
	}

	if i.repl {
		name += ",tiled"
	}
	return name
}

func (i *mockImage) HtmlString() string {
	if i.c == draw.Notacolor {
		return "white"
	}
	return fmt.Sprintf("#%x", i.c>>8)
}

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
