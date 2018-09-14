// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package main

import (
	"os"
	"syscall"
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
