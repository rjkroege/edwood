package main

import (
	"testing"

	"9fans.net/go/draw"
)

func TestXfidAlloc(t *testing.T) {

	cxfidalloc = make(chan *Xfid)
	cxfidfree = make(chan *Xfid)

	d := (*draw.Display)(nil)
	go xfidallocthread(d)

	cxfidalloc <- (*Xfid)(nil) // Request an xfid
	x := <-cxfidalloc
	if x == nil {
		t.Errorf("Failed to get an Xfid")
	}
	cxfidfree <- x
}
