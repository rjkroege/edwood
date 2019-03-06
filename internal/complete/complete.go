// Package complete implements file name completion.
//
// This is a port of Plan 9's libcomplete to Go.
package complete

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"unicode/utf8"
)

// Completion represents the result of file completion.
type Completion struct {
	// Advance reports whether the file name prefix can be extended
	// without changing the set of files that match.
	Advance bool

	// Complete reports whether the extended file name uniquely
	// identifies a file (i.e. NMatch == 1).
	Complete bool

	// String holds the extension of the file name prefix.
	//
	// If Advance is false, String is an empty string. Otherwise,
	// String will be set to the extension; that is, the value of
	// String may be appended to the file name prefix by the caller
	// to extend the embryonic file name unambiguously.
	//
	// If Complete is true, String will be suffixed with a blank,
	// or a path separator, depending on whether the resulting file
	// name identifies a plain file or a directory.
	String string

	// NMatch specifies the number of files that matched.
	NMatch int

	// Filename holds the matching filenames. If there is no match
	// (NMatch == 0), it holds the full set of files in the directory.
	// If the file named is a directory, a slash character will be
	// appended to it.
	Filename []string
}

func longestPrefixLength(a, b string, n int) int {
	i := 0
	for i < n {
		ra, w := utf8.DecodeRuneInString(a[i:])
		rb, _ := utf8.DecodeRuneInString(b[i:])
		if ra != rb {
			break
		}
		i += w
	}
	return i
}

// Complete implements file name completion. Given a directory dir and a
// file name prefix s, it returns an analysis of the file names in that
// directory that begin with the string s.
func Complete(dir, s string) (*Completion, error) {
	if strings.ContainsRune(s, os.PathSeparator) {
		return nil, errors.New("path separator in name argument to complete()")
	}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	// find the matches
	var name []string
	var mode []os.FileMode
	minlen := 1000000
	for _, file := range files {
		if strings.HasPrefix(file.Name(), s) {
			name = append(name, file.Name())
			mode = append(mode, file.Mode())
			if minlen > len(file.Name()) {
				minlen = len(file.Name())
			}
		}
	}

	var c Completion
	if len(name) > 0 {
		// report interesting results
		// trim length back to longest common initial string
		for i := 1; i < len(name); i++ {
			minlen = longestPrefixLength(name[0], name[i], minlen)
		}

		// build the answer
		c.Complete = len(name) == 1
		c.Advance = c.Complete || minlen > len(s)
		c.String = name[0][len(s):minlen]
		if c.Complete {
			if mode[0].IsDir() {
				c.String += string(os.PathSeparator)
			} else {
				c.String += " "
			}
		}
		c.NMatch = len(name)
	} else {
		// no match, so return all possible strings
		for _, file := range files {
			name = append(name, file.Name())
			mode = append(mode, file.Mode())
		}
		c.NMatch = 0
	}

	// attach list of names
	for i := range name {
		if mode[i].IsDir() {
			name[i] += string(os.PathSeparator)
		}
	}
	c.Filename = name
	return &c, nil
}
