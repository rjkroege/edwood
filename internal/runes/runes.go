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
