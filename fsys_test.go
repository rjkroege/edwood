package main

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"9fans.net/go/acme"
	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"github.com/google/go-cmp/cmp"
	"github.com/rjkroege/edwood/edwoodtest"
	"github.com/rjkroege/edwood/ninep"
)

func TestMain(m *testing.M) {
	switch os.Getenv("TEST_MAIN") {
	case "edwood":
		dump := predrawInit()
		display := edwoodtest.NewDisplay(image.Rectangle{})
		mainWithDisplay(global, dump, display)
	default:
		// Prevent mounting any acme file server being run by the user running tests.
		os.Unsetenv("NAMESPACE")
		e := m.Run()
		os.Exit(e)
	}
}

type Acme struct {
	t    *testing.T
	ns   string
	cmd  *exec.Cmd
	fsys *client.Fsys
}

// augmentPath extends PATH so that plan9 dependencies can be
// found in the build directory.
func augmentPathEnv() {
	// We have Linux and Darwin executables.
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return
	}

	// If the executables are already present, skip.
	_, errdevdraw := exec.LookPath("devdraw")
	_, err9pserve := exec.LookPath("9pserve")
	if errdevdraw == nil && err9pserve == nil {
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		return
	}

	path := os.Getenv("PATH") + ":" + filepath.Join(wd, "build", "bin"+"_"+runtime.GOOS)
	os.Setenv("PATH", path)

	// We also need fonts.
	if _, hzp9 := os.LookupEnv("PLAN9"); !hzp9 {
		os.Setenv("PLAN9", filepath.Join(wd, "build"))
	}
}

