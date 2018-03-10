package main

func (r Buffer) Index(set []rune) int {
	for i := 0; i < len(r); i++ {
		for _, s := range set {
			if r[i] == s {
				return i
			}
		}
	}
	return -1
}

func (r Buffer) Eq(s Buffer) bool {
	return runeEq(r, s)
}

func runeEq(r, s []rune) bool {
	if len(s) != len(r) {
		return false
	}
	for i, rr := range r {
		if rr != s[i] {
			return false
		}
	}
	return true
}

func runesplitN(buf []rune, sep []rune, nl int) [][]rune {
	linestart := 0
	lines := [][]rune{}
	for i, r := range buf {
		for _, se := range sep {
			if r == se {
				line := append(buf[linestart:i], rune('\n'))
				lines = append(lines, line)
				linestart = i + 1
			}
			if len(lines) >= nl {
				break
			}
		}
	}
	if linestart != len(buf) {
		lines = append(lines, buf[linestart:]) // trailing chunk
	}
	return lines
}
