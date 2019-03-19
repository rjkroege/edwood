package complete

import (
	"reflect"
	"runtime"
	"testing"
)

var testCases = []struct {
	dir string
	s   string
	c   Completion
}{
	{"testdata", "aaa", Completion{Advance: true, Complete: true, String: ".txt ", NMatch: 1, Filename: []string{"aaa.txt"}}},
	{"testdata", "aaa.txt", Completion{Advance: true, Complete: true, String: " ", NMatch: 1, Filename: []string{"aaa.txt"}}},
	{"testdata", "bbb", Completion{Advance: true, Complete: true, String: ".dir/", NMatch: 1, Filename: []string{"bbb.dir/"}}},
	{"testdata/ccc", "a", Completion{Advance: false, Complete: false, String: "", NMatch: 0, Filename: []string{"x", "y", "z"}}},
	{"testdata/ccc", "", Completion{Advance: false, Complete: false, String: "", NMatch: 3, Filename: []string{"x", "y", "z"}}},
	{"testdata/ddd", "x", Completion{Advance: true, Complete: false, String: "xx", NMatch: 3, Filename: []string{"xxx1", "xxx2", "xxx3"}}},
	{"testdata/eee", "", Completion{Advance: true, Complete: true, String: "xxx ", NMatch: 1, Filename: []string{"xxx"}}},
}

func TestComplete(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	for _, tc := range testCases {
		c, err := Complete(tc.dir, tc.s)
		if err != nil {
			t.Errorf("Complete of %q failed: %v\n", tc.s, err)
			continue
		}
		if !reflect.DeepEqual(*c, tc.c) {
			t.Errorf("Complete of %q is %#v; expected %#v\n", tc.s, c, tc.c)
		}
	}
}
