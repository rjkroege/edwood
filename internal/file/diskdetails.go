package file

import (
	"fmt"
	"os"
)

type DiskDetails struct {
	// TODO(sn0w): Fix places where these are reached into from outside of file
	Name string
	Info os.FileInfo
	Hash Hash // Used to check if the file has changed on disk since loaded.
}

// UpdateInfo updates File's info to d if file hash hasn't changed.
func (f *DiskDetails) UpdateInfo(filename string, d os.FileInfo) error {
	h, err := HashFor(filename)
	if err != nil {
		return fmt.Errorf("failed to compute hash for %v: %v", filename, err)
		// TODO(sn0w): Ask Rob what he wants to do with errors outside the pkg
		//return warnError(nil, "failed to compute hash for %v: %v", filename, err)
	}
	if h.Eq(f.Hash) {
		f.Info = d
	}
	return nil
}
