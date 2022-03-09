package file

import (
	"io"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBufferCursor(t *testing.T) {
	tt := []struct {
		name       string
		buf        []string
		p0         int
		p1         int
		wantstring string
	}{
		{
			name:       "three bufs, not-ASCII, start of end piece",
			buf:        []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			p0:         1,
			p1:         8,
			wantstring: strings.Join([]string{"苦本身", "痛ö本", "a"}, ""),
		},
		{
			name:       "three bufs, not-ASCII, start of end piece",
			buf:        []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			p0:         0,
			p1:         8,
			wantstring: strings.Join([]string{"痛苦本身", "痛ö本", "a"}, ""),
		},
		{
			name:       "three bufs, not-ASCII, start of end piece",
			buf:        []string{"a苦痛苦本b"},
			p0:         4,
			p1:         6,
			wantstring: "本b",
		},
	}

	for _, tv := range tt {
		t.Run(tv.name, func(t *testing.T) {
			b := NewBufferNoNr(nil)
			for _, s := range tv.buf {
				b.insertString(b.Nr(), s, t)
			}
			b.checkPiecesCnt(t, 2+len(tv.buf))

			cursor := MakeBufferCursor(b, b.RuneTuple(tv.p0), b.RuneTuple(tv.p1))

			for i, tr := range tv.wantstring {
				r, sz, err := cursor.ReadRune()

				if got, want := r, tr; got != want {
					t.Errorf("something went wrong? %d got '%c' %d, want '%c'", i, got, got, want)
				}

				if got, want := sz, utf8.RuneLen(tr); got != want {
					t.Errorf("something went wrong? %d got '%c' %d, want '%c'", i, got, got, want)
				}

				if err != nil {
					t.Errorf("unexpected error at %d: %v", i, err)
				}
			}

			if _, _, err := cursor.ReadRune(); err != io.EOF {
				t.Errorf("didn't signal EOF at end of cursor: %v", err)
			}

		})

	}
}
