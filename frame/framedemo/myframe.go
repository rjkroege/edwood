// Demonstrates that the fraame package works.
package main

import (
	"image"
	"log"

	"9fans.net/go/draw"
	"github.com/ProjectSerenity/acme/frame"
)

// A naïve buffer implementation for testing.
type Myframe struct {
	f frame.Frame

	buffer     []rune
	cursor     int // dynamic end of selection.
	cursordown int // mousedown point in selection
	offset     int // the offset of the frame w.r.t. buffer.
}

// TODO(rjk): Add a function to create Myframe instances.
// TODO(rjk): Update for resize.
func (mf *Myframe) Resize(resized bool) {
	log.Println("Myframe.Resize")
	if resized {
		// Stupid implementation.

		mf.f.Clear(false)

		if err := mf.f.Display.Attach(draw.Refmesg); err != nil {
			log.Fatalf("can't reattach to window: %v", err)
		}
		
		mf.f.SetRects( mf.f.Display.Image.R.Inset(20),  mf.f.Display.ScreenImage)
	}

	mf.f.Background.Draw(
		mf.f.Rect,
		mf.f.Cols[frame.ColBack],
		nil,
		image.ZP)


	if resized {
		// insert text such that we fit a window around the cursor.
		// TODO(rjk): Adjust for scrollable buffers.
		mf.f.Insert(mf.buffer,  0)
	}

	// Set the tick
	mf.f.DrawSel(mf.f.Ptofchar(mf.cursordown), mf.cursordown, mf.cursor, true)
	mf.f.Display.Flush()
}

// InsertString is a helper method to pre-populate the model with text.
func (mf *Myframe) InsertString(s string, c int) {
	oc := mf.cursor
	mf.cursor = c
	for _, r := range s {
		mf.Insert(r)
	}

	mf.cursor = oc
	mf.cursordown = mf.cursor
	mf.f.DrawSel(mf.f.Ptofchar(mf.cursor), mf.cursordown, mf.cursor, true)
}

// Insert adds a single rune to the frame at the cursor.
func (mf *Myframe) Insert(r rune) {
	if mf.cursor-mf.cursordown > 0 {
		mf.f.Delete(mf.cursordown, mf.cursor)
		mf.f.Insert([]rune{r}, mf.cursordown)

		if mf.cursor < len(mf.buffer) {
			copy(mf.buffer[mf.cursordown+1:], mf.buffer[mf.cursor:])
		}
		mf.buffer = mf.buffer[0 : len(mf.buffer)-(mf.cursor-mf.cursordown)+1]
		mf.cursor = mf.cursordown
		mf.buffer[mf.cursor] = r
	} else {
		mf.f.Insert([]rune{r}, mf.cursor)

		mf.buffer = append(mf.buffer, ' ')
		copy(mf.buffer[mf.cursor+1:], mf.buffer[mf.cursor:])
		mf.buffer[mf.cursor] = r
		mf.cursor++
		mf.cursordown = mf.cursor
	}
	mf.f.DrawSel(mf.f.Ptofchar(mf.cursordown), mf.cursordown, mf.cursor, true)
}

// Delete removes a single rune at the cursor.
func (mf *Myframe) Delete() {
	if mf.cursor < 1 {
		return
	}
	if mf.cursor-mf.cursordown > 0 {
		mf.f.Delete(mf.cursordown, mf.cursor)

		if mf.cursor < len(mf.buffer) {
			copy(mf.buffer[mf.cursordown:], mf.buffer[mf.cursor:])
		}
		mf.buffer = mf.buffer[0 : len(mf.buffer)-(mf.cursor-mf.cursordown)]
		mf.cursor = mf.cursordown
	} else {
		mf.f.Delete(mf.cursor-1, mf.cursor)
		if mf.cursor < len(mf.buffer) {
			copy(mf.buffer[mf.cursor-1:], mf.buffer[mf.cursor:])
		}
		mf.buffer = mf.buffer[0 : len(mf.buffer)-1]
		mf.cursor--
		mf.cursordown = mf.cursor
	}

	mf.f.DrawSel(mf.f.Ptofchar(mf.cursordown), mf.cursordown, mf.cursor, true)
}

// Up moves the cursor up a line if possible and adjusts the frame.
func (my *Myframe) Up() {
	log.Println("Up no know how to dealz")
}

// Left moves the cursor to the left if possible.
func (mf *Myframe) Left() {
	if mf.cursor > 0 {
		mf.cursor--
		mf.cursordown = mf.cursor
		mf.f.DrawSel(mf.f.Ptofchar(mf.cursor), mf.cursordown, mf.cursor, true)
	}
}

// Right moves the cursor to the right if possible.
func (mf *Myframe) Right() {
	if mf.cursor < len(mf.buffer) {
		mf.cursor++
		mf.cursordown = mf.cursor
		mf.f.DrawSel(mf.f.Ptofchar(mf.cursor), mf.cursordown, mf.cursor, true)
	}
}

// Down moves the cursor down if possible.
func (my *Myframe) Down() {
	log.Println("Down no know how to dealz")
}

func (my *Myframe) Logboxes() {
	my.f.Logboxes("-- current boxes --")
}

func (mf *Myframe) MouseDown(pt image.Point) {
	nc := mf.f.Charofpt(pt)
	mf.cursordown = nc
	mf.cursor = nc

	selpt := mf.f.Ptofchar(mf.cursordown)
	mf.f.DrawSel(selpt, mf.cursordown, mf.cursor, true)
}

func (mf *Myframe) MouseMove(pt image.Point) {
	nc := mf.f.Charofpt(pt)
	mf.cursor = nc

	log.Println("MouseMove basic", pt, "->", nc)

	if mf.cursordown <= mf.cursor {
		selpt := mf.f.Ptofchar(mf.cursordown)
		log.Println("MouseMove str", selpt,string(mf.buffer[mf.cursordown:mf.cursor]))
		mf.f.DrawSel(selpt, mf.cursordown, mf.cursor, true)
	} else {
		selpt := mf.f.Ptofchar(mf.cursor)
		log.Println("MouseMove str", selpt,string(mf.buffer[mf.cursor:mf.cursordown]))
		mf.f.DrawSel(selpt, mf.cursor, mf.cursordown, true)
	}
}

func (mf *Myframe) MouseUp(pt image.Point) {
	log.Println("\nMouseUp")
	nc := mf.f.Charofpt(pt)

	mf.cursor = nc

	if mf.cursordown <= mf.cursor {
		selpt := mf.f.Ptofchar(mf.cursordown)
		mf.f.DrawSel(selpt, mf.cursordown, mf.cursor, true)
	} else {
		selpt := mf.f.Ptofchar(mf.cursor)
		mf.f.DrawSel(selpt, mf.cursor, mf.cursordown, true)
	}

	// At this point, order the cursordown, cursor so that writing the insert / deletion
	// code is easier.
	if mf.cursordown > mf.cursor {
		tc := mf.cursordown
		mf.cursordown = mf.cursor
		mf.cursor = tc
	}
}
