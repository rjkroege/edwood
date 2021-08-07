// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package utf8bytes

import (
	"fmt"
	"math/rand"
	"testing"
	"unicode/utf8"
)

var testStrings = []string{
	"",
	"abcd",
	"☺☻☹",
	"日a本b語ç日ð本Ê語þ日¥本¼語i日©",
	"日a本b語ç日ð本Ê語þ日¥本¼語i日©日a本b語ç日ð本Ê語þ日¥本¼語i日©日a本b語ç日ð本Ê語þ日¥本¼語i日©",
	"\x80\x80\x80\x80",
}

func TestScanForwards(t *testing.T) {
	for _, s := range testStrings {
		runes := []rune(s)
		bytes := NewBytes([]byte(s))
		if bytes.RuneCount() != len(runes) {
			t.Errorf("%s: expected %d runes; got %d", s, len(runes), bytes.RuneCount())
			break
		}
		for i, expect := range runes {
			got := bytes.At(i)
			if got != expect {
				t.Errorf("%s[%d]: expected %c (%U); got %c (%U)", s, i, expect, expect, got, got)
			}
		}
	}
}

func TestScanBackwards(t *testing.T) {
	for _, s := range testStrings {
		runes := []rune(s)
		bytes := NewBytes([]byte(s))
		if bytes.RuneCount() != len(runes) {
			t.Errorf("%s: expected %d runes; got %d", s, len(runes), bytes.RuneCount())
			break
		}
		for i := len(runes) - 1; i >= 0; i-- {
			expect := runes[i]
			got := bytes.At(i)
			if got != expect {
				t.Errorf("%s[%d]: expected %c (%U); got %c (%U)", s, i, expect, expect, got, got)
			}
		}
	}
}

func randCount() int {
	if testing.Short() {
		return 100
	}
	return 100000
}

func TestRandomAccess(t *testing.T) {
	for _, s := range testStrings {
		if len(s) == 0 {
			continue
		}
		runes := []rune(s)
		bytes := NewBytes([]byte(s))
		if bytes.RuneCount() != len(runes) {
			t.Errorf("%s: expected %d runes; got %d", s, len(runes), bytes.RuneCount())
			break
		}
		for j := 0; j < randCount(); j++ {
			i := rand.Intn(len(runes))
			expect := runes[i]
			got := bytes.At(i)
			if got != expect {
				t.Errorf("%s[%d]: expected %c (%U); got %c (%U)", s, i, expect, expect, got, got)
			}
		}
	}
}

func TestRandomSliceAccess(t *testing.T) {
	for _, s := range testStrings {
		if len(s) == 0 || s[0] == '\x80' { // the bad-UTF-8 string fools this simple test
			continue
		}
		runes := []rune(s)
		bytes := NewBytes([]byte(s))
		if bytes.RuneCount() != len(runes) {
			t.Errorf("%s: expected %d runes; got %d", s, len(runes), bytes.RuneCount())
			break
		}
		for k := 0; k < randCount(); k++ {
			i := rand.Intn(len(runes))
			j := rand.Intn(len(runes) + 1)
			if i > j { // include empty strings
				continue
			}
			expect := string(runes[i:j])
			got := string(bytes.Slice(i, j))
			if got != expect {
				t.Errorf("%s[%d:%d]: expected %q got %q", s, i, j, expect, got)
			}
		}
	}
}

func TestLimitSliceAccess(t *testing.T) {
	for _, s := range testStrings {
		bytes := NewBytes([]byte(s))

		if string(bytes.Slice(0, 0)) != "" {
			t.Error("failure with empty slice at beginning")
			t.Error("Failed with string: ", s)

			stuffsBegin := bytes.Slice(0, 0)
			if stuffsBegin != nil {
				println("salkdjaslkdj")
			}
			fmt.Printf("StuffsBegin: %v\n", stuffsBegin)
		}
		nr := utf8.RuneCountInString(s)

		if string(bytes.Slice(nr, nr)) != "" {
			t.Error("failure with empty slice at end")

			stuffsEnd := bytes.Slice(nr, nr)
			fmt.Printf("StuffsEnd: %v\n", stuffsEnd)
		}
	}
}
