package frame

import (
	"bytes"
	"image"
	"strings"
	"testing"
)

type InsertTestResult struct {
	ppt      image.Point
	resultpt image.Point
	frame    *Frame
}

type InsertTest struct {
	name       string
	frame      *Frame
	stim       func(*Frame) (image.Point, image.Point, *Frame)
	nbox       int
	afterboxes []*frbox
	ppt        image.Point
	resultpt   image.Point
}

func (bx InsertTest) Try() interface{} {
	a, b, c := bx.stim(bx.frame)
	return InsertTestResult{a, b, c}
}

func (tv InsertTest) Verify(t *testing.T, prefix string, result interface{}) {
	r := result.(InsertTestResult)

	if got, want := r.ppt, tv.ppt; got != want {
		t.Errorf("%s-%s: running stim ppt got %d but want %d\n", prefix, tv.name, got, want)
	}
	if got, want := r.resultpt, tv.resultpt; got != want {
		t.Errorf("%s-%s: running stim resultpt got %d but want %d\n", prefix, tv.name, got, want)
	}
	// We use the global frame here to make sure that bxscan works as desired.
	// I note in passing that encapsulation here could be improved.
	testcore(t, prefix, tv.name, r.frame, tv.nbox,  tv.afterboxes)
}

func mkRu(s string) []rune {
	return bytes.Runes([]byte(s))
}

func TestBxscan(t *testing.T) {
	var b strings.Builder
	b.WriteString("a本")
	for i := 0; i < (57 / 10); i++ {
		b.WriteString("ポ")
	}
	b.WriteString("hello")
	bigstring := b.String()

	comparecore(t, "TestBxscan", []BoxTester{
		InsertTest{
			"1 rune insertion into empty",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				Rect:   image.Rect(10, 15, 10+57, 15+57),
			},
			func(f *Frame) (image.Point, image.Point, *Frame) {
				pt1 := image.Pt(10, 15)
				pt2, f := f.bxscan(mkRu("本"), &pt1)
				return pt1, pt2, f
			},
			1, 
			[]*frbox{makeBox("本")},
			image.Pt(10, 15),
			image.Pt(20, 15),
		},
		InsertTest{
			"1 rune insertion fits at end of line",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				Rect:   image.Rect(10, 15, 10+57, 15+57),
			},
			func(f *Frame) (image.Point, image.Point, *Frame) {
				pt1 := image.Pt(56, 15)
				pt2, f := f.bxscan(mkRu("本"), &pt1)
				return pt1, pt2, f
			},
			1, 
			[]*frbox{makeBox("本")},
			image.Pt(56, 15),
			image.Pt(66, 15),
		},
		InsertTest{
			"1 rune insertion wraps at end of line",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				Rect:   image.Rect(10, 15, 10+57, 15+57),
			},
			func(f *Frame) (image.Point, image.Point, *Frame) {
				pt1 := image.Pt(57, 15)
				pt2, f := f.bxscan(mkRu("本"), &pt1)
				return pt1, pt2, f
			},
			1, 
			[]*frbox{makeBox("本")},
			image.Pt(10, 15+13),
			image.Pt(20, 15+13),
		},
		InsertTest{
			"splittable 2 rune insertion at end of line",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				Rect:   image.Rect(10, 15, 10+57, 15+57),
			},
			func(f *Frame) (image.Point, image.Point, *Frame) {
				pt1 := image.Pt(56, 15)
				pt2, f := f.bxscan(mkRu("本a"), &pt1)
				return pt1, pt2, f
			},
			2, 
			[]*frbox{makeBox("本"), makeBox("a")},
			image.Pt(56, 15),
			image.Pt(20, 15+13),
		},
		InsertTest{
			"splittable multi-rune rune insertion at start of line",
			&Frame{
				Font:   Fakemetrics(fixedwidth),
				Rect:   image.Rect(10, 15, 10+57, 15+57),
			},
			func(f *Frame) (image.Point, image.Point, *Frame) {
				pt1 := image.Pt(10, 15)
				pt2, f := f.bxscan(mkRu(bigstring), &pt1)
				return pt1, pt2, f
			},
			3, 
			[]*frbox{makeBox("a本ポポポ"), makeBox("ポポhel"), makeBox("lo")},
			image.Pt(10, 15),
			image.Pt(10+2*10, 15+13+13),
		},
	})
}
