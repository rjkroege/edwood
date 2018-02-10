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

	// Trial...
	mf.f.Insert([]rune("hello there"), 0)

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

