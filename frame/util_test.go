package frame

import (
	"image"
	"testing"
)


func TestCanfit(t *testing.T) {
	newlinebox := makeBox("\n")
	tabbox := makeBox("\t")

	comparecore(t, "TestCanfit", []TestStim{
		{
			"multi-glyph box doesn't fit",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 1,
				Rect: image.Rect(10, 15, 10 + 57, 15 + 57),
				box: []*frbox{ makeBox("0123456789")  },
			},
			func(f *Frame) (int, bool) {
				a, b := f.canfit(image.Pt(10 + 14, 15), f.box[0]) 
				return a, b
			},
			1, 1,
			[]*frbox{ makeBox("0123456789")  },
			// 10 + 14 + 40 = 64. less than 67.
			4,
			true,
		},
		{
			"multi-glyph box, fits",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 1,
				Rect: image.Rect(10, 15, 10 + 57, 15 + 57),
				box: []*frbox{ makeBox("0123")  },
			},
			func(f *Frame) (int, bool) {
				a, b := f.canfit(image.Pt(10 + 14, 15), f.box[0]) 
				return a, b
			},
			1, 1,
			[]*frbox{ makeBox("0123")  },
			// 10 + 14 + 40 = 64. less than 67.
			4,
			true,
		},
		{
			"newline box",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 1,
				Rect: image.Rect(10, 15, 10 + 57, 15 + 57),
				box: []*frbox{newlinebox},
			},
			func(f *Frame) (int, bool) {
				a, b := f.canfit(image.Pt(10 + 57, 15), f.box[0]) 
				return a, b
			},
			1, 1,
			[]*frbox{newlinebox},
			// newline fits up to the edge.
			1,
			true,
		},
		{
			"tab box",
			&Frame{
				Font: Fakemetrics(fixedwidth),
				nbox:   1,
				nalloc: 1,
				Rect: image.Rect(10, 15, 10 + 57, 15 + 57),
				box: []*frbox{tabbox},
			},
			func(f *Frame) (int, bool) {
				a, b := f.canfit(image.Pt(10 + 48, 15), f.box[0]) 
				return a, b
			},
			1, 1,
			[]*frbox{tabbox},
			// tab at edge doesn't  fit
			0,
			false,
		},
	})
}
