package dumpfile

import (
	"bytes"
	"os"
	"reflect"
	"testing"
)

var testTab = []Content{
	{
		CurrentDir: "/home/gopher",
		VarFont:    "/lib/fonts/go-font/regular.font",
		FixedFont:  "/lib/fonts/go-font/mono.font",
		RowTag: Text{
			Buffer: "Newcol Kill Putall Dump Exit",
		},
		Columns: []Column{
			{
				Position: 0,
				Tag: Text{
					Buffer: "New Cut Paste Snarf Sort Zerox Delcol",
				},
			},
			{
				Position: 50,
				Tag: Text{
					Buffer: "New Cut Paste Snarf Sort Zerox Delcol",
				},
			},
		},
		Windows: []*Window{},
	},
	{
		CurrentDir: "/home/gopher",
		RowTag:     Text{Buffer: "Newcol Kill Putall Dump Exit"},
		Columns:    []Column{{Position: 0}},
		Windows:    []*Window{},
		Palette: &PaletteSpec{
			Tag: FramePaletteSpec{
				Back:  ColorSpec{Color: 0xeee8d5FF},
				High:  ColorSpec{Color: 0x93a1a1FF},
				Bord:  ColorSpec{Color: 0x6c71c4FF},
				Text:  ColorSpec{Color: 0x657b83FF},
				HText: ColorSpec{Color: 0x586e75FF},
				Tick:  ColorSpec{Color: 0x586e75FF},
			},
			Text: FramePaletteSpec{
				Back:  ColorSpec{Color: 0xfdf6e3FF},
				High:  ColorSpec{Color: 0xb58900FF},
				Bord:  ColorSpec{Color: 0x859900FF},
				Text:  ColorSpec{Color: 0x657b83FF},
				HText: ColorSpec{Color: 0x586e75FF},
				Tick:  ColorSpec{Color: 0x586e75FF},
			},
			Ui: UiPaletteSpec{
				ModButton: ColorSpec{Color: 0x268bd2FF},
				ColButton: ColorSpec{Color: 0x6c71c4FF},
				But2:      ColorSpec{Color: 0xdc322fFF},
				But3:      ColorSpec{Color: 0x859900FF},
			},
		},
	},
}

func TestEncodeDecode(t *testing.T) {
	for _, tc := range testTab {
		var b bytes.Buffer

		err := tc.encode(&b)
		if err != nil {
			t.Errorf("Marshal failed: %v\n", err)
			continue
		}

		dump := b.Bytes()

		c, err := decode(bytes.NewReader(dump))
		if err != nil {
			t.Errorf("Unmarshal failed: %v\n", err)
			continue
		}
		if !reflect.DeepEqual(tc, *c) {
			t.Logf("Dump file:\n%s\n", dump)
			t.Errorf("content is %#v; expected %#v\n", c, tc)
		}
	}
}

func TestSaveLoad(t *testing.T) {
	var tc Content

	file, err := os.CreateTemp("", "edwood-dumpfile-")
	if err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}
	defer os.Remove(file.Name())

	err = tc.Save(file.Name())
	if err != nil {
		t.Fatalf("Save failed: %v\n", err)
	}

	c, err := Load(file.Name())
	if err != nil {
		t.Fatalf("Unmarshal failed: %v\n", err)
	}
	if reflect.DeepEqual(tc, c) {
		t.Errorf("content is %#v; expected %#v\n", c, tc)
	}
}
