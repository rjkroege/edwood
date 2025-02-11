//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package main

import (
	"os"

	"github.com/pkg/term/termios"
	"golang.org/x/sys/unix"
)

func isecho(fp *os.File) bool {
	var ttmode unix.Termios
	err := termios.Tcgetattr(fp.Fd(), &ttmode)
	if err != nil {
		debugf("tcgetattr: %v\n", err)
	}
	return ttmode.Lflag&unix.ECHO != 0
}
