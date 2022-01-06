//go:build !duitdraw && !windows
// +build !duitdraw,!windows

package draw

import (
	draw "9fans.net/go/draw"
)

const (
	Refnone = draw.Refnone

	KeyCmd      = draw.KeyCmd
	KeyDown     = draw.KeyDown
	KeyEnd      = draw.KeyEnd
	KeyHome     = draw.KeyHome
	KeyInsert   = draw.KeyInsert
	KeyLeft     = draw.KeyLeft
	KeyPageDown = draw.KeyPageDown
	KeyPageUp   = draw.KeyPageUp
	KeyRight    = draw.KeyRight
	KeyUp       = draw.KeyUp

	Darkyellow    = draw.Darkyellow
	Medblue       = draw.Medblue
	Nofill        = draw.Nofill
	Notacolor     = draw.Notacolor
	Palebluegreen = draw.Palebluegreen
	Palegreygreen = draw.Palegreygreen
	Paleyellow    = draw.Paleyellow
	Purpleblue    = draw.Purpleblue
	Transparent   = draw.Transparent
	White         = draw.White
	Yellowgreen   = draw.Yellowgreen
)

type (
	Color       = draw.Color
	Cursor      = draw.Cursor
	drawDisplay = draw.Display
	drawFont    = draw.Font
	drawImage   = draw.Image
	Keyboardctl = draw.Keyboardctl
	Mousectl    = draw.Mousectl
	Mouse       = draw.Mouse
	Pix         = draw.Pix
)

var Init = draw.Init

func Main(f func(*Device)) {
	f(new(Device))
}

type Device struct{}

func (dev *Device) NewDisplay(errch chan<- error, fontname, label, winsize string) (Display, error) {
	d, err := Init(errch, fontname, label, winsize)
	if err != nil {
		return nil, err
	}
	return &displayImpl{d}, nil
}
