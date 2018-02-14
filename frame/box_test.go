package frame

import (
	"strings"
	"testing"
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


func makeBox(s string) *frbox {
	return &frbox{
		Wid: 0,
		Nrune: strings.Count(s, "") - 1,
		Ptr: []byte(s),
		// Remaining fields mysterious.
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

		pb.truncatebox(ps.at, fakemetrics(10))
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

		pb.chopbox(ps.at, fakemetrics(10))
		if ab.Nrune != pb.Nrune || string(ab.Ptr) != string(pb.Ptr) {
			t.Errorf("truncating %#v (%#v) at %d failed to provide %#v. Gave %#v (%s)\n",
				makeBox(ps.before), ps.before, ps.at, ps.after, pb, string(pb.Ptr))
		}
	}

}

// TODO(rjk): test addbox
