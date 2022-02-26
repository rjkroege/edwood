package file

import (
	"fmt"
	"io"
	"testing"
)

func TestNewTypeBufferCreation(t *testing.T) {
	b := NewTypeBuffer([]rune("hello"), nil)

	if got, want := b.String(), "hello"; got != want {
		t.Errorf("didn't load right got %q want %q", got, want)
	}
}

func TestNewIndexRune(t *testing.T) {
	b := NewTypeBuffer([]rune("yi 海老hi 海老麺麺"), nil)

	for _, tc := range []struct {
		r      rune
		offset int
	}{
		{
			r:      'y',
			offset: 0,
		},
		{
			r:      'h',
			offset: 5,
		},
		{
			r:      '|',
			offset: -1,
		},
	} {
		t.Run(fmt.Sprintf("%c->%d", tc.r, tc.offset), func(t *testing.T) {
			if got, want := b.IndexRune(tc.r), tc.offset; got != want {
				t.Errorf("IndexRune failed, got  %d want %d", got, want)
			}

		})
	}
}

func TestNewRead(t *testing.T) {
	b := NewTypeBuffer([]rune("yi 海老hi 海老麺麺"), nil)

	for _, tc := range []struct {
		o   int
		r   []rune
		n   int
		gr  string
		err error
	}{
		{
			o:  0,
			r:  make([]rune, 1),
			n:  1,
			gr: "y",
		},
		{
			o:  3,
			r:  make([]rune, 4),
			n:  4,
			gr: "海老hi",
		},
		{
			o:   3,
			r:   make([]rune, 10),
			n:   9,
			gr:  "海老hi 海老麺麺",
			err: io.EOF,
		},
	} {
		t.Run(tc.gr, func(t *testing.T) {

			// Test here
			n, err := b.Read(tc.o, tc.r)
			if got, want := err, tc.err; got != want {
				t.Errorf("got error %v want %v", got, want)
			}
			if got, want := n, tc.n; got != want {
				t.Errorf("got length %d want %d", got, want)
			}
			if got, want := string(tc.r[0:n]), tc.gr; got != want {
				t.Errorf("got value %q want %q", got, want)
			}

		})
	}
}

func TestReadCSplit(t *testing.T) {
	b := NewBuffer([]byte("hello\n1 2 3 4\nfoo"), len("hello\n1 2 3 4\nfoo"))

	t.Log("before insert", b.viewedState())

	b.Insert(b.RuneTuple(len("hello\n1 2")), []byte("X"), 1, 1)

	t.Log("after insert", b.viewedState())

	s := "hello\n1 2X 3 4\nfoo"
	if got, want := b.String(), s; got != want {
		t.Errorf("buffer contents not as expected got %q, want %q", got, want)
	}

	for i, r := range s {
		if got, want := b.ReadC(i), r; got != want {
			t.Errorf("something went wrong? %d got '%c' %d, want '%c'", i, got, got, want)
		}
	}
}
