package main

import (
	"fmt"

	"testing"
)

func TestAppend(t *testing.T) {
	testtab := []struct {
		dot Range
		filename string
		expr string
		expected string
	}{
		{Range{0, 0}, "test", "a/junk", "junkThis is a\nshort text\nto try addressing\n"},
	}
	
	buf := make([]rune, 8192)

	for i, test := range testtab {
		w := NewWindow().initHeadless(nil)
		w.body.Insert(0, []rune("This is a\nshort text\nto try addressing\n"), true)
		fmt.Printf("w.body has window %+v\n", w.body.w)
		editcmd(&w.body, []rune(test.expr))
		// Normally the edit log is applied in allupdate, but we don't have
		// all the window machinery, so we apply it by hand.
		w.body.file.elog.Apply(&w.body)
		n, _ := w.body.ReadB(0, buf[:])
		if string(buf[:n]) != test.expected {
			t.Errorf("test %d: TestAppend expected \n%v\nbut got \n%v\n", i, test.expected, string(buf[:n]))
		}
	}
}