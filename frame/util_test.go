package frame

import (
	"image"
	"testing"
)

type BoxModelTestResult struct {
	result     int
	boolresult bool
}

type BoxModelTest struct {
	name       string
	frame      *Frame
	stim       func(*Frame) (int, bool)
	nbox       int
	nalloc     int
	afterboxes []*frbox
	result     int
	boolresult bool
}

func (bx BoxModelTest) Try() interface{} {
	a, b := bx.stim(bx.frame)
	return BoxModelTestResult{
		result:     a,
		boolresult: b,
	}
}

func (tv BoxModelTest) Verify(t *testing.T, prefix string, result interface{}) {
	r := result.(BoxModelTestResult)

	if got, want := r.result, tv.result; got != want {
		t.Errorf("%s-%s: running stim got %d but want %d\n", prefix, tv.name, got, want)
	}
	if got, want := r.boolresult, tv.boolresult; got != want {
		t.Errorf("%s-%s: running stim bool got %v but want %v\n", prefix, tv.name, got, want)
	}

	testcore(t, prefix, tv.name, tv.frame, tv.nbox, tv.nalloc, tv.afterboxes)
}

func TestCanfit(t *testing.T) {
	newlinebox := makeBox("\n")
	tabbox := makeBox("\t")

	comparecore(t, "TestCanfit", []BoxTester{
		BoxModelTest{
			"multi-glyph box doesn't fit",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 1,
				Rect:   image.Rect(10, 15, 10+57, 15+57),
				box:    []*frbox{makeBox("0123456789")},
			},
			func(f *Frame) (int, bool) {
				a, b := f.canfit(image.Pt(10+14, 15), f.box[0])
				return a, b
			},
			1, 1,
			[]*frbox{makeBox("0123456789")},
			// 10 + 14 + 40 = 64. less than 67.
			4,
			true,
		},
		BoxModelTest{
			"multi-glyph box, fits",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 1,
				Rect:   image.Rect(10, 15, 10+57, 15+57),
				box:    []*frbox{makeBox("0123")},
			},
			func(f *Frame) (int, bool) {
				a, b := f.canfit(image.Pt(10+14, 15), f.box[0])
				return a, b
			},
			1, 1,
			[]*frbox{makeBox("0123")},
			// 10 + 14 + 40 = 64. less than 67.
			4,
			true,
		},
		BoxModelTest{
			"newline box",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 1,
				Rect:   image.Rect(10, 15, 10+57, 15+57),
				box:    []*frbox{newlinebox},
			},
			func(f *Frame) (int, bool) {
				a, b := f.canfit(image.Pt(10+57, 15), f.box[0])
				return a, b
			},
			1, 1,
			[]*frbox{newlinebox},
			// newline fits up to the edge.
			1,
			true,
		},
		BoxModelTest{
			"tab box",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 1,
				Rect:   image.Rect(10, 15, 10+57, 15+57),
				box:    []*frbox{tabbox},
			},
			func(f *Frame) (int, bool) {
				a, b := f.canfit(image.Pt(10+48, 15), f.box[0])
				return a, b
			},
			1, 1,
			[]*frbox{tabbox},
			// tab at edge doesn't  fit
			0,
			false,
		},
		BoxModelTest{
			"multi-glyph box, doesn't fit",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 1,
				Rect:   image.Rect(10, 15, 10+57, 15+57),
				box:    []*frbox{makeBox("本a")},
			},
			func(f *Frame) (int, bool) {
				a, b := f.canfit(image.Pt(10+57-11, 15), f.box[0])
				return a, b
			},
			1, 1,
			[]*frbox{makeBox("本a")},
			// 10 + 14 + 40 = 64. less than 67.
			1,
			true,
		},
	})
}
