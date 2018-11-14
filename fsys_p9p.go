// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package main

import (
	"fmt"
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

func post9pservice(conn net.Conn, name string, mtpt string) int {
	if name == "" && mtpt == "" {
		conn.Close()
		panic("nothing to do")
	}

	if name != "" {
		addr := name
		if !strings.Contains(name, "!") {
			addr = fmt.Sprintf("unix!%s/%s", client.Namespace(), name)
		}
		if addr == "" {
			return -1
		}
		cmd := exec.Command("9pserve", "-lv", addr)
		cmd.Stdin = conn
		cmd.Stdout = conn
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			panic(fmt.Sprintf("failed to start 9pserve: %v", err))
		}
		// 9pserve will fork into the background.  Wait for that.
		if state, err := cmd.Process.Wait(); err != nil || !state.Success() {
			panic(fmt.Sprintf("9pserve wait failed: %v, %v", err, state))
		}
		go func() {
			// Now wait for I/O to finish.
			err = cmd.Wait()
			if err != nil {
				panic(fmt.Sprintf("9pserve wait failed: %v", err))
			}
			conn.Close()
		}()
		if mtpt != "" {
			// reopen
			s := strings.Split(addr, "!")
			unixaddr, err := net.ResolveUnixAddr(s[0], s[1])
			if err != nil {
				panic(fmt.Sprintf("ResolveUnixAddr: %v", err))
			}
			conn, err = net.DialUnix(s[0], nil, unixaddr)
			if err != nil {
				panic(fmt.Sprintf("cannot reopen for mount: %v", err))
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
			panic("9pfuse writes to stdin!")
		}
		fd, err := uconn.File()
		if err != nil {
			panic(fmt.Sprintf("bad mtpt connection: %v", err))
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
				panic(fmt.Sprintf("failed to run 9pfuse: %v", err))
			}
		}
		go func() {
			err = cmd.Wait()
			if err != nil {
				panic(fmt.Sprintf("wait failed: %v", err))
			}
			fd.Close()
		}()
	}
	return 0
}

// Called only in exec.c:/^run(), from a different FD group
func fsysmount(dir string, incl []string) (*MntDir, *client.Fsys, error) {
	md := fsysaddid(dir, incl)
	if md == nil {
		return nil, nil, fmt.Errorf("child: can't allocate mntdir")
	}
	conn, err := client.DialService("acme")
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