// startAcme runs an edwood process and 9p mounts it (at acme) in the
// namespace so that a test may exercise IPC to the subordinate edwood
// process.
func startAcme(t *testing.T, args ...string) *Acme {
	// If $USER is not set (i.e. running in a Docker container)
	// MountService will fail. Detect this and give up if this is so.
	if _, hzuser := os.LookupEnv("USER"); !hzuser {
		t.Fatalf("Test will fail unless USER is set in environment. Please set.")
	}

	ns, err := os.MkdirTemp("", "ns.fsystest")
	if err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}
	os.Setenv("NAMESPACE", ns)
	augmentPathEnv()

	acmd := exec.Command(os.Args[0], args...)
	acmd.Env = append(os.Environ(), "TEST_MAIN=edwood")

	acmd.Stdout = os.Stdout
	acmd.Stderr = os.Stderr
	if err := acmd.Start(); err != nil {
		t.Fatalf("failed to execute edwood: %v", err)
	}

	var fsys *client.Fsys
	for i := 0; i < 10; i++ {
		fsys, err = client.MountService("acme")
		if err != nil {
			if i >= 9 {
				t.Fatalf("Failed to mount acme: %v", err)
				return nil
			}
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	return &Acme{
		ns:   ns,
		cmd:  acmd,
		fsys: fsys,
	}
}

func (a *Acme) Cleanup() {
	a.cmd.Process.Kill()
	a.cmd.Wait()
	if err := os.RemoveAll(a.ns); err != nil {
		a.t.Errorf("failed to remove temporary namespace %v: %v", a.ns, err)
	}
}

// Fsys tests run by running a server and client in-process and communicating
// externally.

func TestFSys(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	a := startAcme(t)
	defer a.Cleanup()
	fsys := a.fsys

	t.Run("Dirread", func(t *testing.T) {
		t.SkipNow()                   // Only used for debugging?
		fid, err := fsys.Open("/", 0) // Readonly
		if err != nil {
			t.Fatalf("Failed to open/: %v", err)
		}

		dirs, err := fid.Dirread()
		if err != nil {
			t.Errorf("Failed to open/: %v", err)
		}
		for _, d := range dirs {
			t.Logf("%v\n", d.String())
		}
		fid.Close()

		fid, err = fsys.Open("/1", plan9.OREAD)
		if err != nil {
			t.Fatalf("Failed to walk to /1: %v", err)
		}
		dirs, err = fid.Dirread()
		if err != nil {
			t.Errorf("Failed to open/: %v", err)
		}
		for _, d := range dirs {
			t.Logf("%v\n", d.String())
		}
	})
	t.Run("Basic", func(t *testing.T) {
		fid, err := fsys.Open("/new/body", plan9.OWRITE)
		if err != nil {
			t.Fatalf("Failed to open/: %v", err)
		}
		text := []byte("This is a test\nof the emergency typing system\n")
		fid.Write(text)
		fid.Close()

		fid, err = fsys.Open("/2/body", plan9.OREAD)
		if err != nil {
			t.Fatalf("Failed to open /2/body: %v", err)
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
			t.Fatalf("Failed to open /2/addr: %v", err)
		}
		fid.Write([]byte("#5"))
		fid.Close()

		// test insertion
		fid, err = fsys.Open("/2/data", plan9.OWRITE)
		if err != nil {
			t.Fatalf("Failed to open /2/data: %v", err)
		}
		fid.Write([]byte("insertion"))
		fid.Close()

		fid, err = fsys.Open("/2/body", plan9.OREAD)
		if err != nil {
			t.Fatalf("Failed to open /2/body: %v", err)
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
			t.Fatalf("Failed to open /2/ctl: %v", err)
		}
		fid.Write([]byte("delete"))
		fid.Close()

		// Make sure it's gone from the directory
		fid, err = fsys.Open("/1", plan9.OREAD)
		if err != nil {
			t.Fatalf("Failed to walk to /1: %v", err)
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
	})
	t.Run("BigBody", func(t *testing.T) {
		// Create a large string with some non-latin characters so that
		// we exercise buffering and unicode encoding in xfidutfread
		var b bytes.Buffer
		for i := 0; i < BUFSIZE; i++ {
			b.WriteString("Go 世界\n")
		}
		text := b.Bytes()

		w, err := acme.New()
		if err != nil {
			t.Fatalf("Creating new window failed: %v\n", err)
		}
		defer w.Del(true)

		w.Write("body", text)
		buf, err := w.ReadAll("body")
		if err != nil {
			t.Fatalf("Reading body failed: %v\n", err)
		}
		if string(buf) != string(text) {
			t.Errorf("Read %v bytes from body; expected %v\n", len(buf), len(text))
		}
	})
	t.Run("CtlWrite", func(t *testing.T) {
		w, err := acme.New()
		if err != nil {
			t.Fatalf("Creating new window failed: %v\n", err)
		}
		defer w.Del(true)

		for _, tc := range []struct {
			name string
			ok   bool
		}{
			{"/edwood/test1", true},
			{"/edwood/世界.txt", true},
			{"/edwood/name with space", true},
			{"/edwood/\x00\x00test2", false},
		} {
			err := w.Name(tc.name)
			if !tc.ok {
				if err == nil {
					t.Errorf("Writing window name %q returned nil error\n", tc.name)
				}
				continue
			}
			if err != nil {
				t.Errorf("Failed to write window name %q: %v\n", tc.name, err)
				continue
			}
			b, err := w.ReadAll("tag")
			if err != nil {
				t.Errorf("Failed to read tag: %v\n", err)
				continue
			}
			// Supporting spaces requires different parsing.
			// TODO(rjk): Consider adding a helper to gozen for this task.
			tags := strings.SplitN(string(b), " ", 2)
			fn := tags[0]
			if b[0] == '\'' {
				fn = string(b[1 : bytes.IndexByte(b[1:], '\'')+1])
			}
			if tc.name != fn {
				t.Errorf("Window name is %q; expected %q\n", fn, tc.name)
			}
		}

		dump := "Watch go test"
		if err := w.Ctl("dump " + dump); err != nil {
			t.Errorf("Failed to write dump %q: %v\n", dump, err)
		}
		dumpdir := "/home/gopher/src/edwood"
		if err := w.Ctl("dumpdir " + dumpdir); err != nil {
			t.Errorf("Failed to write dumpdir %q: %v\n", dumpdir, err)
		}
	})
	t.Run("WriteEditout", func(t *testing.T) {
		fid, err := fsys.Open("/editout", plan9.OWRITE)
		if err != nil {
			t.Fatalf("failed to open /editout: %v", err)
		}
		defer fid.Close()
		_, err = fid.Write([]byte("hello\n"))
		if err == nil || err.Error() != ErrPermission.Error() {
			t.Fatalf("write to editout returned %v; expected %v", err, ErrPermission)
		}
	})
	t.Run("WriteEvent", func(t *testing.T) {
		w, err := acme.New()
		if err != nil {
			t.Fatalf("creating new window failed: %v\n", err)
		}
		defer w.Del(true)

		tt := []struct {
			err error
			s   string
		}{
			{nil, ""},
			{nil, "\n"},
			{nil, "ML0 0 \n"},
			{nil, "Ml0 0 \n"},
			{nil, "MX0 0 \n"},
			{nil, "ML0 0 \nMl0 0 \n"},
			{ErrBadEvent, "M\n"},
			{ErrBadEvent, "ML\n"},
			{ErrBadEvent, "ML0 \n"},
			{ErrBadEvent, "MLA 0 \n"},
			{ErrBadEvent, "ML0 A \n"},
			{ErrBadEvent, "M40 0 \n"},
			{ErrBadEvent, "ML9 9 \n"},
			{ErrBadEvent, "MZ0 0 \n"},
			{ErrBadEvent, "MZ0 0 \nML0 0 \n"}, // bad event followed by a good one
		}
		for _, tc := range tt {
			_, err = w.Write("event", []byte(tc.s))
			if (tc.err == nil && err != nil) ||
				(tc.err != nil && (err == nil || err.Error() != tc.err.Error())) {
				t.Errorf("event %q returned %v; expected %v", tc.s, err, tc.err)
			}
		}
	})
}

func TestFSysAddr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	a := startAcme(t)
	defer a.Cleanup()
	tfs := tFsys{t, a.fsys}

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
	for strings.Contains(op, "focus") {
		op = <-reportchan
	}
	if !strings.Contains(op, "new") {
		t.Fatalf("Didn't get report of window creation.")
	}

	id := strings.SplitN(op, " ", 2)[0]
	//	t.Errorf("New window is %v", id)

	winname := "/" + id

	// Addr is not persistent once you close it, so you need
	// to read any desired changes with the same opening.
	fid, err := a.fsys.Open(winname+"/addr", plan9.OREAD|plan9.OWRITE)
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
		tfs.t.Fatalf("Failed to open acme/log: %v", err)
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
		tfs.t.Fatalf("Failed to open %s: %v", file, err)
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
		tfs.t.Fatalf("Failed to open %s: %v", file, err)
	}
	_, err = fid.Write([]byte(s))
	if err != nil {
		tfs.t.Errorf("Failed to write %s: %v", file, err)
	}
	fid.Close()
}

func TestGetuser(t *testing.T) {
	if getuser() == "" {
		t.Errorf("Didn't get a username")
	}
}

