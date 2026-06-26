package theme

import "github.com/rjkroege/edwood/draw"

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
// (tag strip or text body).
type FramePalette struct {
	Back  ColorSpec // background
	High  ColorSpec // selection highlight
	Bord  ColorSpec // border / scrollbar
	Text  ColorSpec // foreground text
	HText ColorSpec // highlighted text
	Tick  ColorSpec // insertion-point tick
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
