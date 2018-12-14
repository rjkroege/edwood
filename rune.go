package main

func (r Buffer) Index(set []rune) int {
	return runeIndex([]rune(r), set)
}

func runeIndex(r []rune, set []rune) int {
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

func isIn(r []rune, s rune) bool {
	for _, c := range r {
		if s == c {
			return true
		}
	}
	return false
}

func trimLeft(r []rune, skip []rune) []rune {
	for i, c := range r {
		if !isIn(skip, c) {
			return r[i:]
		}
	}
	return r[0:0]
}

func skipbl(r []rune) []rune {
	return trimLeft(r, []rune(" \t\n"))
}
