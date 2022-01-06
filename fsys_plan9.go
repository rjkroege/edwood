package main

import (
	"fmt"
	"net"
	"os"
	"path"
	"syscall"

	"9fans.net/go/plan9/client"
	"github.com/rjkroege/edwood/util"
)

// These constants are from /sys/include/libc.h
const (
	MREPL   = 0x0000 // mount replaces object
	MBEFORE = 0x0001 // mount goes before others in union directory
)

// newPipe is a replacement for net.Pipe that is backed by file descriptors.
// It's useful when a file descriptor is needed to pass to mount(2) syscall.
func newPipe() (net.Conn, net.Conn, error) {
	var p [2]int
	err := syscall.Pipe(p[:])
	if err != nil {
		return nil, nil, err
	}
	return &pipeConn{File: os.NewFile(uintptr(p[0]), "|0")},
		&pipeConn{File: os.NewFile(uintptr(p[1]), "|1")},
		nil
}

// pipeConn represents a bidirectional pipe that implements net.Conn.
type pipeConn struct {
	*os.File
}

func (*pipeConn) LocalAddr() net.Addr  { return pipeAddr{} }
func (*pipeConn) RemoteAddr() net.Addr { return pipeAddr{} }

type pipeAddr struct{}

func (pipeAddr) Network() string { return "pipe" }
func (pipeAddr) String() string  { return "pipe" }

func post9pservice(conn net.Conn, name string, mtpt string) error {
	cfd := int(conn.(*pipeConn).File.Fd())
	go func() {
		// We need to do this within a new goroutine because the file server
		// hasn't been started yet.
		err := syscall.Mount(cfd, -1, mtpt, MREPL, "")
		if err != nil {
			util.AcmeError("mount failed", err)
		}
		err = syscall.Bind(mtpt, "/mnt/wsys", MREPL)
		if err != nil {
			util.AcmeError("bind /mnt/wsys filed", err)
		}
		err = syscall.Bind(mtpt, "/dev", MBEFORE)
		if err != nil {
			util.AcmeError("bind /dev filed", err)
		}
	}()
	return nil
}

// Called only in exec.c:/^run(), from a different FD group
func fsysmount(dir string, incl []string) (*MntDir, *client.Fsys, error) {
	md := mnt.Add(dir, incl) // DecRef in waitthread
	if md == nil {
		return nil, nil, fmt.Errorf("child: can't allocate mntdir")
	}
	// TODO(fhs): This function should really run within the file namespace
	// of the command we're going to run. Then, we can mount /mnt/acme here
	// with attach name set to md.id. Currently we just mount once in post9pservice().
	return md, nil, nil
}

// Fsopenfd opens a plan9 Fid.
func fsopenfd(fsys *client.Fsys, filename string, mode uint8) *os.File {
	f, err := os.OpenFile(path.Join(*mtpt, filename), int(mode), 0)
	if err != nil {
		warning(nil, "Failed to open %v: %v", filename, err)
		return nil
	}
	return f
}
