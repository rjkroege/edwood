// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package file

import (
	"bytes"
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
		b := NewBytes([]byte(s))
		if b.RuneCount() != len(runes) {
			t.Errorf("%s: expected %d runes; got %d", s, len(runes), b.RuneCount())
			break
		}
		for i, expect := range runes {
			got := b.At(i)
			if got != expect {
				t.Errorf("%s[%d]: expected %c (%U); got %c (%U)", s, i, expect, expect, got, got)
			}
		}
	}
}

func TestScanBackwards(t *testing.T) {
	for _, s := range testStrings {
		runes := []rune(s)
		b := NewBytes([]byte(s))
		if b.RuneCount() != len(runes) {
			t.Errorf("%s: expected %d runes; got %d", s, len(runes), b.RuneCount())
			break
		}
		for i := len(runes) - 1; i >= 0; i-- {
			expect := runes[i]
			got := b.At(i)
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
		b := NewBytes([]byte(s))
		if b.RuneCount() != len(runes) {
			t.Errorf("%s: expected %d runes; got %d", s, len(runes), b.RuneCount())
			break
		}
		for j := 0; j < randCount(); j++ {
			i := rand.Intn(len(runes))
			expect := runes[i]
			got := b.At(i)
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
		b := NewBytes([]byte(s))
		if b.RuneCount() != len(runes) {
			t.Errorf("%s: expected %d runes; got %d", s, len(runes), b.RuneCount())
			break
		}
		for k := 0; k < randCount(); k++ {
			i := rand.Intn(len(runes))
			j := rand.Intn(len(runes) + 1)
			if i > j { // include empty strings
				continue
			}
			expect := string(runes[i:j])
			got := string(b.Slice(i, j))
			if got != expect {
				t.Errorf("%s[%d:%d]: expected %q got %q", s, i, j, expect, got)
			}
		}
	}
}

func TestLimitSliceAccess(t *testing.T) {
	for _, s := range testStrings {
		b := NewBytes([]byte(s))

		if string(b.Slice(0, 0)) != "" {
			t.Error("failure with empty slice at beginning")
			t.Error("Failed with string: ", s)
		}
		nr := utf8.RuneCountInString(s)

		if string(b.Slice(nr, nr)) != "" {
			t.Error("failure with empty slice at end")
		}
	}
}

func TestBytes_Read(t *testing.T) {
	for _, s := range testStrings {
		b := NewBytes([]byte(s))
		strLen := len(s)
		var readTo int
		if strLen == 0 {
			readTo = 0
		} else {
			readTo = rand.Intn(strLen)
		}

		got := make([]byte, readTo)
		b.Read(got)

		reader := bytes.NewReader(b.Byte())
		wanted := make([]byte, readTo)
		reader.Read(wanted)

		if string(got) != string(wanted) {
			t.Errorf("Expected: %s, got: %s\n", got, wanted)
		}
	}
}
