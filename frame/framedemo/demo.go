// Demonstrates that the fraame package works.
package main

import (
	"image"
	"log"

	"9fans.net/go/draw"
	"github.com/rjkroege/edwood/frame"
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

	// Try two but one will do. Just cause.
	var myfont *draw.Font
	fontnames  := []string{
		"/mnt/font/Go-Regular/13a/font",
		"/mnt/font/SourceSansPro-Regular/17a/font",
	}

	for _, fn := range fontnames {
		myfont, err = d.OpenFont(fn)
		if err != nil {
			log.Println("Couldn't open font", fn, "because", err)
		}
	}
	if myfont == nil {
		log.Fatalln("None of the font choices were available. Giving up")
	}

	// I need colours to init.
	// TODO(rjk): Test that the window isn't too small.
	mf := new(Myframe)

	mf.f.Init(d.Image.R.Inset(margin), myfont, d.ScreenImage, textcols)

	// get events.
	mousectl := d.InitMouse()
	keyboardctl := d.InitKeyboard()

	mousedown := false

	mf.Resize(false)
	for {
		select {
		case r := <-keyboardctl.C:
			log.Println("----- got rune --------", r)
			switch r {
			case 6: // ^f
				mf.Right()
			case 2: // ^b
				mf.Left()
			case 8: // ^h (delete key)
				mf.Delete()
			case 16: // ^p
				// TODO(rjk): Should go up.
				mf.Logboxes()
			case 7: // ^g
				// Generate some text.
				mf.InsertString(generateParagraphs(1, 8, "\n"), mf.cursor)
			default:
				mf.Insert(r)
			}
			d.Flush()
		case msg := <-mousectl.Resize:
			mf.Resize(msg)
			d.Flush()
		case m := <-mousectl.C:
			// log.Printf("mouse field %v buttons %d\n", m, m.Buttons)

			switch {
			case m.Buttons&1 == 1 && !mousedown:
				mousedown = true
				mf.MouseDown(image.Pt(m.X, m.Y))
				d.Flush()
			case m.Buttons&1 == 1 && mousedown:
				mf.MouseMove(image.Pt(m.X, m.Y))
				d.Flush()
			case m.Buttons&1 == 0 && mousedown:
				mousedown = false
				mf.MouseUp(image.Pt(m.X, m.Y))
				d.Flush()
			}

			//			if m.Buttons&1 == 1 {
			//				// TODO(rjkroege): insert code here to do some drawing and stuff.
			//				d.ScreenImage.Draw(image.Rect(m.X, m.Y, m.X+10, m.Y+10), d.Black, nil, image.ZP)
			//				d.Flush()
			//			}

		}
	}
}
