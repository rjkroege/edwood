// Package runes implements functions for the manipulation of rune slices.
package runes

// HasPrefix tests whether the rune slice s begins with prefix.
func HasPrefix(s, prefix []rune) bool {
	if len(prefix) > len(s) {
		return false
	}
	for i, r := range prefix {
		if s[i] != r {
			return false
		}
	}
	return true
}

// Index returns the index of the first instance of sep in s, or -1 if sep is not present in s.
func Index(s, sep []rune) int {
	n := len(sep)
	switch {
	case n > len(s):
		return -1
	case n == 0:
		return 0
	}
	for i := range s[:len(s)-n+1] {
		if HasPrefix(s[i:], sep) {
			return i
		}
	}
	return -1
}

// IndexRune returns the index of the first occurrence in s of the given rune r.
// It returns -1 if rune is not present in s.
func IndexRune(s []rune, r rune) int {
	for i, c := range s {
		if c == r {
			return i
		}
	}
	return -1
}

// ContainsRune reports whether the rune is contained in the runes slice s.
func ContainsRune(s []rune, r rune) bool {
	return IndexRune(s, r) >= 0
}

// Equal returns a boolean reporting whether a and b
// are the same length and contain the same runes.
func Equal(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i, r := range a {
		if r != b[i] {
			return false
		}
	}
	return true
}

// TrimLeft returns a subslice of s by slicing off all leading
// UTF-8-encoded code points contained in cutset.
func TrimLeft(s []rune, cutset string) []rune {
	switch {
	case len(s) == 0:
		return nil
	case len(cutset) == 0:
		return s
	}
	inCutset := func(r rune) bool {
		for _, c := range cutset {
			if c == r {
				return true
			}
		}
		return false
	}
	for i, r := range s {
		if !inCutset(r) {
			return s[i:]
		}
	}
	return nil
}
