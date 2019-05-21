package frame

import (
	"testing"
)

type SimpleBoxModelTest struct {
	name       string
	frame      *frameimpl
	stim       func(*frameimpl)
	nbox       int
	afterboxes []*frbox
}

func (bx SimpleBoxModelTest) Try() interface{} {
	bx.stim(bx.frame)
	return struct{}{}
}

func (bx SimpleBoxModelTest) Verify(t *testing.T, prefix string, result interface{}) {
	testcore(t, prefix, bx.name, bx.frame, bx.nbox, bx.afterboxes)
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
	frame := &frameimpl{
		font: mockFont(),
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
	frame := &frameimpl{
		font: mockFont(),
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
			&frameimpl{},
			func(f *frameimpl) { f.addbox(0, 1) },
			1,
			[]*frbox{nil},
		},
		SimpleBoxModelTest{
			"one element frame",
			&frameimpl{
				box: []*frbox{hellobox},
			},
			func(f *frameimpl) { f.addbox(0, 1) },
			2,
			[]*frbox{hellobox, hellobox},
		},
		SimpleBoxModelTest{
			"two element frame",
			&frameimpl{
				box: []*frbox{hellobox, worldbox},
			},
			func(f *frameimpl) { f.addbox(0, 1) },
			3,
			[]*frbox{hellobox, hellobox, worldbox},
		},
		SimpleBoxModelTest{
			"two element frame",
			&frameimpl{
				box: []*frbox{hellobox, worldbox},
			},
			func(f *frameimpl) { f.addbox(1, 1) },
			3,
			[]*frbox{hellobox, worldbox, worldbox},
		},
		SimpleBoxModelTest{
			"at very end of 2 element frame",
			&frameimpl{
				box: []*frbox{hellobox, worldbox},
			},
			func(f *frameimpl) { f.addbox(2, 2) },
			4,
			[]*frbox{hellobox, worldbox, nil, nil},
		},
	})
}

func TestClosebox(t *testing.T) {
	hellobox := makeBox("hi")
	worldbox := makeBox("world")

	comparecore(t, "TestClosebox", []BoxTester{
		SimpleBoxModelTest{
			"one element frame",
			&frameimpl{
				box: []*frbox{hellobox},
			},
			func(f *frameimpl) { f.closebox(0, 0) },
			0,
			[]*frbox{},
		},
		SimpleBoxModelTest{
			"two element frame 0",
			&frameimpl{
				box: []*frbox{hellobox, worldbox},
			},
			func(f *frameimpl) { f.closebox(0, 0) },
			1,
			[]*frbox{worldbox},
		},
		SimpleBoxModelTest{
			"two element frame 1",
			&frameimpl{
				box: []*frbox{hellobox, worldbox},
			},
			func(f *frameimpl) { f.closebox(1, 1) },
			1,
			[]*frbox{hellobox},
		},
		SimpleBoxModelTest{
			"three element frame",
			&frameimpl{
				box: []*frbox{hellobox, worldbox, hellobox},
			},
			func(f *frameimpl) { f.closebox(1, 1) },
			2,
			[]*frbox{hellobox, hellobox},
		},
	})
}

