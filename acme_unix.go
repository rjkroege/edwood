//go:build (darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris) && duitdraw
// +build darwin dragonfly freebsd linux netbsd openbsd solaris
// +build duitdraw

package main

import (
	"os"
	"syscall"
)

const (
	defaultVarFont   = ""
	defaultFixedFont = ""
	defaultMtpt      = ""
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
