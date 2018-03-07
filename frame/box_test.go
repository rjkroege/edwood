package frame

import (
	"testing"
)

type SimpleBoxModelTest struct {
	name       string
	frame      *Frame
	stim       func(*Frame)
	nbox       int
	nalloc     int
	afterboxes []*frbox
}

func (bx SimpleBoxModelTest) Try() interface{} {
	bx.stim(bx.frame)
	return struct{}{}
}

func (tv SimpleBoxModelTest) Verify(t *testing.T, prefix string, result interface{}) {
	testcore(t, prefix, tv.name, tv.frame, tv.nbox, tv.nalloc, tv.afterboxes)
}

func TestRunIndex(t *testing.T) {

	testvector := []struct {
		thestring string
		arg       int
		want      int
	}{
		{"", 0, 0},
		{"a\x02b", 0, 0},
		{"a\x02b", 1, 1},
		{"a\x02b", 2, 2},
		{"a\x02日本b", 0, 0},
		{"a\x02日本b", 1, 1},
		{"a\x02日本b", 2, 2},
		{"a\x02日本b", 3, 5},
		{"a\x02日本b", 4, 8},
		{"Kröger", 3, 4},
		{"本a", 1, 3},
	}

	for _, ps := range testvector {
		b := ps.thestring

		if got, want := runeindex([]byte(b), ps.arg), ps.want; got != want {
			t.Errorf("comparing %#v at %d got %d, want %d", b, ps.arg, got, want)
		}
	}
}

func TestTruncatebox(t *testing.T) {
	frame := &Frame{
		Font:   Fakemetrics(fixedwidth),
		nbox:   0,
		nalloc: 0,
	}

	testvector := []struct {
		before string
		after  string
		at     int
	}{
		{"ab", "a", 1},
		{"abc", "a", 2},
		{"a\x02日本b", "a", 4},
	}

	for _, ps := range testvector {
		pb := makeBox(ps.before)
		ab := makeBox(ps.after)

		frame.truncatebox(pb, ps.at)
		if ab.Nrune != pb.Nrune || string(ab.Ptr) != string(pb.Ptr) {
			t.Errorf("truncating %#v (%#v) at %d failed to provide %#v. Gave %#v (%s)\n",
				makeBox(ps.before), ps.before, ps.at, ps.after, pb, string(pb.Ptr))
		}

		if ab.Wid != pb.Wid {
			t.Errorf("wrong width: got %d, want %d for %s", pb.Wid, ab.Wid, string(pb.Ptr))
		}
	}
}

func TestChopbox(t *testing.T) {
	frame := &Frame{
		Font:   Fakemetrics(fixedwidth),
		nbox:   0,
		nalloc: 0,
	}

	testvector := []struct {
		before string
		after  string
		at     int
	}{
		{"ab", "b", 1},
		{"abc", "c", 2},
		{"a\x02日本b", "本b", 3},
	}

	for _, ps := range testvector {
		pb := makeBox(ps.before)
		ab := makeBox(ps.after)

		frame.chopbox(pb, ps.at)
		if ab.Nrune != pb.Nrune || string(ab.Ptr) != string(pb.Ptr) {
			t.Errorf("truncating %#v (%#v) at %d failed to provide %#v. Gave %#v (%s)\n",
				makeBox(ps.before), ps.before, ps.at, ps.after, pb, string(pb.Ptr))
		}

		if ab.Wid != pb.Wid {
			t.Errorf("wrong width: got %d, want %d for %s", pb.Wid, ab.Wid, string(pb.Ptr))
		}
	}
}

func TestAddbox(t *testing.T) {
	hellobox := makeBox("hi")
	worldbox := makeBox("world")

	comparecore(t, "TestAddbox", []BoxTester{
		SimpleBoxModelTest{
			"empty frame",
			&Frame{
				nbox:   0,
				nalloc: 0,
			},
			func(f *Frame) { f.addbox(0, 1) },
			1, 26,
			[]*frbox{},
		},
		SimpleBoxModelTest{
			"one element frame",
			&Frame{
				nbox:   1,
				nalloc: 2,
				box:    []*frbox{hellobox, nil},
			},
			func(f *Frame) { f.addbox(0, 1) },
			2, 2,
			[]*frbox{hellobox, hellobox},
		},
		SimpleBoxModelTest{
			"two element frame",
			&Frame{
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{hellobox, worldbox},
			},
			func(f *Frame) { f.addbox(0, 1) },
			3, 28,
			[]*frbox{hellobox, hellobox, worldbox},
		},
		SimpleBoxModelTest{
			"two element frame",
			&Frame{
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{hellobox, worldbox},
			},
			func(f *Frame) { f.addbox(1, 1) },
			3, 28,
			[]*frbox{hellobox, worldbox, worldbox},
		},
	})
}

