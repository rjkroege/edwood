package main

import (
	"fmt"
	"log"
	"net"

	"9fans.net/go/plan9/client"
	"github.com/fhs/mux9p"
)

var fsysAddr net.Addr

func newPipe() (net.Conn, net.Conn, error) {
	c1, c2 := net.Pipe()
	return c1, c2, nil
}

func post9pservice(conn net.Conn, name string, mtpt string) int {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Panicf("Listen failed: %v\n", err)
	}
	go func() {
		defer l.Close()
		if err := mux9p.Do(l, conn, nil); err != nil {
			log.Panicf("9P multiplexer failed: %v\n", err)
		}
	}()
	fsysAddr = l.Addr()
	fmt.Printf("9P fileserver listening on address %v\n", fsysAddr)

	if mtpt != "" {
		log.Panicf("mounting in Windows is not supported\n")
	}
	return 0
}

// Called only in exec.c:/^run(), from a different FD group
func fsysmount(dir string, incl []string) (*MntDir, *client.Fsys, error) {
	md := fsysaddid(dir, incl)
	if md == nil {
		return nil, nil, fmt.Errorf("child: can't allocate mntdir")
	}
	conn, err := client.Dial(fsysAddr.Network(), fsysAddr.String())
	if err != nil {
		fsysdelid(md)
		return nil, nil, fmt.Errorf("child: can't connect to acme: %v", err)
	}
	fs, err := conn.Attach(nil, getuser(), fmt.Sprintf("%d", md.id))
	if err != nil {
		fsysdelid(md)
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
