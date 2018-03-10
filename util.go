package main

import (
	"fmt"
	"image"
	"unicode"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func minu(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func abs(x int) int { 
	if x < 0 { return -x }
	return x
}

func region(a, b uint) int {
	if a < b {
		return -1
	}
	if a == b {
		return 0
	}
	return 1
}

func warning(md *MntDir, s string, args ...interface{}) {
	// TODO(flux): Port to actually output to the error window
	_ = md
	fmt.Printf(s, args...)
}

var (
	prevmouse image.Point
	mousew    *Window
)

func clearmouse() {
	mousew = nil
}

func savemouse(w *Window) {
	prevmouse = mouse.Point
	mousew = w
}

func restoremouse(w *Window) bool {
	defer func() { mousew = nil }()
	if mousew != nil && mousew == w {
		display.MoveTo(prevmouse)
		return true
	}
	return false
}

func isalnum(c rune) bool {
	return unicode.IsNumber(c) || unicode.IsLetter(c)
}

func runestrchr(s []rune, r rune) int {
	for ret, sr := range s {
		if sr == r {
			return ret
		}
	}
	return -1
}