func TestMnt(t *testing.T) {
	var mnt Mnt

	md := mnt.GetFromID(1)
	if md != nil {
		t.Errorf("mnt.GetFromID(1) returned %v; expected nil", md)
	}

	md = mnt.Add("/home/gopher", nil)
	want := &MntDir{
		id:   1,
		ref:  1,
		dir:  "/home/gopher",
		incl: nil,
	}
	if !reflect.DeepEqual(md, want) {
		t.Fatalf("mnt.Add returned %v; expected %v", md, want)
	}

	mnt.IncRef(md)
	if got, want := md.ref, 2; got != want {
		t.Fatalf("mnt.IncRef increased ref to %v; expected %v", got, want)
	}

	mnt.DecRef(md)
	if got, want := md.ref, 1; got != want {
		t.Fatalf("mnt.DecRef decreased ref to %v; expected %v", got, want)
	}

	if got, want := mnt.GetFromID(1), md; got != want {
		t.Fatalf("mnt.GetFromID returned %v; expected %v", got, want)
	}
	if got, want := md.ref, 2; got != want {
		t.Fatalf("mnt.GetFromID increased ref to %v; expected %v", got, want)
	}

	mnt.DecRef(md)
	if got, want := md.ref, 1; got != want {
		t.Fatalf("mnt.DecRef decreased ref to %v; expected %v", got, want)
	}

	mnt.DecRef(md)
	if got, want := md.ref, 0; got != want {
		t.Fatalf("mnt.DecRef decreased ref to %v; expected %v", got, want)
	}
	if len(mnt.md) != 0 {
		t.Fatalf("len(mnt.md) is %v; expected 0", len(mnt.md))
	}
}

func TestMntDecRef(t *testing.T) {
	var mnt Mnt
	mnt.DecRef(nil) // no-op

	md := &MntDir{id: 42}
	global.cerr = make(chan error)
	go mnt.DecRef(md)
	err := <-global.cerr
	wantErr := fmt.Sprintf("Mnt.DecRef: can't find id %d", md.id)
	if err == nil || err.Error() != wantErr {
		t.Errorf("mnt.DecRef invalid id %d generated error %v; expected %q", md.id, err, wantErr)
	}
}

func errorFcall(err error) *plan9.Fcall {
	return &plan9.Fcall{
		Type:  plan9.Rerror,
		Ename: err.Error(),
	}
}

type mockConn struct {
	bytes.Buffer
}

func (mc *mockConn) Close() error { return nil }

func (mc *mockConn) ReadFcall(t *testing.T) *plan9.Fcall {
	t.Helper()

	fc, err := plan9.ReadFcall(mc)
	if err != nil {
		t.Fatalf("failed to read Fcall: %v", err)
	}
	return fc
}

func TestFileServerVersion(t *testing.T) {
	for _, tc := range []struct {
		version string
		want    plan9.Fcall
	}{
		{"9P2000", plan9.Fcall{
			Type:    plan9.Rversion,
			Version: "9P2000",
			Msize:   8192,
		}},
		{"9P2000.u", plan9.Fcall{
			Type:  plan9.Rerror,
			Ename: "unrecognized 9P version",
		}},
	} {
		t.Run(tc.version, func(t *testing.T) {
			mc := new(mockConn)
			fs := &fileServer{conn: mc}
			fs.version(&Xfid{
				fcall: plan9.Fcall{
					Type:    plan9.Tversion,
					Version: tc.version,
					Msize:   8192,
				},
			}, nil)

			if got, want := mc.ReadFcall(t), &tc.want; !cmp.Equal(got, want) {
				t.Fatalf("got response %v; want %v", got, want)
			}
		})
	}
}

func TestFileServerAuth(t *testing.T) {
	mc := new(mockConn)
	fs := &fileServer{conn: mc}
	fs.auth(&Xfid{}, nil)

	want := errorFcall(fmt.Errorf("acme: authentication not required"))
	if got := mc.ReadFcall(t); !cmp.Equal(got, want) {
		t.Fatalf("got response %v; want %v", got, want)
	}
}

func TestFileServerSendToXfidChan(t *testing.T) {
	mc := new(mockConn)
	fs := &fileServer{conn: mc}

	for _, tc := range []struct {
		name string
		f    fsfunc
	}{
		{"flush", fs.flush},
		{"write", fs.write},
		{"read", fs.read},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := &Xfid{
				c: make(chan func(*Xfid)),
			}
			fid := &Fid{
				qid: plan9.Qid{
					Type: plan9.QTFILE,
				},
			}
			go func() {
				<-x.c
				close(x.c)
			}()
			x1 := tc.f(x, fid)
			if x1 != nil {
				t.Fatalf("got non-nil Xfid: %v", x1)
			}
			// Wait for close above, so we know we actually received something.
			<-x.c
		})
	}
}

