// Demonstrates that the fraame package works.
package main

import (
	"log"
	"image"


//	"9fans.net/go/draw"
	"github.com/paul-lalonde/acme/frame"

	
)

// Stupid buffer world. Not intended to be particularly smart.
type Myframe struct {
	f frame.Frame

	buffer []rune
	cursor int // a position at which we can insert text into the backing buffer.
	offset int // the offset of the frame w.r.t. buffer. 
}

const motext = ` 2018/02/11 16:35:03 first box frame.frbox{Wid:112, Nrune:11, Ptr:[]uint8{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x74, 0x68, 0x65, 0x72, 0x65, 0x0}, Bc:0, Minwid:0x0}, hello there`

// TODO(rjk): Add a function to create Myframe instances.


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
//	mf.f.Insert([]rune("hello there"), 0)
//	mf.InsertString("hello there", 0)
//	mf.f.Display.Flush()

//	mf.f.Insert([]rune("motext "), 1)
//	mf.InsertString("motext ", 1)
//	mf.f.Display.Flush()

//	mf.f.Insert([]rune("≤日本b≥"), 3)
//	mf.InsertString("≤日本b≥", 3)

	// TODO(rjk): Redraw does the wrong thing. Fix that if necessary.
	// Redraw is not part of frame(3) interface (e.g. no frredraw)
	// mf.f.Redraw()
//	mf.f.Display.Flush()

//	mf.f.Insert([]rune("Bytes draws the byte slice in the specified\nfont using SoverD on the image,"), 8)
//	mf.InsertString("Bytes draws the byte slice in the specified\nfont using SoverD on the image,", 8)
	
	mf.InsertString("ab", 0)


	// Set the tick
	mf.f.Tick(mf.f.Ptofchar(0), true)

	mf.f.Display.Flush()

	log.Printf("starting buffer %#v\n", string(mf.buffer))

}

// InsertString is a helper method to pre-populate the model with text.
func (mf *Myframe) InsertString(s string, c int) {
	oc := mf.cursor
	mf.cursor = c
	for _, r := range s {
		mf.Insert(r)
	}
	mf.cursor = oc
}

// Insert adds a single rune to the frame at the cursor.
func (mf *Myframe) Insert(r rune) {
	mf.f.Tick(mf.f.Ptofchar(mf.cursor), false)

	mf.f.Insert([]rune{r}, mf.cursor)

	mf.buffer = append(mf.buffer, ' ')
	copy(mf.buffer[mf.cursor+1:], mf.buffer[mf.cursor:])
	mf.buffer[mf.cursor] = r
	mf.cursor++

	mf.f.Tick(mf.f.Ptofchar(mf.cursor), true)
}

// Delete removes a single rune at the cursor.
func (mf *Myframe) Delete() {
	if mf.cursor < 1 {
		return
	}
	mf.f.Tick(mf.f.Ptofchar(mf.cursor), false)
	
	mf.f.Delete(mf.cursor - 1, mf.cursor)

	if mf.cursor < len(mf.buffer) {
		copy(mf.buffer[mf.cursor-1:], mf.buffer[mf.cursor:])
	}
	mf.buffer = mf.buffer[0:len(mf.buffer)-1]
	mf.cursor--

	mf.f.Tick(mf.f.Ptofchar(mf.cursor), true)
}

// Up moves the cursor up a line if possible and adjusts the frame.
func (my *Myframe) Up() {
	log.Println("Up no know how to dealz")
}

// Left moves the cursor to the left if possible.
func (mf *Myframe) Left() {
	if mf.cursor > 0 {
		mf.f.Tick(mf.f.Ptofchar(mf.cursor), false)
		mf.cursor--
		mf.f.Tick(mf.f.Ptofchar(mf.cursor), true)
	}
}

// Right moves the cursor to the right if possible.
func (mf *Myframe) Right() {
	if mf.cursor <=  len(mf.buffer) {
		mf.f.Tick(mf.f.Ptofchar(mf.cursor), false)
		mf.cursor++
		mf.f.Tick(mf.f.Ptofchar(mf.cursor), true)
	}
}

// Down moves the cursor down if possible.
func (my *Myframe) Down() {
	log.Println("Down no know how to dealz")
}

func (my *Myframe) Logboxes() {
	my.f.Logboxes("-- current boxes --")
}
