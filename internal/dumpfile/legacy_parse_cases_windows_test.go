package dumpfile

import (
	"path/filepath"
)

func init() {
	// Remember to keep this updated between the Windows and !windows versions.

	missingpth = filepath.Join("testdata", "legacy", "nothere")
	tests = []testvector{
		{
			filename:   missingpth,
			tc:         nil,
			parseerror: "loading old dumpfile file " + missingpth + " failed: open " + missingpth + ": The system cannot find the file specified.",
		},
	}
}
