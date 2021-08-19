package main

import (
	"flag"
	"fmt"
	"net"

	"9fans.net/go/plan9/client"
	"github.com/fhs/mux9p"
	"github.com/rjkroege/edwood/util"
)

var (
	fsysAddrFlag = flag.String("fsys.addr", "localhost:0", "9P file system listen address")
	fsysAddr     net.Addr
)

func newPipe() (net.Conn, net.Conn, error) {
	c1, c2 := net.Pipe()
	return c1, c2, nil
}

func post9pservice(conn net.Conn, name string, mtpt string) error {
	l, err := net.Listen("tcp", *fsysAddrFlag)
	if err != nil {
		return fmt.Errorf("listen failed: %v", err)
	}
	go func() {
		defer l.Close()
		if err := mux9p.Do(l, conn, nil); err != nil {
			util.AcmeError("9P multiplexer failed", err)
		}
	}()
	fsysAddr = l.Addr()
	fmt.Printf("9P fileserver listening on address %v\n", fsysAddr)

	if mtpt != "" {
		return fmt.Errorf("mounting in Windows is not supported")
	}
	return nil
}

// Called only in exec.c:/^run(), from a different FD group
func fsysmount(dir string, incl []string) (*MntDir, *client.Fsys, error) {
	md := mnt.Add(dir, incl) // DecRef in waitthread
	if md == nil {
		return nil, nil, fmt.Errorf("child: can't allocate mntdir")
	}
	if fsysAddr == nil {
		return nil, nil, fmt.Errorf("child: unknown address")
	}
	conn, err := client.Dial(fsysAddr.Network(), fsysAddr.String())
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
