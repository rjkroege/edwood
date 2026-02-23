package theme

import (
	"github.com/rjkroege/edwood/draw"
)

type Palette struct {
	TagColBack   draw.Color
	TagColHigh   draw.Color
	TagColBord   draw.Color
	TagColText   draw.Color
	TagHText     draw.Color
	TextColBack  draw.Color
	TextColHigh  draw.Color
	TextColBord  draw.Color
	TextColText  draw.Color
	TextColHText draw.Color
	ModButton    draw.Color
	ColButton    draw.Color
	ButtonColor  draw.Color
	But2Col      draw.Color
	But3Col      draw.Color
	Background   draw.Color
	TickColor    draw.Color
}

var (
	darkMode bool
	current  Palette
)

var lightPalette = Palette{
	// Plan 9 defaults
	TagColBack:   draw.Palebluegreen,
	TagColHigh:   draw.Palegreygreen,
	TagColBord:   draw.Purpleblue,
	TagColText:   draw.Black,
	TagHText:     draw.Black,
	TextColBack:  draw.Paleyellow,
	TextColHigh:  draw.Darkyellow,
	TextColBord:  draw.Yellowgreen,
	TextColText:  draw.Black,
	TextColHText: draw.Black,
	ModButton:    0x222222FF,
	ColButton:    0x666666FF,
	ButtonColor:  draw.Black,
	But2Col:      0xAA0000FF,
	But3Col:      0x006600FF,
	Background:   draw.White,
	TickColor:    draw.Black,
}

var darkPalette = Palette{
	TagColBack:   0x333333FF,
	TagColHigh:   0x888888FF,
	TagColBord:   0x888888FF,
	TagColText:   0xEEEEEEFF,
	TagHText:     0xEEEEEEFF,
	TextColBack:  0x222222FF,
	TextColHigh:  0x444444FF,
	TextColBord:  0x888888FF,
	TextColText:  0xEEEEEEFF,
	TextColHText: 0xEEEEEEFF,
	ModButton:    0x666666FF,
	ColButton:    0x666666FF,
	ButtonColor:  draw.White,
	But2Col:      0xAA0000FF,
	But3Col:      0x006600FF,
	Background:   draw.Black,
	TickColor:    draw.White,
}

// SetDarkMode selects between the light and dark palettes.
func SetDarkMode(enabled bool) {
	darkMode = enabled
	if enabled {
		current = darkPalette
	} else {
		current = lightPalette
	}
}

// IsDarkMode reports the current mode.
func IsDarkMode() bool { return darkMode }

// Current returns the active colour palette.
func Current() Palette { return current }
