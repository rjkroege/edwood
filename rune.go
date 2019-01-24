package main

import "github.com/rjkroege/edwood/internal/runes"

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
	return runes.Equal(r, s)
}

func skipbl(r []rune) []rune {
	return runes.TrimLeft(r, " \t\n")
}
