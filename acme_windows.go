// +build windows

package main

import (
	"os"
	"syscall"
)

const defaultVarFont = "/lib/font/bit/lucsans/euro.8.font"

var ignoreSignals = []os.Signal{
	syscall.SIGPIPE,
}

var hangupSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGHUP,
}
