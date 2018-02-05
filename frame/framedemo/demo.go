// Demonstrates that the fraame package works.
package main

import (
	"fmt"
	"image"
	"log"

	"9fans.net/go/draw"
)


// redraw is the view implementation
func redraw(d *draw.Display, resized bool, myfont *draw.Font) {
	if resized {
		if err := d.Attach(draw.Refmesg); err != nil {
			log.Fatalf("can't reattach to window: %v", err)
		}
	}

	// draw coloured rects at mouse positions
	// first param is the clip rectangle. which can be 0. meaning no clip?
	var clipr image.Rectangle
	fmt.Printf("empty clip? %v\n", clipr)
	d.ScreenImage.Draw(clipr, d.White, nil, image.ZP)


	// draw some text
	d.ScreenImage.String(image.Pt(100,100), d.Black, image.ZP, myfont, "hello world")
	d.Flush()
}

func main() {
	fmt.Print("hello from framedemo\n")

	// Make the window.
	d, err := draw.Init(nil, "", "framedemo", "")
	if err != nil {
		log.Fatal(err)
	}

	// make some colors
	back, _ := d.AllocImage(image.Rect(0, 0, 1, 1), d.ScreenImage.Pix, true, 0xDADBDAff)

	// make a font
	// TODO(rjk): Use a font that always is available.
	fontname := "/mnt/font/SourceSansPro-Regular/13a/font"
	myfont, err := d.OpenFont(fontname)
	if err != nil {
		log.Fatalln("Couldn't open font", fontname, "because", err)
	}

	fmt.Printf("background colour: %v\n ", back)

	// get events.
	mousectl := d.InitMouse()
	keyboardctl := d.InitKeyboard()

	redraw(d, false, myfont)
	for {
		select {
		case r := <- keyboardctl.C:
			log.Println("got rune", r)
		case <-mousectl.Resize:
			redraw(d, true, myfont)
		case m := <-mousectl.C:
			// fmt.Printf("mouse field %v buttons %d\n", m, m.Buttons)
	
			if (m.Buttons & 1 == 1) {
				// TODO(rjkroege): insert code here to do some drawing and stuff.
				d.ScreenImage.Draw(image.Rect(m.X, m.Y, m.X+10, m.Y+10), back, nil, image.ZP)
				d.Flush()
			}
		}
	}

	fmt.Print("bye\n")
}
