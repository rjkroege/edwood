// +build !windows

package dumpfile

import (
	"path/filepath"
)

func init() {
	missingpth = filepath.Join("testdata", "legacy", "nothere")
	tests = []testvector{
		{
			filename:   missingpth,
			tc:         nil,
			parseerror: "loading old dumpfile file " + missingpth + " failed: open " + missingpth + ": no such file or directory",
		},
	}

}
