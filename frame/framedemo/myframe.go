// Demonstrates that the fraame package works.
package main

import (
	"log"
	"image"


//	"9fans.net/go/draw"
	"github.com/ProjectSerenity/acme/frame"

	
)

// Stupid buffer world. Not intended to be particularly smart.
type Myframe struct {
	f frame.Frame

	buffer []rune
	cursor int // a position at which we can insert text into the backing buffer.
	offset int // the offset of the frame w.r.t. buffer. 
}

const motext = ` 2018/02/11 16:35:03 first box frame.frbox{Wid:112, Nrune:11, Ptr:[]uint8{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x74, 0x68, 0x65, 0x72, 0x65, 0x0}, Bc:0, Minwid:0x0}, hello there`


// must insert the size
func (mf *Myframe) Resize(resized bool) {
	log.Println("Myframe.Resize")
	if (resized) {
		log.Println("i no know how to dealz")
		// TODO(rjk): stuff.
	}

	mf.f.Background.Draw(
		mf.f.Rect,
		mf.f.Cols[frame.ColBack],
		nil, 
		image.ZP)

	// I could imagine doing this again? More draw ops?
	mf.f.Insert([]rune("hello there"), 0)
	mf.f.Display.Flush()

	mf.f.Insert([]rune("motext "), 1)
	mf.f.Display.Flush()

	mf.f.Insert([]rune("≤日本b≥"), 3)

	// TODO(rjk): Redraw does the wrong thing. Fix that if necessary.
	// Redraw is not part of frame(3) interface (e.g. no frredraw)
	// mf.f.Redraw()
	mf.f.Display.Flush()

	mf.f.Insert([]rune("Bytes draws the byte slice in the specified\nfont using SoverD on the image,"), 8)
	mf.f.Display.Flush()

}

// Insert adds a single rune to the frame at the cursor.
func (my *Myframe) Insert(r rune) {
		log.Println("Insert i no know how to dealz")
}

// Delete removes a single rune at the cursor.
func (my *Myframe) Delete() {
		log.Println("Deletei no know how to dealz")
}

// Up moves the cursor up a line if possible and adjusts the frame.
func (my *Myframe) Up() {
		log.Println("Upi no know how to dealz")
}

// Left moves the cursor to the left if possible.
func (my *Myframe) Left() {
		log.Println("Lefti no know how to dealz")
}

// Right moves the cursor to the right if possible.
func (my *Myframe) Right() {
		log.Println("Righti no know how to dealz")
}

// Down moves the cursor down if possible.
func (my *Myframe) Down() {
		log.Println("Downi no know how to dealz")
}

