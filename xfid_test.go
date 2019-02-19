package main

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"9fans.net/go/plan9"
	"github.com/rjkroege/edwood/internal/draw"
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

func TestFullrunewrite(t *testing.T) {
	testCases := []struct {
		in        []byte
		out       []rune
		pin, pout []byte
	}{
		{[]byte("hello world"), []rune("hello world"), nil, nil},
		{[]byte("Hello, 世界"), []rune("Hello, 世界"), nil, nil},
		{[]byte("hello \x00\x00world"), []rune("hello world"), nil, nil},
		{[]byte("abc\xe4\xb8xyz"), []rune("abc\uFFFD\uFFFDxyz"), nil, nil},        // invalid rune
		{[]byte("abcxyz\xe4\xb8"), []rune("abcxyz"), nil, []byte{'\xe4', '\xb8'}}, // ends with partial rune
		{[]byte("\x96hello"), []rune("世hello"), []byte{'\xe4', '\xb8'}, nil},      // begins with partial rune
	}
	for _, tc := range testCases {
		x := Xfid{
			f: &Fid{
				nrpart: len(tc.pin),
			},
			fcall: plan9.Fcall{
				Data:  tc.in,
				Count: uint32(len(tc.in)),
			},
		}
		copy(x.f.rpart[:], tc.pin)

		r := fullrunewrite(&x)
		if !reflect.DeepEqual(r, tc.out) {
			for i, b := range tc.in {
				fmt.Printf("%v %x %d\n", i, b, b)
			}
			t.Errorf("Fullrunewrite(%q, %q) full runes are %q; expected %q\n", tc.pin, tc.in, r, tc.out)
		}
		if x.f.nrpart != len(tc.pout) || !bytes.Equal(x.f.rpart[:x.f.nrpart], tc.pout[:]) {
			t.Errorf("Fullrunewrite(%q, %q) partial runes are %q; expected %q\n", tc.pin, tc.in, x.f.rpart[:x.f.nrpart], tc.pout)
		}
	}
}
