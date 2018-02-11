package frame

import (
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