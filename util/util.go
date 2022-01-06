package util

import (
	"log"
	"unicode/utf8"
)

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func Minu(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func AcmeError(s string, err error) {
	log.Panicf("acme: %s: %v\n", s, err)
}

// Cvttorunes decodes runes r from p. It's guaranteed that first n
// bytes of p will be interpreted without worrying about partial runes.
// This may mean reading up to UTFMax-1 more bytes than n; the caller
// must ensure p is large enough. Partial runes and invalid encodings
// are converted to RuneError. Nb (always >= n) is the number of bytes
// interpreted.
//
// If any U+0000 rune is present in r, they are elided and nulls is set
// to true.
func Cvttorunes(p []byte, n int) (r []rune, nb int, nulls bool) {
	for nb < n {
		var w int
		var ru rune
		if p[nb] < utf8.RuneSelf {
			w = 1
			ru = rune(p[nb])
		} else {
			ru, w = utf8.DecodeRune(p[nb:])
		}
		if ru != 0 {
			r = append(r, ru)
		} else {
			nulls = true
		}
		nb += w
	}
	return
}