func TestDupbox(t *testing.T) {
	hellobox := makeBox("hi")

	stim := SimpleBoxModelTest{
		"one element frame",
		&frameimpl{
			box: []*frbox{hellobox},
		},
		func(f *frameimpl) { f.dupbox(0) },
		2,
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
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{makeBox("hiworld")},
			},
			func(f *frameimpl) { f.splitbox(0, 2) },
			2,
			[]*frbox{hibox, worldbox},
		},
		SimpleBoxModelTest{
			"two element frame 1",
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{worldbox, makeBox("hiworld")},
			},
			func(f *frameimpl) { f.splitbox(1, 2) },
			3,
			[]*frbox{worldbox, hibox, worldbox},
		},
		SimpleBoxModelTest{
			"one element 0, 0",
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{makeBox("hi")},
			},
			func(f *frameimpl) { f.splitbox(0, 0) },
			2,
			[]*frbox{zerobox, hibox},
		},
		SimpleBoxModelTest{
			"one element 0, 2",
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{makeBox("hi")},
			},
			func(f *frameimpl) { f.splitbox(0, 2) },
			2,
			[]*frbox{hibox, zerobox},
		},
		SimpleBoxModelTest{
			"one element 0, 2",
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{makeBox("hi")},
			},
			func(f *frameimpl) { f.splitbox(0, 2) },
			2,
			[]*frbox{hibox, zerobox},
		},
		/*
			SimpleBoxModelTest{
				"no element 0, 0",
				&Frame{
					Font:   mockFont(),
					box:    []*frbox{},
				},
				func(f *Frame) { f.splitbox(0, 0) },
				2,
				[]*frbox{hibox, zerobox},
			},
		*/
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
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{hibox, worldbox},
			},
			func(f *frameimpl) { f.mergebox(0) },
			1,
			[]*frbox{hiworldbox},
		},
		SimpleBoxModelTest{
			"two null -> 1",
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{hibox, zerobox},
			},
			func(f *frameimpl) { f.mergebox(0) },
			1,
			[]*frbox{hibox},
		},
		SimpleBoxModelTest{
			"three -> 2",
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{makeBox("hi"), worldbox, hibox},
			},
			func(f *frameimpl) { f.mergebox(0) },
			2,
			[]*frbox{hiworldbox, hibox},
		},
		SimpleBoxModelTest{
			"three -> 1",
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{makeBox("hi"), makeBox("world"), makeBox("hi")},
			},
			func(f *frameimpl) {
				f.mergebox(1)
				f.mergebox(0)
			},
			1,
			[]*frbox{makeBox("hiworldhi")},
		},
	})
}

type FindBoxModelTest struct {
	name       string
	frame      *frameimpl
	stim       func(*frameimpl) int
	nbox       int
	afterboxes []*frbox
	foundbox   int
}

func (bx FindBoxModelTest) Try() interface{} {
	return bx.stim(bx.frame)
}

func (bx FindBoxModelTest) Verify(t *testing.T, prefix string, result interface{}) {
	r := result.(int)
	testcore(t, prefix, bx.name, bx.frame, bx.nbox, bx.afterboxes)
	if got, want := r, bx.foundbox; got != want {
		t.Errorf("%s-%s: running stim got %d but want %d\n", prefix, bx.name, got, want)
	}
}

func TestFindbox(t *testing.T) {
	hibox := makeBox("hi")
	worldbox := makeBox("world")
	hiworldbox := makeBox("hiworld")

	comparecore(t, "TestFindbox", []BoxTester{
		FindBoxModelTest{
			"find in 1",
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{makeBox("hiworld")},
			},
			func(f *frameimpl) int { return f.findbox(0, 0, 2) },
			2,
			[]*frbox{hibox, worldbox},
			1,
		},
		FindBoxModelTest{
			"find at beginning",
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{makeBox("hiworld")},
			},
			func(f *frameimpl) int { return f.findbox(0, 0, 0) },
			1,
			[]*frbox{hiworldbox},
			0,
		},
		FindBoxModelTest{
			"find at edge",
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{makeBox("hi"), makeBox("world")},
			},
			func(f *frameimpl) int { return f.findbox(0, 0, 2) },
			2,
			[]*frbox{hibox, worldbox},
			1,
		},
		FindBoxModelTest{
			"find continuing",
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{makeBox("hi"), makeBox("world")},
			},
			func(f *frameimpl) int { return f.findbox(1, 0, 2) },
			3,
			[]*frbox{hibox, makeBox("wo"), makeBox("rld")},
			2,
		},
		FindBoxModelTest{
			"find in empty",
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{},
			},
			func(f *frameimpl) int { return f.findbox(0, 0, 0) },
			0,
			[]*frbox{},
			0,
		},
		FindBoxModelTest{
			"find at end",
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{makeBox("hi"), makeBox("world")},
			},
			func(f *frameimpl) int { return f.findbox(0, 0, 7) },
			2,
			[]*frbox{hibox, worldbox},
			2,
		},
		FindBoxModelTest{
			"find very near end",
			&frameimpl{
				font: mockFont(),
				box:  []*frbox{makeBox("hi"), makeBox("world")},
			},
			func(f *frameimpl) int { return f.findbox(1, 2, 6) },
			3,
			[]*frbox{hibox, makeBox("worl"), makeBox("d")},
			2,
		},
	})
}
