// Demonstrates that the fraame package works.
package main

import (
	"image"
	"log"

	"9fans.net/go/draw"
	"github.com/paul-lalonde/acme/frame"
)

const margin = 20

func main() {
	// Make the window.
	d, err := draw.Init(nil, "", "framedemo", "")
	if err != nil {
		log.Fatal(err)
	}

	// TODO(rjk): capture errors correctly.
	// TODO(rjk): Make the list of colours be a slice.
	var textcols [frame.NumColours]*draw.Image
	textcols[frame.ColBack] = d.AllocImageMix(draw.Paleyellow, draw.White)
	textcols[frame.ColHigh], _ = d.AllocImage(image.Rect(0, 0, 1, 1), d.ScreenImage.Pix, true, draw.Darkyellow)
	textcols[frame.ColBord], _ = d.AllocImage(image.Rect(0, 0, 1, 1), d.ScreenImage.Pix, true, draw.Yellowgreen)
	textcols[frame.ColText] = d.Black
	textcols[frame.ColHText] = d.Black

	// TODO(rjk): Use a font that always is available.
	fontname := "/mnt/font/Go-Regular/13a/font"
	myfont, err := d.OpenFont(fontname)
	if err != nil {
		log.Fatalln("Couldn't open font", fontname, "because", err)
	}

	// I need colours to init. I
	// TODO(rjk): Test that the window isn't too small.
	mf := new(Myframe)

	mf.f.Init(d.Image.R.Inset(margin), myfont, d.ScreenImage, textcols)

	// get events.
	mousectl := d.InitMouse()
	keyboardctl := d.InitKeyboard()

	mf.Resize(false)
	for {
		select {
		case r := <-keyboardctl.C:
			log.Println("----- got rune --------", r)
			switch r {
			case 6:
				mf.Right()
			case 2:
				mf.Left()
			case 8:
				mf.Delete()
			case 16:
				mf.Logboxes()
			default:
				mf.Insert(r)
			}
			d.Flush()
		case <-mousectl.Resize:
			mf.Resize(true)
			d.Flush()
		case m := <-mousectl.C:
			// fmt.Printf("mouse field %v buttons %d\n", m, m.Buttons)

			if m.Buttons&1 == 1 {
				// TODO(rjkroege): insert code here to do some drawing and stuff.
				d.ScreenImage.Draw(image.Rect(m.X, m.Y, m.X+10, m.Y+10), d.Black, nil, image.ZP)
				d.Flush()
			}
		}
	}
}
