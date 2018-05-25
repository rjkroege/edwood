package frame

import (
	"image"
	"testing"
)

type CharofptTestCase struct {
	name     string
	frame    *Frame
	stim     image.Point
	expected int
}

func TestCharofpt(t *testing.T) {

	for _, tv := range []CharofptTestCase{
		{
			"empty",
			&Frame{
				font:              Fakemetrics(fixedwidth),
				defaultfontheight: 13,
				rect:              image.Rect(10, 15, 10+57, 15+57),
			},
			image.Pt(10+56, 15+56),
			0,
		},
		{
			"one box",
			&Frame{
				font:              Fakemetrics(fixedwidth),
				defaultfontheight: 13,
				box: []*frbox{
					makeBox("本"),
				},
				rect: image.Rect(10, 15, 10+57, 15+57),
			},
			image.Pt(10+56, 15+56),
			1,
		},
		{
			"two boxes, target first pixel of first char",
			&Frame{
				font:              Fakemetrics(fixedwidth),
				defaultfontheight: 13,
				box: []*frbox{
					makeBox("12345"),
					makeBox("本b"),
				},
				rect: image.Rect(10, 15, 10+57, 15+57),
			},
			image.Pt(10, 15),
			0,
		},
		{
			"two boxes, last pixel in first char",
			&Frame{
				font:              Fakemetrics(fixedwidth),
				defaultfontheight: 13,
				box: []*frbox{
					makeBox("12345"),
					makeBox("本b"),
				},
				rect: image.Rect(10, 15, 10+57, 15+57),
			},
			image.Pt(19, 27),
			0,
		},
		{
			"two boxes, bottom edge of second char",
			&Frame{
				font:              Fakemetrics(fixedwidth),
				defaultfontheight: 13,
				box: []*frbox{
					makeBox("12345"),
					makeBox("本b"),
				},
				rect: image.Rect(10, 15, 10+57, 15+57),
			},
			image.Pt(20, 27),
			1,
		},
		{
			"two boxes, top edge of second box",
			&Frame{
				font:              Fakemetrics(fixedwidth),
				defaultfontheight: 13,
				box: []*frbox{
					makeBox("12345"),
					makeBox("本bcd"),
				},
				rect: image.Rect(10, 15, 10+57, 15+57),
			},
			image.Pt(19, 28),
			5,
		},
		{
			"two boxes, top edge of second box",
			&Frame{
				font:              Fakemetrics(fixedwidth),
				defaultfontheight: 13,
				box: []*frbox{
					makeBox("12345"),
					makeBox("本bcd"),
				},
				rect: image.Rect(10, 15, 10+57, 15+57),
			},
			image.Pt(20, 28),
			6,
		},
		{
			"three boxes, top edge of second box",
			&Frame{
				font:              Fakemetrics(fixedwidth),
				defaultfontheight: 13,
				box: []*frbox{
					makeBox("12345"),
					makeBox("本bcd"),
					makeBox("Göph"),
				},
				rect: image.Rect(10, 15, 10+57, 15+57),
			},
			image.Pt(20, 28),
			6,
		},
		{
			"three boxes, top edge of second box",
			&Frame{
				font:              Fakemetrics(fixedwidth),
				defaultfontheight: 13,
				box: []*frbox{
					makeBox("12345"),
					makeBox("本bcd"),
					makeBox("Göph"),
				},
				rect: image.Rect(10, 15, 10+57, 15+57),
			},
			image.Pt(30, 1+15+2*13),
			11,
		},
	} {
		if got, want := tv.frame.Charofpt(tv.stim), tv.expected; got != want {
			t.Errorf("TestCharofpt(%v), case %s, got %d, want %d", tv.stim, tv.name, got, want)
		}
	}
}
