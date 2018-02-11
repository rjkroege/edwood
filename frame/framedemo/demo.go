// Demonstrates that the fraame package works.
package main

import (
	"fmt"
	"image"
	"log"

	"9fans.net/go/draw"
	"github.com/ProjectSerenity/acme/frame"
)

/*

// redraw is the view implementation
func showwindow(d *draw.Display, resized bool, f *MyFrame) {
	if resized {
		if err := d.Attach(draw.Refmesg); err != nil {
			log.Fatalf("can't reattach to window: %v", err)
		}


		f.resize()


	}


	// I don't think this is necessary...

	// draw coloured rects at mouse positions
	// first param is the clip rectangle. which can be 0. meaning no clip?
	var clipr image.Rectangle
	fmt.Printf("empty clip? %v\n", clipr)
	d.ScreenImage.Draw(clipr, d.White, nil, image.ZP)

	// how do I know how big the display is?
	//



	// draw some text
	d.ScreenImage.String(image.Pt(100,100), d.Black, image.ZP, myfont, "hello world")
	d.Flush()
}
*/

const margin = 20

func main() {
	log.Println("hello from framedemo\n")

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
	fontname := "/mnt/font/SourceSansPro-Regular/13a/font"
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

// So, why don't I haz anything on screen?

	mf.Resize(false)
	for {
		select {
		case r := <-keyboardctl.C:
			log.Println("got rune", r)
		case <-mousectl.Resize:
			mf.Resize(true)
		case m := <-mousectl.C:
			// fmt.Printf("mouse field %v buttons %d\n", m, m.Buttons)

			if m.Buttons&1 == 1 {
				// TODO(rjkroege): insert code here to do some drawing and stuff.
				d.ScreenImage.Draw(image.Rect(m.X, m.Y, m.X+10, m.Y+10), d.Black, nil, image.ZP)
				d.Flush()
			}
		}
	}

	fmt.Print("bye\n")
}
