// +build windows

package main

import (
	"os"
	"syscall"
)

var ignoreSignals = []os.Signal{
	syscall.SIGPIPE,
}

var hangupSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGHUP,
}
