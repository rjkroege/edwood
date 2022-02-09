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

func TestNewDeleteAt(t *testing.T) {
	b := NewTypeBuffer([]rune("hello"), nil)

	b.DeleteAt(0, 2, 0)

	if got, want := b.String(), "llo"; got != want {
		t.Errorf("didn't run delete correctly got %q want %q", got, want)
	}
	if got, want := b.HasUndoableChanges(), false; got != want {
		t.Errorf("HasUndoableChanges wrong got %v want %v", got, want)
	}
	if got, want := b.HasRedoableChanges(), false; got != want {
		t.Errorf("HasRedoableChanges wrong got %v want %v", got, want)
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
