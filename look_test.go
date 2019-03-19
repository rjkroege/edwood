package main

import (
	"fmt"
	"reflect"
	"runtime"
	"testing"
)

func TestDirname(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
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

func TestExpand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	tt := []struct {
		ok   bool
		sel1 int
		s    string
		inq  int
		q    string
		name string
		addr string
	}{
		{false, 0, "     ", 2, "", "", ""},
		{false, 0, "@@@@", 2, "", "", ""},
		{true, 0, "hello", 2, "hello", "", ""},
		{true, 5, "chicken", 2, "chick", "", ""},
		{true, 0, "hello.go", 2, "hello", "", ""},
		{true, 0, "hello.go:42", 2, "hello", "", ""},
		{true, 0, "世界.go:42", 2, "世界", "", ""},
		{true, 0, ":123", 2, ":123", "", "123"},
		{true, 0, ":/hello/", 2, ":/", "", "/hello/"},
		{true, 0, ":/世界/", 2, ":/", "", "/世界/"},
		{true, 0, "look_test.go", 2, "look_test.go", "look_test.go", ""},
		{true, 0, "look_test.go:42", 2, "look_test.go:42", "look_test.go", "42"},
		{true, 0, "look_test.go:42 ", 2, "look_test.go:42", "look_test.go", "42"},
		{true, 0, "look_test.go:42", 14, "look_test.go:42", "look_test.go", "42"},
		{true, 0, "<stdio.h>", 2, "stdio", "", ""},
		{true, 0, "/etc/hosts", 2, "/etc/hosts", "/etc/hosts", ""},
		{true, 0, "/etc/hosts:42", 2, "/etc/hosts:42", "/etc/hosts", "42"},
	}
	for i, tc := range tt {
		t.Run(fmt.Sprintf("test-%02d", i), func(t *testing.T) {
			r := []rune(tc.s)
			text := &Text{
				file: &File{
					b: r,
				},
				q0: 0,
				q1: tc.sel1,
			}
			e, ok := expand(text, tc.inq, tc.inq)
			if ok != tc.ok {
				t.Fatalf("expand of %q returned %v; expected %v", tc.s, ok, tc.ok)
			}
			//t.Logf("expansion: %#v", e)
			q := string(r[e.q0:e.q1])
			if q != tc.q {
				t.Errorf("q0:q1 of %q is %q; expected %q", tc.s, q, tc.q)
			}
			if e.name != tc.name {
				t.Errorf("name of %q is %q; expected %q", tc.s, e.name, tc.name)
			}
			addr := ""
			if e.a0 < len(r) {
				addr = string(r[e.a0:e.a1])
			}
			if addr != tc.addr {
				t.Errorf("address of %q is %q; expected %q", tc.s, addr, tc.addr)
			}
		})
	}
}

func TestExpandJump(t *testing.T) {
	tt := []struct {
		kind TextKind
		jump bool
	}{
		{Tag, false},
		{Body, true},
	}

	for _, tc := range tt {
		text := &Text{
			file: &File{
				b: []rune("chicken"),
			},
			q0:   0,
			q1:   5,
			what: tc.kind,
		}
		e, _ := expand(text, 2, 2)
		if e.jump != tc.jump {
			t.Errorf("expand of %v set jump to %v; expected %v", tc.kind, e.jump, tc.jump)
		}
	}
}
