// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package main

import (
	"os"
	"syscall"
)

const (
	// lucidasans font is called lucsans in plan9port.
	// See https://marc.info/?l=9fans&m=114412454010468&w=2
	defaultVarFont = "/lib/font/bit/lucsans/euro.8.font"

	defaultMtpt = ""
)

var ignoreSignals = []os.Signal{
	syscall.SIGPIPE,
	syscall.SIGTTIN,
	syscall.SIGTTOU,
	syscall.SIGTSTP,
}

var hangupSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGHUP,
}
