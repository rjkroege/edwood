package dumpfile

import (
	"bytes"
	"io/ioutil"
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
		if reflect.DeepEqual(tc, c) {
			t.Logf("Dump file:\n%s\n", dump)
			t.Errorf("content is %#v; expected %#v\n", c, tc)
		}
	}
}

func TestSaveLoad(t *testing.T) {
	var tc Content

	file, err := ioutil.TempFile("", "edwood-dumpfile-")
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