func TestFreebox(t *testing.T) {
	hellobox := makeBox("hi")
	worldbox := makeBox("world")

	comparecore(t, "TestFreebox", []BoxTester{
		SimpleBoxModelTest{
			"one element frame",
			&Frame{
				nbox:   1,
				nalloc: 2,
				box:    []*frbox{hellobox, nil},
			},
			func(f *Frame) { f.freebox(0, 0) },
			1, 2,
			[]*frbox{nil},
		},
		SimpleBoxModelTest{
			"two element frame 0",
			&Frame{
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{hellobox, worldbox},
			},
			func(f *Frame) { f.freebox(0, 0) },
			2, 2,
			[]*frbox{nil, worldbox},
		},
		SimpleBoxModelTest{
			"two element frame 1",
			&Frame{
				nbox:   3,
				nalloc: 3,
				box:    []*frbox{hellobox, worldbox, hellobox},
			},
			func(f *Frame) { f.freebox(1, 1) },
			3, 3,
			[]*frbox{hellobox, nil, hellobox},
		},
	})
}

func TestClosebox(t *testing.T) {
	hellobox := makeBox("hi")
	worldbox := makeBox("world")

	comparecore(t, "TestClosebox", []BoxTester{
		SimpleBoxModelTest{
			"one element frame",
			&Frame{
				nbox:   1,
				nalloc: 2,
				box:    []*frbox{hellobox, nil},
			},
			func(f *Frame) { f.closebox(0, 0) },
			0, 2,
			[]*frbox{nil},
		},
		SimpleBoxModelTest{
			"two element frame 0",
			&Frame{
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{hellobox, worldbox},
			},
			func(f *Frame) { f.closebox(0, 0) },
			1, 2,
			[]*frbox{worldbox},
		},
		SimpleBoxModelTest{
			"two element frame 1",
			&Frame{
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{hellobox, worldbox},
			},
			func(f *Frame) { f.closebox(1, 1) },
			1, 2,
			[]*frbox{hellobox},
		},
		SimpleBoxModelTest{
			"three element frame",
			&Frame{
				nbox:   3,
				nalloc: 3,
				box:    []*frbox{hellobox, worldbox, hellobox},
			},
			func(f *Frame) { f.closebox(1, 1) },
			2, 3,
			[]*frbox{hellobox, hellobox},
		},
	})
}

func TestDupbox(t *testing.T) {
	hellobox := makeBox("hi")

	stim := SimpleBoxModelTest{
		"one element frame",
		&Frame{
			nbox:   1,
			nalloc: 2,
			box:    []*frbox{hellobox, nil},
		},
		func(f *Frame) { f.dupbox(0) },
		2, 2,
		[]*frbox{hellobox, hellobox},
	}
	comparecore(t, "TestDupbox", []BoxTester{
		stim,
	})

	// Specifically must verify that the box string is different.
	if stim.frame.box[0] == stim.frame.box[1] {
		t.Errorf("dupbox failed to make a copy of the backing rune string")
	}
}

func TestSplitbox(t *testing.T) {
	hibox := makeBox("hi")
	worldbox := makeBox("world")
	zerobox := makeBox("")

	comparecore(t, "TestSplitbox", []BoxTester{
		SimpleBoxModelTest{
			"one element frame",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 2,
				box:    []*frbox{makeBox("hiworld"), nil},
			},
			func(f *Frame) { f.splitbox(0, 2) },
			2, 2,
			[]*frbox{hibox, worldbox},
		},
		SimpleBoxModelTest{
			"two element frame 1",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   2,
				nalloc: 3,
				box:    []*frbox{worldbox, makeBox("hiworld"), nil},
			},
			func(f *Frame) { f.splitbox(1, 2) },
			3, 3,
			[]*frbox{worldbox, hibox, worldbox},
		},
		SimpleBoxModelTest{
			"one element 0, 0",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 2,
				box:    []*frbox{makeBox("hi"), nil},
			},
			func(f *Frame) { f.splitbox(0, 0) },
			2, 2,
			[]*frbox{zerobox, hibox},
		},
		SimpleBoxModelTest{
			"one element 0, 2",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 2,
				box:    []*frbox{makeBox("hi"), nil},
			},
			func(f *Frame) { f.splitbox(0, 2) },
			2, 2,
			[]*frbox{hibox, zerobox},
		},
		SimpleBoxModelTest{
			"one element 0, 2",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 2,
				box:    []*frbox{makeBox("hi"), nil},
			},
			func(f *Frame) { f.splitbox(0, 2) },
			2, 2,
			[]*frbox{hibox, zerobox},
		},
	})
}

