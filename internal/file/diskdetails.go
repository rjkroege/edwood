package file

import (
	"fmt"
	"os"
)

type DiskDetails struct {
	Name string
	Info os.FileInfo
	Hash Hash // Used to check if the file has changed on disk since loaded.
}

// UpdateInfo updates File's info to d if file hash hasn't changed.
func (f *DiskDetails) UpdateInfo(filename string, d os.FileInfo) error {
	h, err := HashFor(filename)
	if err != nil {
		return fmt.Errorf("failed to compute hash for %v: %v", filename, err)
	}
	if h.Eq(f.Hash) {
		f.Info = d
	}
	return nil
}
