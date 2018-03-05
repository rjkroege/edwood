package main

import (
	"9fans.net/go/draw"
	"github.com/ProjectSerenity/acme/frame"
	"image"
)

const (
	NSnarf = 1000
)

var (
	snarfrune [NSnarf + 1]rune

	fontnames = [2]string{
		"/lib/font/bit/lucsans/euro.8.font",
		"/lib/font/bit/lucm/unicode.9.font",
	}

//	command *Command
)

func mousethread() {

}

func keyboardthread() {

}

func waitthread() {

}

func xfidallocthread() {

}

func newwindowthread() {

}

func plumbproc() {

}

func timefmt( /*Fmt* */ ) int {
	return 0
}

func main() {
	var cols [5]*draw.Image
	errch := make(chan<- error)
	display, err := draw.Init(errch, "", "acme", "1024x720")
	if err != nil {
		panic(err)
	}
	img, err := display.AllocImage(image.Rect(0, 0, 1024, 720), draw.RGB16, true, draw.Cyan)
	if err != nil {
		panic(err)
	}
	f := frame.NewFrame(image.Rect(0, 0, 500, 600), display.DefaultFont, img, cols)

	for {
		f.Tick(image.ZP, true)
	}
}
