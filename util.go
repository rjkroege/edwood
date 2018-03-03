package main

import (
	"fmt"
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

func region(a, b int) int {
	if a < b {
		return -1
	}
	if a == b {
		return 0
	}
	return 1
}

type Mntdir string // TODO(flux): This will get implemented and conflict at some point :-)
func warning(md *Mntdir, s string, args ...interface{}) {
	// TODO(flux): Port to actually output to the error window
	_ = md
	fmt.Printf(s, args...)
}
