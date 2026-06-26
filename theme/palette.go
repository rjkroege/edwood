package theme

import "github.com/rjkroege/edwood/draw"

// ColIndex selects a colour entry from a Palette.
type ColIndex int

const (
	// Tag strip frame colours — map to frame.Col* slots.
	TagBack  ColIndex = iota
	TagHigh
	TagBord
	TagText
	TagHText
	TagTick
	// Text body frame colours — map to frame.Col* slots.
	TextBack
	TextHigh
	TextBord
	TextText
	TextHText
	TextTick
	// Application chrome.
	ModButton
	ColButton
	But2
	But3
	NumCols
)

// ColorSpec describes a single colour entry.
// If Mix is non-zero the colour is produced by AllocImageMix(Color, Mix);
// otherwise AllocImage is used with Color.
type ColorSpec struct {
	Color draw.Color
	Mix   draw.Color
}

func solid(c draw.Color) ColorSpec          { return ColorSpec{Color: c} }
func mixed(c, m draw.Color) ColorSpec       { return ColorSpec{Color: c, Mix: m} }

// Palette holds the complete set of colours for one visual mode.
type Palette [NumCols]ColorSpec

// Light is the built-in light-mode palette.
var Light = Palette{
	TagBack:  mixed(draw.Palebluegreen, draw.White),
	TagHigh:  solid(draw.Palegreygreen),
	TagBord:  solid(draw.Purpleblue),
	TagText:  solid(draw.Black),
	TagHText: solid(draw.Black),
	TagTick:  solid(draw.Black),

	TextBack:  mixed(draw.Paleyellow, draw.White),
	TextHigh:  solid(draw.Darkyellow),
	TextBord:  solid(draw.Yellowgreen),
	TextText:  solid(draw.Black),
	TextHText: solid(draw.Black),
	TextTick:  solid(draw.Black),

	ModButton: solid(draw.Medblue),
	ColButton: solid(draw.Purpleblue),
	But2:      solid(0xAA0000FF),
	But3:      solid(0x006600FF),
}

// Dark is the built-in dark (Vampira) mode palette.
var Dark = Palette{
	TagBack:  solid(0x333333FF),
	TagHigh:  solid(0x888888FF),
	TagBord:  solid(0x888888FF),
	TagText:  solid(draw.White),
	TagHText: solid(draw.White),
	TagTick:  solid(draw.White),

	TextBack:  solid(0x222222FF),
	TextHigh:  solid(0x444444FF),
	TextBord:  solid(0x888888FF),
	TextText:  solid(draw.White),
	TextHText: solid(draw.White),
	TextTick:  solid(draw.White),

	ModButton: solid(0x666666FF),
	ColButton: solid(0x666666FF),
	But2:      solid(0xAA0000FF),
	But3:      solid(0x006600FF),
}
