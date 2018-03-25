package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
)

func startAcme(t *testing.T) (*exec.Cmd, *client.Fsys) {
	// Fork off an acme and talk with it.

	os.Setenv("NAMESPACE", os.TempDir()+"/ns.fsystest")
	os.Mkdir(os.TempDir()+"/ns.fsystest", os.ModeDir|os.ModePerm)
	os.Remove(os.TempDir() + "/ns.fsystest/acme")

	acmd := exec.Command("./edwood")
	acmd.Stdout = os.Stdout
	acmd.Start()

	var fsys *client.Fsys
	var err error
	for i := 0; i < 10; i++ {
		fsys, err = client.MountService("acme")
		if err != nil {
			if i > 9 {
				t.Fatalf("Failed to mount acme: %v", err)
				return nil, nil
			} else {
				time.Sleep(time.Second)
			}
		} else {
			break
		}
	}
	return acmd, fsys
}

// Fsys tests run my running a server and client in-process and communicating
// externally.

func TestFSys(t *testing.T) {
	var err error

	acmd, fsys := startAcme(t)

	/*	fid, err := fsys.Open("/", 0) // Readonly
		if err != nil {
			t.Errorf("Failed to open/: %v", err)
		}

		dirs, err := fid.Dirread()
		if err != nil {
			t.Errorf("Failed to open/: %v", err)
		}
		for _, d := range dirs {
			fmt.Printf("%v\n", d.String())
		}
		fid.Close()

		fid, err = fsys.Open("/1", plan9.OREAD)
		if err != nil {
			t.Errorf("Failed to walk to /1: %v", err)
		}
		dirs, err = fid.Dirread()
		if err != nil {
			t.Errorf("Failed to open/: %v", err)
		}
		for _, d := range dirs {
			fmt.Printf("%v\n", d.String())
		}
	*/
	fid, err := fsys.Open("/new/body", plan9.OWRITE)
	if err != nil {
		t.Errorf("Failed to open/: %v", err)
	}
	text := []byte("This is a test\nof the emergency typing system\n")
	fid.Write(text)
	fid.Close()

	fid, err = fsys.Open("/2/body", plan9.OREAD)
	if err != nil {
		t.Errorf("Failed to open /2/body: %v", err)
	}
	buf := make([]byte, len(text))
	_, err = fid.ReadFull(buf)
	if err != nil {
		t.Errorf("Failed to read back body: %v", err)
	}
	if string(buf) != string(text) {
		t.Errorf("Corrupted body readback: %v", buf)
	}
	fid.Close()

	fid, err = fsys.Open("/2/addr", plan9.OWRITE)
	if err != nil {
		t.Errorf("Failed to open /2/addr: %v", err)
	}
	fid.Write([]byte("#5"))
	fid.Close()

	// test insertion
	fid, err = fsys.Open("/2/data", plan9.OWRITE)
	if err != nil {
		t.Errorf("Failed to open /2/data: %v", err)
	}
	fid.Write([]byte("insertion"))
	fid.Close()

	fid, err = fsys.Open("/2/body", plan9.OREAD)
	if err != nil {
		t.Errorf("Failed to open /2/body: %v", err)
	}
	text = append(text[0:5], append([]byte("insertion"), text[5:]...)...)
	buf = make([]byte, len(text))
	_, err = fid.ReadFull(buf)
	if err != nil {
		t.Errorf("Failed to read back body: %v", err)
	}
	if string(buf) != string(text) {
		t.Errorf("Corrupted body readback: %v instead of %v", string(buf), string(text))
	}
	fid.Close()

	// Delete the window
	fid, err = fsys.Open("/2/ctl", plan9.OWRITE)
	if err != nil {
		t.Errorf("Failed to open /2/ctl: %v", err)
	}
	fid.Write([]byte("delete"))
	fid.Close()

	// Make sure it's gone from the directory
	fid, err = fsys.Open("/1", plan9.OREAD)
	if err != nil {
		t.Errorf("Failed to walk to /1: %v", err)
	}
	dirs, err := fid.Dirread()
	if err != nil {
		t.Errorf("Failed to open/: %v", err)
	}
	for _, d := range dirs {
		if d.Name == "2" {
			t.Errorf("delete didn't remove /2")
		}
	}
	fid.Close()

	acmd.Process.Kill()
	acmd.Wait()
}

func TestFSysAddr(t *testing.T) {
	acmd, fsys := startAcme(t)
	defer func() {
		acmd.Process.Kill()
		acmd.Wait()
	}()
	tfs := tFsys{t, fsys}

	//Add some known text
	text := `
This is a short block
Of text crafted
Just for this 
Occasion
`
	reportchan, exitchan := tfs.startlog()
	defer close(exitchan)

	tfs.Write("/new/body", text)

	op := <-reportchan
	if strings.Index(op, "new") == -1 {
		t.Fatalf("Didn't get report of window creation.")
	}

	id := strings.SplitN(op, " ", 2)[0]
	//	t.Errorf("New window is %v", id)

	winname := "/" + id

	// Addr is not persistent once you close it, so you need
	// to read any desired changes with the same opening.
	fid, err := fsys.Open(winname+"/addr", plan9.OREAD | plan9.OWRITE)
	if err != nil {
		t.Fatalf("Failed to open %s/addr", winname)
	}
	// TODO(flux): Should table drive this and add a pile more cases.
	fid.Write([]byte("1,2"))
	var buf [8192]byte
	n, err := fid.Read(buf[:])
	if err != nil {
		t.Fatalf("Failed to read %s/addr", winname)
	}
	var q0, q1 int
	fmt.Sscanf(string(buf[:n]), "%d %d", &q0, &q1)
	if q0 != 0 || q1 != 23 {
		t.Errorf("Expected range of 0..23 retured.  Got %d-%d.", q0, q1)
	}
	fid.Close()
}

type tFsys struct {
	t    *testing.T
	fsys *client.Fsys
}

func (tfs tFsys) startlog() (rc chan string, exit chan struct{}) {
	rc = make(chan string)
	exit = make(chan struct{})
	fid, err := tfs.fsys.Open("/log", plan9.OREAD)
	if err != nil {
		tfs.t.Errorf("Failed to open acme/log: %v", err)
	}

	go func() {
		var buf [1024]byte
		for {
			n, err := fid.Read(buf[:])
			if err != nil {
				return
			}
			rc <- string(buf[0:n])
		}
	}()
	go func() {
		<-exit
		fid.Close()
	}()
	return rc, exit
}

func (tfs tFsys) Read(file string) (s string) {
	fid, err := tfs.fsys.Open(file, plan9.OREAD)
	if err != nil {
		tfs.t.Errorf("Failed to open %s: %v", file, err)
	}
	var buf [8192]byte
	n, err := fid.Read(buf[:])
	if err != nil {
		tfs.t.Errorf("Failed to write %s: %v", file, err)
	}
	fid.Close()
	return string(buf[:n])
}

func (tfs tFsys) Write(file, s string) {
	fid, err := tfs.fsys.Open(file, plan9.OWRITE)
	if err != nil {
		tfs.t.Errorf("Failed to open %s: %v", file, err)
	}
	_, err = fid.Write([]byte(s))
	if err != nil {
		tfs.t.Errorf("Failed to write %s: %v", file, err)
	}
	fid.Close()
}
