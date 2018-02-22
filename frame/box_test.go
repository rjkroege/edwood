package frame

import (
	"reflect"
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

func TestAddbox(t *testing.T) {

//	zerobox := new(frbox)
	hellobox := makeBox("hi")
	worldbox := makeBox("world")	
//	byebox := makeBox("bye")	

	testvector := []struct {
		name string
		frame *Frame
		bn int
		n int
		nbox int
		nalloc int
		afterboxes []*frbox
		
	} {
		{
			"empty frame",
			&Frame{
				nbox: 0,
				nalloc:0,
			},
			0, 1,   1, 26,
			[]*frbox{},
		 },
		{
			"one element frame",
			&Frame{
				nbox: 1,
				nalloc:2,
				box: []*frbox{hellobox, nil},
			},
			0, 1,   2, 2,
			[]*frbox{hellobox, hellobox},
		 },
		{
			"two element frame",
			&Frame{
				nbox: 2,
				nalloc:2,
				box: []*frbox{hellobox, worldbox},
			},
			0, 1,   3, 28,
			[]*frbox{hellobox, hellobox, worldbox},
		 },
		{
			"two element frame",
			&Frame{
				nbox: 2,
				nalloc:2,
				box: []*frbox{hellobox, worldbox},
			},
			1, 1,   3, 28,
			[]*frbox{hellobox, worldbox, worldbox},
		 },
	}

	for _, tv := range testvector {   
		tv.frame.addbox(tv.bn, tv.n)
		if got, want := tv.frame.nbox, tv.nbox; got != want {
			t.Errorf("%s: nbox got %d but want %d\n", tv.name, got, want)
		}
		if got, want := tv.frame.nalloc, tv.nalloc; got != want {
			t.Errorf("%s: nalloc got %d but want %d\n", tv.name, got, want)
		}

		if tv.frame.box == nil {
			t.Errorf("%s: ran add but did not succeed in creating boxex", tv.name)
		}

		// First part of box array must match the provided afterboxes slice.
		for i, _ := range tv.afterboxes {
			// t.Logf("%s [%d]  %#v", tv.name,  i, tv.frame.box[i])
			if got, want := tv.frame.box[i], tv.afterboxes[i]; !reflect.DeepEqual(got, want) {
				switch {
				case got ==  nil && want != nil:
					t.Errorf("%s: result box [%d] mismatch: got nil want %#v (%s)", tv.name, i, want, string(want.Ptr))
				case got != nil && want == nil:
					t.Errorf("%s: result box [%d] mismatch: got %#v (%s) want nil", tv.name, i, got, string(got.Ptr))
				case got.Ptr == nil && want.Ptr == nil:
					t.Errorf("%s: result box [%d] mismatch: got %#v (nil) want %#v (nil)", tv.name, i, got, want)
				case got.Ptr == nil && want.Ptr != nil:
					t.Errorf("%s: result box [%d] mismatch: got %#v (nil) want %#v (%s)", tv.name, i, got, want, string(want.Ptr))
				case want.Ptr == nil && got.Ptr != nil:
					t.Errorf("%s: result box [%d] mismatch: got %#v (%s) want %#v (nil)", tv.name, i, got, string(got.Ptr), want)
				case want.Ptr != nil && got.Ptr != nil:
					t.Errorf("%s: result box [%d] mismatch: got %#v (%s) want %#v (%s)", tv.name, i, got, string(got.Ptr), want, string(want.Ptr))
				}
			}
		}

		// Remaining part of box array must merely exist.
		for i, b := range tv.frame.box[len(tv.afterboxes):] {
			if b != nil {
				t.Errorf("%s: result box [%d] should be nil", tv.name, i + len(tv.afterboxes))
			}
		}
	}


}