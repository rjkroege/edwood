package file

import (
	"fmt"
	"hash/fnv"
	"io"
	"os"
)

type DiskDetails struct {
	Name  string
	Info  os.FileInfo
	Hash  uint64 // Used to check if the file has changed on disk since loaded.
	isdir bool   // Used to track if this File is populated from a directory list. [private]
}

// IsDir returns true if the File has a synthetic backing of
// a directory.
// TODO(rjk): File is a facade that subsumes the entire Model
// of an Edwood MVC. As such, it should look like a text buffer for
// view/controller code. isdir is true for a specific kind of File innards
// where we automatically alter the contents in various ways.
// Automatically altering the contents should be expressed differently.
// Directory listings should not be special cased throughout.
func (f *DiskDetails) IsDir() bool {
	return f.isdir
}

func (f *DiskDetails) SetDir(isdir bool) {
	f.isdir = isdir
}

// UpdateInfo updates File's info to d if file hash hasn't changed.
func (f *DiskDetails) UpdateInfo(filename string, d os.FileInfo) error {

	filehashcode, err := HashFor(filename)
	if err != nil {
		return fmt.Errorf("failed to compute hash for %v: %v", filename, err)
	}
	if f.Hash == filehashcode {
		f.Info = d
	}
	return nil
}

func HashFor(filename string) (uint64, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer fd.Close()

	hh := fnv.New64a()
	if _, err := io.Copy(hh, fd); err != nil {
		return 0, err
	}
	return hh.Sum64(), nil
}