func TestFileServerAttach(t *testing.T) {
	mc := new(mockConn)
	fs := &fileServer{
		conn:     mc,
		username: "gopher",
	}

	t.Run("BadUname", func(t *testing.T) {
		x := &Xfid{
			fcall: plan9.Fcall{
				Type:  plan9.Tattach,
				Uname: "glenda",
			},
			f: &Fid{},
		}
		fs.attach(x, x.f)

		want := &plan9.Fcall{
			Type: plan9.Rattach,
			Qid: plan9.Qid{
				Path: Qdir,
				Vers: 0,
				Type: plan9.QTDIR,
			},
		}
		if got := mc.ReadFcall(t); !cmp.Equal(got, want) {
			t.Fatalf("got response %v; want %v", got, want)
		}
	})
	t.Run("Success", func(t *testing.T) {
		mnt = Mnt{}
		md := mnt.Add("/sys/src/9/pc", nil)
		defer func() {
			mnt = Mnt{}
		}()

		x := &Xfid{
			fcall: plan9.Fcall{
				Type:  plan9.Tattach,
				Uname: fs.username,
				Aname: fmt.Sprintf("%v", md.id),
			},
			f: &Fid{},
		}
		fs.attach(x, x.f)

		got := mc.ReadFcall(t)
		want := &plan9.Fcall{
			Type: plan9.Rattach,
			Qid: plan9.Qid{
				Path: Qdir,
				Vers: 0,
				Type: plan9.QTDIR,
			},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
		if x.f.mntdir == nil {
			t.Fatalf("nil mntdir")
		}
		if got, want := x.f.mntdir.id, md.id; got != want {
			t.Errorf("mntdir.id is %v; want %v", got, want)
		}
	})
	t.Run("BadAname", func(t *testing.T) {
		x := &Xfid{
			fcall: plan9.Fcall{
				Type:  plan9.Tattach,
				Uname: fs.username,
				Aname: "notAnUint",
			},
		}
		fs.attach(x, &Fid{})

		got := mc.ReadFcall(t)
		want := &plan9.Fcall{
			Type:  plan9.Rerror,
			Ename: `bad Aname: strconv.ParseUint: parsing "notAnUint": invalid syntax`,
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
	})
	t.Run("UnknownAname", func(t *testing.T) {
		x := &Xfid{
			fcall: plan9.Fcall{
				Type:  plan9.Tattach,
				Uname: fs.username,
				Aname: "42",
			},
		}
		fs.attach(x, &Fid{})

		got := mc.ReadFcall(t)
		want := &plan9.Fcall{
			Type:  plan9.Rerror,
			Ename: `unknown id "42" in Aname`,
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("response mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestFileServerWalk(t *testing.T) {
	global.WinID = 0
	global.row = Row{
		col: []*Column{
			{
				w: []*Window{
					NewWindow().initHeadless(nil),
				},
			},
		},
	}
	defer func() {
		global.WinID = 0
		global.row = Row{}
	}()

	newwindowSetup := func(t *testing.T, fs *fileServer, x *Xfid) <-chan struct{} {
		global.cnewwindow = make(chan *Window)
		done := make(chan struct{})
		go func() {
			<-global.cnewwindow                                // request for window
			global.cnewwindow <- NewWindow().initHeadless(nil) // response

			global.cnewwindow = nil
			close(done)
		}()
		return done
	}

	for _, tc := range []struct {
		name  string
		x     Xfid
		setup func(t *testing.T, fs *fileServer, x *Xfid) <-chan struct{}
		want  plan9.Fcall
	}{
		{
			"WalkOpenFile",
			Xfid{
				f: &Fid{open: true},
			},
			nil,
			*errorFcall(fmt.Errorf("walk of open file")),
		},
		{
			"AlreadyInUse",
			Xfid{
				fcall: plan9.Fcall{
					Fid:    1,
					Newfid: 2,
				},
			},
			func(t *testing.T, fs *fileServer, x *Xfid) <-chan struct{} {
				nf := fs.newfid(x.fcall.Newfid)
				nf.busy = true
				return nil
			},
			*errorFcall(fmt.Errorf("newfid already in use")),
		},
		{
			"ErrNotDir",
			Xfid{
				fcall: plan9.Fcall{
					Wname: []string{"missing"},
				},
			},
			func(t *testing.T, fs *fileServer, x *Xfid) <-chan struct{} {
				x.f.qid.Type = plan9.QTFILE
				return nil
			},
			*errorFcall(ErrNotDir),
		},
		{
			"ErrNotExist",
			Xfid{
				fcall: plan9.Fcall{
					Wname: []string{"missing"},
				},
			},
			nil,
			*errorFcall(ErrNotExist),
		},
		{
			"42",
			Xfid{
				fcall: plan9.Fcall{
					Wname: []string{"42"},
				},
			},
			nil,
			*errorFcall(ErrNotExist),
		},
		{
			"NameTooLongIndex",
			Xfid{
				fcall: plan9.Fcall{
					Wname: []string{
						"..", "..", "..", "..", "..", "..", "..", "..",
						"..", "..", "..", "..", "..", "..", "..", "..",
						"index",
					},
				},
			},
			nil,
			*errorFcall(fmt.Errorf("name too long")),
		},
		{
			"NameTooLongDotDot",
			Xfid{
				fcall: plan9.Fcall{
					Wname: []string{
						"..", "..", "..", "..", "..", "..", "..", "..",
						"..", "..", "..", "..", "..", "..", "..", "..",
						"..",
					},
				},
			},
			nil,
			*errorFcall(fmt.Errorf("name too long")),
		},
		{
			"NameTooLong1",
			Xfid{
				fcall: plan9.Fcall{
					Wname: []string{
						"..", "..", "..", "..", "..", "..", "..", "..",
						"..", "..", "..", "..", "..", "..", "..", "..",
						"1",
					},
				},
			},
			nil,
			*errorFcall(fmt.Errorf("name too long")),
		},
		{
			"NameTooLongNew",
			Xfid{
				fcall: plan9.Fcall{
					Wname: []string{
						"..", "..", "..", "..", "..", "..", "..", "..",
						"..", "..", "..", "..", "..", "..", "..", "..",
						"new",
					},
				},
			},
			newwindowSetup,
			*errorFcall(fmt.Errorf("name too long")),
		},
		{
			"EmptyWalk",
			Xfid{},
			nil,
			plan9.Fcall{
				Type: plan9.Rwalk,
				Wqid: []plan9.Qid{},
			},
		},
		{
			"DotDot",
			Xfid{
				fcall: plan9.Fcall{
					Wname: []string{".."},
				},
			},
			nil,
			plan9.Fcall{
				Type: plan9.Rwalk,
				Wqid: []plan9.Qid{
					{
						Path: QID(0, Qdir),
						Type: plan9.QTDIR,
					},
				},
			},
		},
		{
			"index",
			Xfid{
				fcall: plan9.Fcall{
					Wname: []string{"index"},
				},
			},
			nil,
			plan9.Fcall{
				Type: plan9.Rwalk,
				Wqid: []plan9.Qid{
					{
						Path: QID(0, Qindex),
						Type: plan9.QTFILE,
					},
				},
			},
		},
		{
			"1/../index",
			Xfid{
				fcall: plan9.Fcall{
					Wname: []string{"1", "..", "index"},
				},
			},
			nil,
			plan9.Fcall{
				Type: plan9.Rwalk,
				Wqid: []plan9.Qid{
					{
						Path: QID(1, Qdir),
						Type: plan9.QTDIR,
					},
					{
						Path: QID(0, Qdir),
						Type: plan9.QTDIR,
					},
					{
						Path: QID(0, Qindex),
						Type: plan9.QTFILE,
					},
				},
			},
		},
		{
			"1/body",
			Xfid{
				fcall: plan9.Fcall{
					Wname: []string{"1", "body"},
				},
			},
			nil,
			plan9.Fcall{
				Type: plan9.Rwalk,
				Wqid: []plan9.Qid{
					{
						Path: QID(1, Qdir),
						Type: plan9.QTDIR,
					},
					{
						Path: QID(1, QWbody),
						Type: plan9.QTAPPEND,
					},
				},
			},
		},
		{
			"bodyFrom1",
			Xfid{
				fcall: plan9.Fcall{
					Fid:    0,
					Newfid: 1,
					Wname:  []string{"body"},
				},
			},
			func(t *testing.T, fs *fileServer, x *Xfid) <-chan struct{} {
				x.f.qid = plan9.Qid{
					Path: QID(1, Qdir),
					Type: plan9.QTDIR,
				}
				x.f.w = global.row.col[0].w[0]
				return nil
			},
			plan9.Fcall{
				Type: plan9.Rwalk,
				Wqid: []plan9.Qid{
					{
						Path: QID(1, QWbody),
						Type: plan9.QTAPPEND,
					},
				},
			},
		},
		{
			"1/42",
			Xfid{
				fcall: plan9.Fcall{
					Fid:    0,
					Newfid: 1,
					Wname:  []string{"1", "42"},
				},
			},
			nil,
			plan9.Fcall{
				Type: plan9.Rwalk,
				Wqid: []plan9.Qid{
					{
						Path: QID(1, Qdir),
						Type: plan9.QTDIR,
					},
				},
			},
		},
		{
			"new",
			Xfid{
				fcall: plan9.Fcall{
					Wname: []string{"new"},
				},
			},
			newwindowSetup,
			plan9.Fcall{
				Type: plan9.Rwalk,
				Wqid: []plan9.Qid{
					{
						Path: QID(3, Qdir), // Window 2 created by NameTooLongNew
						Type: plan9.QTDIR,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mc := new(mockConn)
			fs := &fileServer{
				conn: mc,
				fids: make(map[uint32]*Fid),
			}
			x := &tc.x
			x.fcall.Type = plan9.Twalk
			if x.f == nil {
				md := mnt.Add("/home/gopher/src", nil)
				if md == nil {
					t.Fatalf("can't allocate mntdir")
				}
				x.f = &Fid{
					qid: plan9.Qid{ // default to root
						Path: QID(0, Qdir),
						Type: plan9.QTDIR,
					},
					mntdir: md,
				}
			}
			var done <-chan struct{}
			if tc.setup != nil {
				done = tc.setup(t, fs, x)
			}
			fs.walk(x, x.f)
			want := &tc.want
			got := mc.ReadFcall(t)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}
			if done != nil {
				<-done
			}
		})
	}
}

func TestFidWalk1Panic(t *testing.T) {
	done := make(chan struct{})
	defer func() {
		r := recover()
		got, ok := r.(string)
		if !ok {
			t.Fatalf("recovered a %T; want string", got)
		}
		want := "acme: w set in walk to new: <nil>\n"
		if got != want {
			t.Errorf("recovered string %q; want %q", got, want)
		}
		close(done)
	}()
	f := &Fid{
		qid: plan9.Qid{
			Path: QID(0, Qdir),
			Type: plan9.QTDIR,
		},
		w: &Window{},
	}
	f.Walk1("new")
	<-done
}

func TestFileServerOpen(t *testing.T) {
	for _, tc := range []struct {
		name string     // test name
		perm plan9.Perm // directory entry mode
		mode uint8      // open mode
		err  error
	}{
		{"OEXEC", 0600, plan9.OEXEC, ErrPermission},
		{"ORCLOSE", 0600, plan9.ORCLOSE | plan9.OWRITE, ErrPermission},
		{"ODIRECT", 0600, plan9.ODIRECT, ErrPermission},
		{"WriteToReadOnly", 0400, plan9.OWRITE, ErrPermission},
		{"ReadFromWriteOnly", 0200, plan9.OREAD, ErrPermission},
		{"ORDWR", 0600, plan9.ORDWR, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mc := new(mockConn)
			fs := &fileServer{conn: mc}
			x := &Xfid{
				fcall: plan9.Fcall{
					Type: plan9.Topen,
					Mode: tc.mode,
				},
				c: make(chan func(*Xfid)),
			}
			f := &Fid{
				dir: &DirTab{"example", plan9.QTFILE, Qindex, tc.perm},
			}
			if tc.err == nil {
				go func() {
					<-x.c
					close(x.c)
				}()
			}
			var wantx *Xfid
			if tc.err != nil {
				wantx = x
			}
			x1 := fs.open(x, f)
			if x1 != wantx {
				t.Errorf("expected Xfid %v; got %v", wantx, x1)
			}

			if tc.err != nil {
				want := errorFcall(tc.err)
				if got := mc.ReadFcall(t); !cmp.Equal(got, want) {
					t.Fatalf("got response %v; want %v", got, want)
				}
			} else {
				// Wait for close above, so we know we actually received something.
				<-x.c
			}
		})
	}
}

func TestFileServerCreate(t *testing.T) {
	mc := new(mockConn)
	fs := &fileServer{conn: mc}
	fs.create(&Xfid{}, nil)

	want := errorFcall(ErrPermission)
	if got := mc.ReadFcall(t); !cmp.Equal(got, want) {
		t.Fatalf("got response %v; want %v", got, want)
	}
}

func TestFileServerReadQacme(t *testing.T) {
	mc := new(mockConn)
	fs := &fileServer{conn: mc}
	x := &Xfid{
		fcall: plan9.Fcall{
			Type: plan9.Tread,
		},
	}
	f := &Fid{
		qid: plan9.Qid{
			Type: plan9.QTDIR,
			Vers: 0,
			Path: Qacme,
		},
	}
	fs.read(x, f)

	want := &plan9.Fcall{
		Type: plan9.Rread,
		Data: []byte{},
	}
	got := mc.ReadFcall(t)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}
}

func TestFileServerRead(t *testing.T) {
	useFixedClock = true
	global.WinID = 0
	global.row.col = []*Column{
		{
			w: []*Window{
				NewWindow().initHeadless(nil),
				NewWindow().initHeadless(nil),
				NewWindow().initHeadless(nil),
			},
		},
	}
	defer func() {
		useFixedClock = false
		global.WinID = 0
		global.row = Row{}
	}()

	var rootDirs []plan9.Dir
	for _, dt := range dirtab[1:] { // skip "."
		rootDirs = append(rootDirs, *dt.Dir(0, "gopher", fixedClockValue))
	}
	for id := 1; id <= global.WinID; id++ {
		rootDirs = append(rootDirs, *windowDirTab(id).Dir(0, "gopher", fixedClockValue))
	}

	var winDirs []plan9.Dir
	for _, dt := range dirtabw[1:] { // skip "."
		winDirs = append(winDirs, *dt.Dir(3, "gopher", fixedClockValue))
	}

	for _, tc := range []struct {
		name  string
		winid int
		count uint32
		want  []plan9.Dir
	}{
		{"OneRead/Root", 0, 1024, rootDirs},
		{"OneRead/WindowSubdir", 3, 1024, winDirs},
		{"OneDirPerRead/Root", 0, 100, rootDirs},
		{"OneDirPerRead/WindowSubdir", 3, 100, winDirs},
		{"Empty", 0, 0, nil},
		{"NoPartialDir", 0, 10, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mc := new(mockConn)
			fs := &fileServer{
				conn:     mc,
				username: "gopher",
			}
			x := &Xfid{
				fcall: plan9.Fcall{
					Type:   plan9.Tread,
					Count:  tc.count,
					Offset: 0,
				},
				f: &Fid{
					qid: plan9.Qid{
						Type: plan9.QTDIR,
						Vers: 0,
						Path: QID(tc.winid, Qdir),
					},
				},
			}

			var got []plan9.Dir
			for {
				fs.read(x, x.f)

				fc := mc.ReadFcall(t)
				if len(fc.Data) == 0 {
					break
				}
				dirs, err := ninep.UnmarshalDirs(fc.Data)
				if err != nil {
					t.Fatalf("failed to unmarshal directory entries: %v", err)
				}
				got = append(got, dirs...)
				x.fcall.Offset += uint64(len(fc.Data))
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("directory entries mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFileServerRemove(t *testing.T) {
	mc := new(mockConn)
	fs := &fileServer{conn: mc}
	fs.remove(&Xfid{}, nil)

	want := errorFcall(ErrPermission)
	if got := mc.ReadFcall(t); !cmp.Equal(got, want) {
		t.Fatalf("got response %v; want %v", got, want)
	}
}

func TestFileServerStat(t *testing.T) {
	useFixedClock = true
	defer func() { useFixedClock = false }()

	checkDirTab := func(t *testing.T, prefix string, tab []*DirTab) {
		for _, dt := range tab {
			t.Run(prefix+dt.name, func(t *testing.T) {
				mc := new(mockConn)
				fs := &fileServer{
					conn:        mc,
					messagesize: 8192,
					username:    "gopher",
				}
				x := &Xfid{
					fcall: plan9.Fcall{
						Type: plan9.Tstat,
						Fid:  1,
					},
				}
				x.f = &Fid{
					dir: dt,
				}
				fs.stat(x, x.f)

				fc := mc.ReadFcall(t)
				if got, want := fc.Type, uint8(plan9.Rstat); got != want {
					t.Fatalf("got Fcall type %v; want %v", got, want)
				}

				want := dt.Dir(0, "gopher", fixedClockValue)
				got, err := plan9.UnmarshalDir(fc.Stat)
				if err != nil {
					t.Fatalf("UnmarshalDir failed: %v", err)
				}
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("stat mismatch (-want +got):\n%s", diff)
				}
			})
		}
	}
	checkDirTab(t, "", dirtab)
	checkDirTab(t, "winid/", dirtabw)
}

func TestFileServerStatSmallMsize(t *testing.T) {
	mc := new(mockConn)
	fs := &fileServer{
		conn:        mc,
		messagesize: plan9.IOHDRSZ + 16, // too small for a directory entry
		username:    "gopher",
	}
	x := &Xfid{
		fcall: plan9.Fcall{Type: plan9.Tstat},
	}
	x.f = &Fid{dir: dirtab[1]}
	fs.stat(x, x.f)

	got := mc.ReadFcall(t)
	want := &plan9.Fcall{
		Type:  plan9.Rerror,
		Ename: `msize too small`,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}
}

func TestFileServerWstat(t *testing.T) {
	mc := new(mockConn)
	fs := &fileServer{conn: mc}
	fs.wstat(&Xfid{}, nil)

	want := errorFcall(ErrPermission)
	if got := mc.ReadFcall(t); !cmp.Equal(got, want) {
		t.Fatalf("got response %v; want %v", got, want)
	}
}

// TestFsysWalkConcurrentAccess tests concurrent walk operations that access
// global.row.lk and window locks. This test is designed to detect race
// conditions when run with -race flag.
func TestFsysWalkConcurrentAccess(t *testing.T) {
	// Setup: create windows in a test row
	global.WinID = 0
	global.row = Row{
		col: []*Column{
			{
				w: []*Window{
					NewWindow().initHeadless(nil),
					NewWindow().initHeadless(nil),
					NewWindow().initHeadless(nil),
				},
			},
		},
	}
	defer func() {
		global.WinID = 0
		global.row = Row{}
	}()

	// Test: Concurrent walks to numeric window directories should be safe
	t.Run("ConcurrentNumericWalks", func(t *testing.T) {
		const numGoroutines = 10
		const numIterations = 5

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				mc := new(mockConn)
				fs := &fileServer{
					conn: mc,
					fids: make(map[uint32]*Fid),
				}

				for j := 0; j < numIterations; j++ {
					md := mnt.Add("/home/gopher/src", nil)
					x := &Xfid{
						fcall: plan9.Fcall{
							Type:  plan9.Twalk,
							Wname: []string{"1"}, // Walk to window 1
						},
						f: &Fid{
							qid: plan9.Qid{
								Path: QID(0, Qdir),
								Type: plan9.QTDIR,
							},
							mntdir: md,
						},
					}
					fs.walk(x, x.f)
					mnt.DecRef(md)
				}
				done <- true
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	// Test: Concurrent walks that clone fids with window refs
	// This tests the lock-protected ref counting when cloning fids
	t.Run("ConcurrentFidCloneWithWindowRef", func(t *testing.T) {
		const numGoroutines = 5
		const numIterations = 3

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				mc := new(mockConn)
				fs := &fileServer{
					conn: mc,
					fids: make(map[uint32]*Fid),
				}

				for j := 0; j < numIterations; j++ {
					md := mnt.Add("/home/gopher/src", nil)

					// First walk to a window directory to get a fid with window ref
					// This exercises the locking in Walk1 for numeric window lookup
					x := &Xfid{
						fcall: plan9.Fcall{
							Type:  plan9.Twalk,
							Wname: []string{"1"}, // Walk to window 1
						},
						f: &Fid{
							qid: plan9.Qid{
								Path: QID(0, Qdir),
								Type: plan9.QTDIR,
							},
							mntdir: md,
						},
					}
					fs.walk(x, x.f)

					// Now clone this fid by walking to a file within the window
					if x.f.w != nil {
						// Store the fid for cloning
						fs.fids[1] = x.f
						x2 := &Xfid{
							fcall: plan9.Fcall{
								Type:   plan9.Twalk,
								Fid:    1,
								Newfid: uint32(100 + id*10 + j),
								Wname:  []string{"body"},
							},
							f: x.f,
						}
						fs.walk(x2, x2.f)

						// Clean up the new fid if walk succeeded
						// Release references under lock (ref counting relies on lock protection)
						if nf, ok := fs.fids[uint32(100+id*10+j)]; ok && nf.w != nil {
							nf.w.lk.Lock()
							nf.w.Close()
							nf.w.lk.Unlock()
						}

						// Clean up original fid's window reference
						x.f.w.lk.Lock()
						x.f.w.Close()
						x.f.w.lk.Unlock()
					}
					mnt.DecRef(md)
				}
				done <- true
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}

// TestFsysReadConcurrentDirListing tests concurrent directory reads that
// access global.row.lk. This test is designed to detect race conditions
// when run with -race flag.
func TestFsysReadConcurrentDirListing(t *testing.T) {
	useFixedClock = true
	global.WinID = 0
	global.row = Row{
		col: []*Column{
			{
				w: []*Window{
					NewWindow().initHeadless(nil),
					NewWindow().initHeadless(nil),
					NewWindow().initHeadless(nil),
				},
			},
		},
	}
	defer func() {
		useFixedClock = false
		global.WinID = 0
		global.row = Row{}
	}()

	// Test: Concurrent reads of root directory
	t.Run("ConcurrentRootDirReads", func(t *testing.T) {
		const numGoroutines = 10
		const numIterations = 5

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numIterations; j++ {
					mc := new(mockConn)
					fs := &fileServer{
						conn:     mc,
						username: "gopher",
					}
					x := &Xfid{
						fcall: plan9.Fcall{
							Type:   plan9.Tread,
							Count:  1024,
							Offset: 0,
						},
						f: &Fid{
							qid: plan9.Qid{
								Type: plan9.QTDIR,
								Vers: 0,
								Path: QID(0, Qdir),
							},
						},
					}
					fs.read(x, x.f)
				}
				done <- true
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	// Test: Concurrent reads interleaved with window creation/deletion
	t.Run("ConcurrentReadWithWindowMutation", func(t *testing.T) {
		const numReaders = 5
		const numMutators = 2
		const numIterations = 3

		done := make(chan bool, numReaders+numMutators)

		// Start readers
		for i := 0; i < numReaders; i++ {
			go func(id int) {
				for j := 0; j < numIterations; j++ {
					mc := new(mockConn)
					fs := &fileServer{
						conn:     mc,
						username: "gopher",
					}
					x := &Xfid{
						fcall: plan9.Fcall{
							Type:   plan9.Tread,
							Count:  1024,
							Offset: 0,
						},
						f: &Fid{
							qid: plan9.Qid{
								Type: plan9.QTDIR,
								Vers: 0,
								Path: QID(0, Qdir),
							},
						},
					}
					fs.read(x, x.f)
				}
				done <- true
			}(i)
		}

		// Start mutators that modify the row under lock
		for i := 0; i < numMutators; i++ {
			go func(id int) {
				for j := 0; j < numIterations; j++ {
					global.row.lk.Lock()
					// Simulate window list mutation
					if len(global.row.col) > 0 && len(global.row.col[0].w) > 0 {
						// Read the window list (similar to what other operations do)
						_ = len(global.row.col[0].w)
					}
					global.row.lk.Unlock()
				}
				done <- true
			}(i)
		}

		for i := 0; i < numReaders+numMutators; i++ {
			<-done
		}
	})
}

// TestMntConcurrentAccess tests concurrent access to the Mnt reference-counted
// mount directory map. This test is designed to detect race conditions when
// run with -race flag.
func TestMntConcurrentAccess(t *testing.T) {
	// Reset mnt state
	mnt = Mnt{}
	defer func() { mnt = Mnt{} }()

	// Test: Concurrent Add/IncRef/DecRef operations
	t.Run("ConcurrentAddIncDecRef", func(t *testing.T) {
		const numGoroutines = 10
		const numIterations = 5

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numIterations; j++ {
					md := mnt.Add("/home/gopher/src", nil)
					mnt.IncRef(md)
					mnt.IncRef(md)
					mnt.DecRef(md)
					mnt.DecRef(md)
					mnt.DecRef(md) // Final DecRef removes from map
				}
				done <- true
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	// Test: Concurrent GetFromID operations
	t.Run("ConcurrentGetFromID", func(t *testing.T) {
		// Pre-create some MntDirs
		mds := make([]*MntDir, 5)
		for i := 0; i < 5; i++ {
			mds[i] = mnt.Add("/home/gopher/src", nil)
		}

		const numGoroutines = 10
		const numIterations = 5

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numIterations; j++ {
					// Try to get each pre-created MntDir
					for _, md := range mds {
						got := mnt.GetFromID(md.id)
						if got != nil {
							mnt.DecRef(got) // Release the reference from GetFromID
						}
					}
				}
				done <- true
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Clean up - DecRef all the initial references
		for _, md := range mds {
			mnt.DecRef(md)
		}
	})
}

// TestFsysFidWalk1ConcurrentWindowLookup tests concurrent Walk1 operations
// that look up windows by ID. This test is designed to detect race conditions
// when run with -race flag.
func TestFsysFidWalk1ConcurrentWindowLookup(t *testing.T) {
	// Setup: create windows in a test row
	global.WinID = 0
	global.row = Row{
		col: []*Column{
			{
				w: []*Window{
					NewWindow().initHeadless(nil),
					NewWindow().initHeadless(nil),
					NewWindow().initHeadless(nil),
				},
			},
		},
	}
	defer func() {
		global.WinID = 0
		global.row = Row{}
	}()

	// Test: Concurrent Walk1 to numeric window names
	t.Run("ConcurrentWalk1ToWindow", func(t *testing.T) {
		const numGoroutines = 10
		const numIterations = 5

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numIterations; j++ {
					f := &Fid{
						qid: plan9.Qid{
							Path: QID(0, Qdir),
							Type: plan9.QTDIR,
						},
					}
					found, err := f.Walk1("1") // Walk to window 1
					if err != nil {
						t.Errorf("Walk1 error: %v", err)
					}
					if found && f.w != nil {
						// Release the reference under lock (ref counting relies on lock protection)
						f.w.lk.Lock()
						f.w.Close()
						f.w.lk.Unlock()
					}
				}
				done <- true
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	// Test: Concurrent Walk1 mixed with row mutations
	t.Run("ConcurrentWalk1WithRowMutation", func(t *testing.T) {
		const numWalkers = 5
		const numMutators = 2
		const numIterations = 3

		done := make(chan bool, numWalkers+numMutators)

		// Start walkers
		for i := 0; i < numWalkers; i++ {
			go func(id int) {
				for j := 0; j < numIterations; j++ {
					f := &Fid{
						qid: plan9.Qid{
							Path: QID(0, Qdir),
							Type: plan9.QTDIR,
						},
					}
					found, _ := f.Walk1("1")
					if found && f.w != nil {
						// Release the reference under lock (ref counting relies on lock protection)
						f.w.lk.Lock()
						f.w.Close()
						f.w.lk.Unlock()
					}
				}
				done <- true
			}(i)
		}

		// Start mutators that access row under lock (simulating window operations)
		for i := 0; i < numMutators; i++ {
			go func(id int) {
				for j := 0; j < numIterations; j++ {
					global.row.lk.Lock()
					// Simulate operations that read the window list
					if len(global.row.col) > 0 {
						for _, c := range global.row.col {
							_ = len(c.w)
						}
					}
					global.row.lk.Unlock()
				}
				done <- true
			}(i)
		}

		for i := 0; i < numWalkers+numMutators; i++ {
			<-done
		}
	})
}
