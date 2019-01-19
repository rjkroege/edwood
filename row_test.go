package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const shortFile = `/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood
/mnt/font/GoRegular/13a/font
`

const fullFile = `/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood
/mnt/font/GoRegular/13a/font
/mnt/font/Iosevka/12a/font
`

func TestLoadFonts(t *testing.T) {
	dir, err := ioutil.TempDir("", "testloadfonts")
	if err != nil {
		t.Fatal("TestLoadFonts can't make directory:", err)
	}
	defer os.RemoveAll(dir)

	if resp := LoadFonts(filepath.Join(dir, "not_there")); !reflect.DeepEqual(resp, []string{}) {
		t.Errorf("TestLoadFonts not_there want %v, got %v", []string{}, resp)
	}

	f, err := os.Create(filepath.Join(dir, "invalid_file"))
	if err != nil {
		t.Fatal("TestLoadFonts invalid_file can't create file:", err)
	}
	f.Close()

	if resp := LoadFonts(filepath.Join(dir, "invalid_file")); !reflect.DeepEqual(resp, []string{}) {
		t.Errorf("TestLoadFonts invalid_file want %v, got %v", []string{}, resp)
	}

	f, err = os.Create(filepath.Join(dir, "short_file"))
	if err != nil {
		t.Fatal("TestLoadFonts short_file can't create file:", err)
	}
	if _, err := f.WriteString(shortFile); err != nil {
		t.Fatal("TestLoadFonts short_file can't write file:", err)
	}
	f.Close()

	if resp := LoadFonts(filepath.Join(dir, "short_file")); !reflect.DeepEqual(resp, []string{}) {
		t.Errorf("TestLoadFonts short_file want %v, got %#v", []string{}, resp)
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
