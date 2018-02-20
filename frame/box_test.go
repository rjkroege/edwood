package frame

import (
	"strings"
	"testing"
	"unicode/utf8"
)


func TestRunIndex(t *testing.T) {

	testvector := []struct {
		thestring string
		arg int
		want int
	} {
		{ "", 0, 0 },
		{ "a\x02b", 0, 0 },
		{ "a\x02b", 1, 1 },
		{ "a\x02b", 2, 2 },
		{ "a\x02日本b", 0, 0 },
		{ "a\x02日本b", 1, 1 },
		{ "a\x02日本b", 2, 2 },
		{ "a\x02日本b", 3, 5 },
		{ "a\x02日本b", 4, 8 },
		{ "Kröeger", 3, 4 },
	}
		
	
	for _, ps := range testvector {
		b := ps.thestring

		if got, want := runeindex([]byte(b),ps.arg), ps.want; got != want {
			t.Errorf("comparing %#v at %d got %d, want %d", b, ps.arg, got, want)
		}
	}
}

const fixedwidth = 10

// makeBox creates somewhat realistic test boxes in 10pt fixed width font.
func makeBox(s string) *frbox {

	r, _ := utf8.DecodeRuneInString(s)

	switch s {
	case "\t":
		return &frbox{
			Wid: 5000,
			Nrune: -1,
			Ptr: []byte(s),
			Bc: r,
			Minwid: 10,
		}

	case "\n":
		return &frbox{
			Wid: 5000,
			Nrune: -1,
			Ptr: []byte(s),
			Bc: r,
			Minwid: 0,
		}
	default:
		nrune := strings.Count(s, "") - 1
		return &frbox{
			Wid: fixedwidth * nrune,
			Nrune: nrune,
			Ptr: []byte(s),
			// Remaining fields not used.
		}
	}
}

type fakemetrics int

func (w fakemetrics) BytesWidth([]byte) int {
	return int(w)
}

func TestTruncatebox(t *testing.T) {

	testvector := []struct {
		before string
		after string
		at int
	} {
		{ "ab", "a", 1 },
		{ "abc", "a", 2 },
		{ "a\x02日本b", "a", 4 },
	}
		
	
	for _, ps := range testvector {
		pb := makeBox(ps.before)
		ab := makeBox(ps.after)

		pb.truncatebox(ps.at, fakemetrics(fixedwidth))
		if ab.Nrune != pb.Nrune || string(ab.Ptr) != string(pb.Ptr) {
			t.Errorf("truncating %#v (%#v) at %d failed to provide %#v. Gave %#v (%s)\n",
				makeBox(ps.before), ps.before, ps.at, ps.after, pb, string(pb.Ptr))
		}
	}

}

func TestChopbox(t *testing.T) {

	testvector := []struct {
		before string
		after string
		at int
	} {
		{ "ab", "b", 1 },
		{ "abc", "c", 2 },
		{ "a\x02日本b", "本b",3 },
	}
		
	
	for _, ps := range testvector {
		pb := makeBox(ps.before)
		ab := makeBox(ps.after)

		pb.chopbox(ps.at, fakemetrics(fixedwidth))
		if ab.Nrune != pb.Nrune || string(ab.Ptr) != string(pb.Ptr) {
			t.Errorf("truncating %#v (%#v) at %d failed to provide %#v. Gave %#v (%s)\n",
				makeBox(ps.before), ps.before, ps.at, ps.after, pb, string(pb.Ptr))
		}
	}

}

// TODO(rjk): test addbox
