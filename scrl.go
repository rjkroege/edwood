package main

import (
	"9fans.net/go/draw"
	"fmt"
	"image"
)

var scrtmp *draw.Image

func ScrSleep(dt uint) {
	Unimpl()
}

func ScrlResize(dt uint) {
	var err error
	scrtmp, err = display.AllocImage(image.Rect(0, 0, 32, display.ScreenImage.R.Max.Y), display.ScreenImage.Pix, false, draw.Nofill)
	if err != nil {
		panic(fmt.Sprintf("scroll alloc: %v", err))
	}
}

func (t *Text) ScrDraw() {
	Unimpl()
}

func (t *Text) Scroll(but uint) {
	Unimpl()
}
