// +build plan9

package main

import (
	"os"
	"syscall"
)

var ignoreSignals = []os.Signal{
	syscall.Note("sys: write on closed pipe"),
}

var hangupSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGHUP,
}
