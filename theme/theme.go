package theme

import (
	"github.com/rjkroege/edwood/draw"
	"image"
)

var darkMode bool

// SetDarkMode sets the dark mode state and applies the appropriate colors
func SetDarkMode(enabled bool, display draw.Display) {
	darkMode = enabled
	SetColorsForMode(darkMode, display)
}

// IsDarkMode returns the current dark mode state
func IsDarkMode() bool {
	return darkMode
}

var (
	Black         draw.Color
	Darkyellow    draw.Color
	Medblue       draw.Color
	Nofill        draw.Color
	Notacolor     draw.Color
	Opaque        draw.Color
	Palebluegreen draw.Color
	Palegreygreen draw.Color
	Paleyellow    draw.Color
	Purpleblue    draw.Color
	Transparent   draw.Color
	White         draw.Color
	Yellowgreen   draw.Color

	BackgroundColor draw.Color

	TickColor  draw.Image // Now this is not a pointer but an interface
	TagColBack draw.Color
	TagColHigh draw.Color

	TagColBord draw.Color
	TagColText draw.Color
	TagHText   draw.Color

	TextColBack  draw.Color // The text backgrounds
	TextColHigh  draw.Color // The text highlight
	TextColBord  draw.Color // The scroll bar background borders
	TextColText  draw.Color // The text
	TextColHText draw.Color // The color of the text when highlighted

	ModButton draw.Color
	ColButton draw.Color

	ButtonColor draw.Color
	But2Col     draw.Color
	But3Col     draw.Color

	KeyCmd      rune = draw.KeyCmd
	KeyDown     rune = draw.KeyDown
	KeyEnd      rune = draw.KeyEnd
	KeyHome     rune = draw.KeyHome
	KeyInsert   rune = draw.KeyInsert
	KeyLeft     rune = draw.KeyLeft
	KeyPageDown rune = draw.KeyPageDown
	KeyPageUp   rune = draw.KeyPageUp
	KeyRight    rune = draw.KeyRight
	KeyUp       rune = draw.KeyUp
)

// SetColorsForMode sets colors based on the mode (dark or light)
func SetColorsForMode(isDarkMode bool, display draw.Display) {
	if isDarkMode {
		// Define colors for dark mode
		Black = draw.Black
		Darkyellow = 0x6665A8FF
		Medblue = 0xFFFF6DFF
		Nofill = draw.Nofill
		Notacolor = draw.Notacolor
		Palebluegreen = 0x110100FF
		Palegreygreen = draw.Palegreygreen
		Paleyellow = 0x000013FF
		Purpleblue = 0x777738FF
		Transparent = draw.Transparent
		White = draw.White
		Yellowgreen = 0x6665A8FF

		TagColBack = 0x333333FF // The tagcolumn background
		TagColHigh = 0x888888FF // The tagcolumn highlight

		TagColBord = 0x888888FF // The tagcolumn border
		TagColText = 0xEEEEEEFF // The tagcolumn text
		TagHText = 0xEEEEEEFF   // The tagcolumn text when highlighted

		TextColBack = 0x222222FF  // The text backgrounds
		TextColHigh = 0x444444FF  // The text highlight
		TextColBord = 0x888888FF  // The scroll bar background borders
		TextColText = 0xEEEEEEFF  // The text
		TextColHText = 0xEEEEEEFF // The color of the text when highlighted

		BackgroundColor = draw.Black

		ModButton = 0x666666FF // The color of the file-modified button
		ColButton = 0x666666FF // The color of the file-colour button

		ButtonColor = draw.White // The color of the mouse buttons
		But2Col = 0xAA0000FF     // The color of the mouse button 2 functions
		But3Col = 0x006600FF     // The color of the mouse button 3 functions
	} else {
		// Define colors for light mode (default)
		Darkyellow = draw.Darkyellow
		Medblue = draw.Medblue
		Nofill = draw.Nofill
		Notacolor = draw.Notacolor
		Palebluegreen = draw.Palebluegreen
		Palegreygreen = draw.Palegreygreen
		Paleyellow = draw.Paleyellow
		Purpleblue = draw.Purpleblue
		Transparent = draw.Transparent
		White = draw.White
		Yellowgreen = draw.Yellowgreen
		Black = draw.Black

		TagColBack = 0xE0E0E0FF // The tagcolumn background
		TagColHigh = 0xC0C0C0FF // The tagcolumn highlight

		TagColBord = 0x888888FF // The tagcolumn border
		TagColText = draw.Black // The tagcolumn text
		TagHText = draw.Black   // The tagcolumn text when highlighted

		TextColBack = 0xFFFFFFFF  // The text backgrounds
		TextColHigh = 0xFFFF00FF  // The text highlight
		TextColBord = 0x888888FF  // The scroll bar background borders
		TextColText = draw.Black  // The text
		TextColHText = draw.Black // The color of the text when highlighted

		ModButton = 0x222222FF // The color of the file-modified button
		ColButton = 0x666666FF // The color of the file-colour button

		ButtonColor = draw.Black // The color of the mouse buttons
		But2Col = 0xAA0000FF     // The color of the mouse button 2 functions
		But3Col = 0x006600FF     // The color of the mouse button 3 functions
	}

	// Access the ScreenImage method and AllocImage method directly
	screenImage := display.ScreenImage() // This calls the method on the Display interface

	// Allocate an Image for TickColor based on the mode
	var err error
	if isDarkMode {
		TickColor, err = display.AllocImage(image.Rect(0, 0, 1, 1), screenImage.Pix(), true, draw.White)
	} else {
		TickColor, err = display.AllocImage(image.Rect(0, 0, 1, 1), screenImage.Pix(), true, draw.Palegreygreen)
	}

	if err != nil {
		panic("Failed to allocate TickColor image: " + err.Error())
	}
}
