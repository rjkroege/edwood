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

func post9pservice(fd *os.File, name string, mtpt string) int {
	var (
		err      error
		ns, addr string
		conn     *net.UnixConn
	)

	if name == "" && mtpt == "" {
		fd.Close()
		panic("nothing to do")
		return -1
	}

	if name != "" {
		if strings.Index(name, "!") != -1 {
			addr = name
		} else {
			ns = getns()
			if ns == "" {
				return -1
			}
			addr = fmt.Sprintf("unix!%s/%s", ns, name)
		}
		if addr == "" {
			return -1
		}
		cmd := exec.Command("9pserve", "-lv", addr)
		cmd.Stdin = fd
		cmd.Stdout = fd
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			panic("Failed to start 9pserve")
		}
		fd.Close()
		// 9pserve will fork into the background.  Wait for that.
		err = cmd.Wait()
		if err != nil {
			panic("Failed to start 9pserve")
		}
		if mtpt != "" {
			// reopen
			s := strings.Split(addr, "!")
			unixaddr, err := net.ResolveUnixAddr(s[0], s[1])
			if err != nil {
				panic(fmt.Sprintf("ResolveUnixAddr: %v", err))
			}
			conn, err = net.DialUnix(s[0], unixaddr, nil)
			if err != nil {
				panic(fmt.Sprintf("cannot reopen for mount: %v", err))
			}
		}
	}
	if mtpt != "" {
		cmd := exec.Command("mount9p", "-", mtpt)
		if conn == nil {
			cmd.Stdout = fd
		} else {
			fd, err = conn.File()
			if err != nil {
				panic("Bad mtpt connection")
			}
		}
		err := cmd.Start()
		if err != nil {
			if chattyfuse {
				cmd = exec.Command("9pfuse", "-D", "-", mtpt)
			} else {
				cmd = exec.Command("9pfuse", "-", mtpt)
			}
			cmd.Stdout = fd
			err = cmd.Start()
			if err != nil {
				panic(fmt.Sprintf("failed to run 9pfuse: %v", err))
			}
		}
		fd.Close()
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
