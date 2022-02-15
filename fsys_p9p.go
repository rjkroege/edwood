//go:build (darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris) && !mux9p
// +build darwin dragonfly freebsd linux netbsd openbsd solaris
// +build !mux9p

package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	"9fans.net/go/plan9/client"
)

var chattyfuse bool

func newPipe() (net.Conn, net.Conn, error) {
	c1, c2 := net.Pipe()
	return c1, c2, nil
}

func post9pservice(conn net.Conn, name string, mtpt string) error {
	if name == "" && mtpt == "" {
		conn.Close()
		return fmt.Errorf("nothing to do")
	}

	if name != "" {
		addr := name
		if !strings.Contains(name, "!") {
			ns := client.Namespace()
			if err := os.MkdirAll(ns, 0700); err != nil {
				return err
			}
			addr = fmt.Sprintf("unix!%s/%s", ns, name)
		}
		// TODO(fhs): create /some/dir if name is unix!/some/dir/acme

		if addr == "" {
			return fmt.Errorf("empty listen address")
		}
		cmd := exec.Command("9pserve", "-lv", addr)
		cmd.Stdin = conn
		cmd.Stdout = conn
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			return fmt.Errorf("failed to start 9pserve: %v", err)
		}
		// 9pserve will fork into the background.  Wait for that.
		if state, err := cmd.Process.Wait(); err != nil || !state.Success() {
			return fmt.Errorf("9pserve wait failed: %v, %v", err, state)
		}
		go func() {
			// Now wait for I/O to finish.
			err = cmd.Wait()
			if err != nil {
				// Most likely Edwood is preparing to exit and closed its end of the pipe.
				// Don't panic -- give Edwood a chance to clean up before exit.
				log.Printf("9pserve wait failed: %v", err)
			}
			conn.Close()
		}()
		if mtpt != "" {
			// reopen
			s := strings.Split(addr, "!")
			unixaddr, err := net.ResolveUnixAddr(s[0], s[1])
			if err != nil {
				return fmt.Errorf("ResolveUnixAddr: %v", err)
			}
			conn, err = net.DialUnix(s[0], nil, unixaddr)
			if err != nil {
				return fmt.Errorf("cannot reopen for mount: %v", err)
			}
		}
	}
	if mtpt != "" {
		// 9pfuse uses fd 0 for both reads and writes, so we need to set cmd.Stdin to
		// a *os.File instead of an io.Reader. os/exec package will set stdin to the
		// fd in *os.File, which supports both read and writes. It doesn't matter what
		// we set cmd.Stdout to because it's never used!
		uconn, ok := conn.(*net.UnixConn)
		if !ok {
			// Thankfully, we should never reach here beacause name is always "acme".
			return fmt.Errorf("9pfuse writes to stdin")
		}
		fd, err := uconn.File()
		if err != nil {
			return fmt.Errorf("bad mtpt connection: %v", err)
		}

		// Try v9fs on Linux, which will mount 9P directly.
		cmd := exec.Command("mount9p", "-", mtpt)
		cmd.Stdin = fd
		cmd.Stdout = fd
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			if chattyfuse {
				cmd = exec.Command("9pfuse", "-D", "-", mtpt)
			} else {
				cmd = exec.Command("9pfuse", "-", mtpt)
			}
			cmd.Stdin = fd
			cmd.Stdout = fd
			cmd.Stderr = os.Stderr
			err = cmd.Start()
			if err != nil {
				return fmt.Errorf("failed to run 9pfuse: %v", err)
			}
		}
		go func() {
			err = cmd.Wait()
			if err != nil {
				log.Printf("mount9p/9pfuse wait failed: %v", err)
			}
			fd.Close()
		}()
	}
	return nil
}

// Called only in exec.c:/^run(), from a different FD group
// TODO(rjk): Do we ever run this code?
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
