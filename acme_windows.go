//go:build windows
// +build windows

package main

import (
	"os"
	"syscall"
)

const (
	defaultVarFont   = `C:\Windows\Fonts\arial.ttf@12pt`
	defaultFixedFont = `C:\Windows\Fonts\lucon.ttf@12pt`
	defaultMtpt      = ""
)

var ignoreSignals = []os.Signal{
	syscall.SIGPIPE,
}

var hangupSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGHUP,
}
