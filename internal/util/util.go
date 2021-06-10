package util

import "log"

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

func Acmeerror(s string, err error) {
	log.Panicf("acme: %s: %v\n", s, err)
}
