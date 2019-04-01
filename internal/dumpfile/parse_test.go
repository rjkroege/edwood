package dumpfile

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var testTab = []Content{
	{
		CurrentDir: "/home/gopher",
		VarFont:    "/lib/fonts/go-font/regular.font",
		FixedFont:  "/lib/fonts/go-font/mono.font",
		RowTag:     "Newcol Kill Putall Dump Exit",
		Columns: []Column{
			{
				Position: 0,
				Tag:      "New Cut Paste Snarf Sort Zerox Delcol",
			},
			{
				Position: 50,
				Tag:      "New Cut Paste Snarf Sort Zerox Delcol",
			},
		},
		Windows: []Window{},
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

const shortFile = `{"Version": 1,
"CurrentDir": "/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood",
"VarFont": "/mnt/font/GoRegular/13a/font"}
`

const fullFile = `{"Version": 1,
"CurrentDir": "/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood",
"VarFont": "/mnt/font/GoRegular/13a/font",
"FixedFont": "/mnt/font/Iosevka/12a/font"}
`

func TestLoadFonts(t *testing.T) {
	dir, err := ioutil.TempDir("", "testloadfonts")
	if err != nil {
		t.Fatal("TestLoadFonts can't make directory:", err)
	}
	defer os.RemoveAll(dir)

	if resp := LoadFonts(filepath.Join(dir, "not_there")); resp != nil {
		t.Errorf("TestLoadFonts not_there want nil, got %#v", resp)
	}

	f, err := os.Create(filepath.Join(dir, "invalid_file"))
	if err != nil {
		t.Fatal("TestLoadFonts invalid_file can't create file:", err)
	}
	f.Close()

	if resp := LoadFonts(filepath.Join(dir, "invalid_file")); resp != nil {
		t.Errorf("TestLoadFonts invalid_file want nil, got %#v", resp)
	}

	f, err = os.Create(filepath.Join(dir, "short_file"))
	if err != nil {
		t.Fatal("TestLoadFonts short_file can't create file:", err)
	}
	if _, err := f.WriteString(shortFile); err != nil {
		t.Fatal("TestLoadFonts short_file can't write file:", err)
	}
	f.Close()

	if resp := LoadFonts(filepath.Join(dir, "short_file")); resp != nil {
		t.Errorf("TestLoadFonts short_file want nil, got %#v", resp)
	}

	f, err = os.Create(filepath.Join(dir, "full_file"))
	if err != nil {
		t.Fatal("TestLoadFonts full_file can't create file:", err)
	}
	if _, err := f.WriteString(fullFile); err != nil {
		t.Fatal("TestLoadFonts full_file can't write file:", err)
	}
	f.Close()

	if resp := LoadFonts(filepath.Join(dir, "full_file")); !reflect.DeepEqual(resp, []string{
		"/mnt/font/GoRegular/13a/font",
		"/mnt/font/Iosevka/12a/font",
	}) {
		t.Errorf("TestLoadFonts full_file want %v, got %v", []string{
			"/mnt/font/GoRegular/13a/font",
			"/mnt/font/Iosevka/12a/font",
		}, resp)
	}
}
