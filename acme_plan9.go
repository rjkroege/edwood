//go:build plan9
// +build plan9

package main

import (
	"os"
	"syscall"
)

const (
	defaultVarFont   = "/lib/font/bit/lucidasans/euro.8.font"
	defaultFixedFont = "/lib/font/bit/lucm/unicode.9.font"
	defaultMtpt      = "/mnt/acme"
)

var ignoreSignals = []os.Signal{
	syscall.Note("sys: write on closed pipe"),
}

var hangupSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGHUP,
}
