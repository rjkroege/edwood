package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

var chattyfuse bool

func post9pservice(conn net.Conn, name string, mtpt string) int {
	if name == "" && mtpt == "" {
		conn.Close()
		panic("nothing to do")
	}

	if name != "" {
		addr := name
		if !strings.Contains(name, "!") {
			ns := getns()
			if ns == "" {
				return -1
			}
			addr = fmt.Sprintf("unix!%s/%s", ns, name)
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

func getns() string {
	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		var err error
		ns, err = nsfromdisplay()
		if err != nil {
			acmeerror(ns, err)
		}
	}
	if ns == "" {
		panic("$NAMESPACE not set")
	}
	return ns
}

func getuser() string {
	user, err := user.Current()
	if err != nil {
		return "Wile E. Coyote"
	}
	return user.Username
}

func nsfromdisplay() (ns string, err error) {
	disp := os.Getenv("DISPLAY")
	if disp == "" {
		disp = ":0.0"
	}

	disp = strings.TrimSuffix(disp, ".0")
	disp = strings.Replace(disp, "/", "_", -1)
	ns = fmt.Sprintf("/tmp/ns.%s.%s", getuser(), disp)
	err = os.Mkdir(ns, 0700)
	if err == nil {
		return ns, nil
	}

	// See if it's already there
	f, err := os.Open(ns)
	if err != nil {
		return "", fmt.Errorf("Can't open namespace %s: %v", ns, err)
	}
	defer f.Close()
	s, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("Can't stat namespace %s: %v", ns, err)
	}
	if !s.IsDir() || s.Mode()&0777 != 0700 { // || !isme(d->uid)
		return "", fmt.Errorf("Bad namespace %s: %v", ns, err)
	}
	return ns, nil
}
