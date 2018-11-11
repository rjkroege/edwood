package main

import (
	"reflect"
	"testing"
)

func TestDirname(t *testing.T) {
	testCases := []struct {
		b, r, dir []rune
	}{
		{[]rune("/a/b/c/d.go Del Snarf | Look "), nil, []rune("/a/b/c")},
		{[]rune("/a/b/c/d.go Del Snarf | Look "), []rune("e.go"), []rune("/a/b/c/e.go")},
		{[]rune("/a/b/c/d.go Del Snarf | Look "), []rune("/x/e.go"), []rune("/x/e.go")},
	}

	for _, tc := range testCases {
		text := Text{
			w: &Window{
				tag: Text{
					file: &File{
						b: Buffer(tc.b),
					},
				},
			},
		}
		dir := dirname(&text, tc.r)
		if !reflect.DeepEqual(dir, tc.dir) {
			t.Errorf("dirname of %q (r=%q) is %q; expected %q\n", tc.b, tc.r, dir, tc.dir)
		}
	}
}
