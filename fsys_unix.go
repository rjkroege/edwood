// +build darwin dragonfly freebsd linux netbsd openbsd solaris
// +build mux9p

package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"9fans.net/go/plan9/client"
	"github.com/fhs/mux9p"
	"github.com/rjkroege/edwood/util"
)

func newPipe() (net.Conn, net.Conn, error) {
	c1, c2 := net.Pipe()
	return c1, c2, nil
}

func post9pservice(conn net.Conn, name string, mtpt string) error {
	if name == "" {
		conn.Close()
		return fmt.Errorf("nothing to do")
	}
	ns := client.Namespace()
	if err := os.MkdirAll(ns, 0700); err != nil {
		return err
	}
	addr := filepath.Join(ns, name)
	go func() {
		err := mux9p.Listen("unix", addr, conn, nil)
		if err != nil {
			util.AcmeError("9P multiplexer failed", err)
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
	conn, err := client.DialService("acme")
	if err != nil {
		mnt.DecRef(md)
		return nil, nil, fmt.Errorf("child: can't connect to acme: %v", err)
	}
	fs, err := conn.Attach(nil, getuser(), fmt.Sprintf("%d", md.id))
	if err != nil {
		mnt.DecRef(md)
		return nil, nil, fmt.Errorf("child: can't attach to acme: %v", err)
	}
	return md, fs, nil
}

// Fsopenfd opens a plan9 Fid.
func fsopenfd(fsys *client.Fsys, path string, mode uint8) *client.Fid {
	fid, err := fsys.Open(path, mode)
	if err != nil {
		warning(nil, "Failed to open %v: %v", path, err)
		return nil
	}
	return fid
}