func TestMergebox(t *testing.T) {
	hibox := makeBox("hi")
	worldbox := makeBox("world")
	hiworldbox := makeBox("hiworld")
	zerobox := makeBox("")

	comparecore(t, "TestMergebox", []BoxTester{
		SimpleBoxModelTest{
			"two -> 1",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{hibox, worldbox},
			},
			func(f *Frame) { f.mergebox(0) },
			1, 2,
			[]*frbox{hiworldbox},
		},
		SimpleBoxModelTest{
			"two null -> 1",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{hibox, zerobox},
			},
			func(f *Frame) { f.mergebox(0) },
			1, 2,
			[]*frbox{hibox},
		},
		SimpleBoxModelTest{
			"three -> 2",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   3,
				nalloc: 3,
				box:    []*frbox{makeBox("hi"), worldbox, hibox},
			},
			func(f *Frame) { f.mergebox(0) },
			2, 3,
			[]*frbox{hiworldbox, hibox},
		},
		SimpleBoxModelTest{
			"three -> 1",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   3,
				nalloc: 3,
				box:    []*frbox{makeBox("hi"), makeBox("world"), makeBox("hi")},
			},
			func(f *Frame) {
				f.mergebox(1)
				f.mergebox(0)
			},
			1, 3,
			[]*frbox{makeBox("hiworldhi")},
		},
	})
}

type FindBoxModelTest struct {
	name       string
	frame      *Frame
	stim       func(*Frame) int
	nbox       int
	nalloc     int
	afterboxes []*frbox
	foundbox   int
}

func (bx FindBoxModelTest) Try() interface{} {
	return bx.stim(bx.frame)
}

func (tv FindBoxModelTest) Verify(t *testing.T, prefix string, result interface{}) {
	r := result.(int)
	testcore(t, prefix, tv.name, tv.frame, tv.nbox, tv.nalloc, tv.afterboxes)
	if got, want := r, tv.foundbox; got != want {
		t.Errorf("%s-%s: running stim got %d but want %d\n", prefix, tv.name, got, want)
	}
}

func TestFindbox(t *testing.T) {
	hibox := makeBox("hi")
	worldbox := makeBox("world")
	hiworldbox := makeBox("hiworld")

	comparecore(t, "TestFindbox", []BoxTester{
		FindBoxModelTest{
			"find in 1",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 1,
				box:    []*frbox{makeBox("hiworld")},
			},
			func(f *Frame) int { return f.findbox(0, 0, 2) },
			2, 27,
			[]*frbox{hibox, worldbox},
			1,
		},
		FindBoxModelTest{
			"find at beginning",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 1,
				box:    []*frbox{makeBox("hiworld")},
			},
			func(f *Frame) int { return f.findbox(0, 0, 0) },
			1, 1,
			[]*frbox{hiworldbox},
			0,
		},
		FindBoxModelTest{
			"find at edge",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{makeBox("hi"), makeBox("world")},
			},
			func(f *Frame) int { return f.findbox(0, 0, 2) },
			2, 2,
			[]*frbox{hibox, worldbox},
			1,
		},
		FindBoxModelTest{
			"find continuing",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   2,
				nalloc: 2,
				box:    []*frbox{makeBox("hi"), makeBox("world")},
			},
			func(f *Frame) int { return f.findbox(1, 0, 2) },
			3, 28,
			[]*frbox{hibox, makeBox("wo"), makeBox("rld")},
			2,
		},
		FindBoxModelTest{
			"find in empty",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   0,
				nalloc: 2,
				box:    []*frbox{nil, nil},
			},
			func(f *Frame) int { return f.findbox(0, 0, 0) },
			0, 2,
			[]*frbox{},
			0,
		},
	})
}
