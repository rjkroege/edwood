package theme

import (
	"image"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/frame"
)

// ColorSpec describes a single colour entry.
// If Mix is non-zero the colour is produced by AllocImageMix(Color, Mix);
// otherwise AllocImage is used with Color.
type ColorSpec struct {
	Color draw.Color
	Mix   draw.Color
}

func solid(c draw.Color) ColorSpec    { return ColorSpec{Color: c} }
func mixed(c, m draw.Color) ColorSpec { return ColorSpec{Color: c, Mix: m} }

// FramePalette holds the six colour slots used by a single frame
// (tag strip or text body).  The imgs cache is populated lazily on the
// first call to Colors; subsequent calls with the same display are free.
type FramePalette struct {
	Back  ColorSpec // background
	High  ColorSpec // selection highlight
	Bord  ColorSpec // border / scrollbar
	Text  ColorSpec // foreground text
	HText ColorSpec // highlighted text
	Tick  ColorSpec // insertion-point tick

	// cached allocated images — valid when display != nil
	display draw.Display
	imgs    [frame.NumColours]draw.Image
}

// AllocOne allocates one image from a ColorSpec against display.
func AllocOne(display draw.Display, cs ColorSpec) draw.Image {
	if cs.Mix != 0 {
		return display.AllocImageMix(cs.Color, cs.Mix)
	}
	img, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, cs.Color)
	return img
}

// Colors returns the [frame.NumColours]draw.Image array for this palette,
// allocating images on the first call and returning the cached result
// on subsequent calls with the same display.
func (fp *FramePalette) Colors(display draw.Display) [frame.NumColours]draw.Image {
	if fp.display == display && fp.imgs[frame.ColBack] != nil {
		return fp.imgs
	}
	fp.display = display
	fp.imgs[frame.ColBack] = AllocOne(display, fp.Back)
	fp.imgs[frame.ColHigh] = AllocOne(display, fp.High)
	fp.imgs[frame.ColBord] = AllocOne(display, fp.Bord)
	fp.imgs[frame.ColText] = AllocOne(display, fp.Text)
	fp.imgs[frame.ColHText] = AllocOne(display, fp.HText)
	fp.imgs[frame.ColTick] = AllocOne(display, fp.Tick)
	return fp.imgs
}

// UiPalette holds colours used for application chrome elements.
type UiPalette struct {
	ModButton ColorSpec // file-modified indicator button
	ColButton ColorSpec // column-colour button
	But2      ColorSpec // mouse-button-2 highlight
	But3      ColorSpec // mouse-button-3 highlight
}

// Palette holds the complete set of colours for one visual mode.
type Palette struct {
	Tag  FramePalette
	Text FramePalette
	Ui   UiPalette
}

// tagImg returns the image for the given slot from the Tag palette.
// Panics if Tag.Colors has not been called yet.
func (p *Palette) tagImg(slot int) draw.Image {
	if p.Tag.display == nil {
		panic("theme: Palette.Tag not initialized; call Tag.Colors first")
	}
	return p.Tag.Colors(p.Tag.display)[slot]
}

// textImg returns the image for the given slot from the Text palette.
// Panics if Text.Colors has not been called yet.
func (p *Palette) textImg(slot int) draw.Image {
	if p.Text.display == nil {
		panic("theme: Palette.Text not initialized; call Text.Colors first")
	}
	return p.Text.Colors(p.Text.display)[slot]
}

func (p *Palette) TagBack() draw.Image  { return p.tagImg(frame.ColBack) }
func (p *Palette) TagHigh() draw.Image  { return p.tagImg(frame.ColHigh) }
func (p *Palette) TagBord() draw.Image  { return p.tagImg(frame.ColBord) }
func (p *Palette) TagText() draw.Image  { return p.tagImg(frame.ColText) }
func (p *Palette) TagHText() draw.Image { return p.tagImg(frame.ColHText) }
func (p *Palette) TagTick() draw.Image  { return p.tagImg(frame.ColTick) }

func (p *Palette) TextBack() draw.Image  { return p.textImg(frame.ColBack) }
func (p *Palette) TextHigh() draw.Image  { return p.textImg(frame.ColHigh) }
func (p *Palette) TextBord() draw.Image  { return p.textImg(frame.ColBord) }
func (p *Palette) TextText() draw.Image  { return p.textImg(frame.ColText) }
func (p *Palette) TextHText() draw.Image { return p.textImg(frame.ColHText) }
func (p *Palette) TextTick() draw.Image  { return p.textImg(frame.ColTick) }

// Light is the built-in light-mode palette.
var Light = Palette{
	Tag: FramePalette{
		Back:  mixed(draw.Palebluegreen, draw.White),
		High:  solid(draw.Palegreygreen),
		Bord:  solid(draw.Purpleblue),
		Text:  solid(draw.Black),
		HText: solid(draw.Black),
		Tick:  solid(draw.Black),
	},
	Text: FramePalette{
		Back:  mixed(draw.Paleyellow, draw.White),
		High:  solid(draw.Darkyellow),
		Bord:  solid(draw.Yellowgreen),
		Text:  solid(draw.Black),
		HText: solid(draw.Black),
		Tick:  solid(draw.Black),
	},
	Ui: UiPalette{
		ModButton: solid(draw.Medblue),
		ColButton: solid(draw.Purpleblue),
		But2:      solid(0xAA0000FF),
		But3:      solid(0x006600FF),
	},
}

// Dark is the built-in dark (Vampira) mode palette.
var Dark = Palette{
	Tag: FramePalette{
		Back:  solid(0x333333FF),
		High:  solid(0x888888FF),
		Bord:  solid(0x888888FF),
		Text:  solid(draw.White),
		HText: solid(draw.White),
		Tick:  solid(draw.White),
	},
	Text: FramePalette{
		Back:  solid(0x222222FF),
		High:  solid(0x444444FF),
		Bord:  solid(0x888888FF),
		Text:  solid(draw.White),
		HText: solid(draw.White),
		Tick:  solid(draw.White),
	},
	Ui: UiPalette{
		ModButton: solid(0x666666FF),
		ColButton: solid(0x666666FF),
		But2:      solid(0xAA0000FF),
		But3:      solid(0x006600FF),
	},
}
