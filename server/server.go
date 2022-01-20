package server

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"time"

	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

var EdSrv EdServer = LocalSrv{}

type File interface {
	Stat() (fs.FileInfo, error)
	Read([]byte) (int, error)
	ReadDir(n int) ([]fs.DirEntry, error)
	Write([]byte) (int, error)
	Close() error
}

type EdServer interface {
	Open(name string) (File, error)
	OpenFile(name string, flag int, perm fs.FileMode) (File, error)
	Chdir(dir string) error
	Getwd() (dir string, err error)
	MkdirAll(path string, perm fs.FileMode) error
	Stat(name string) (fs.FileInfo, error)
	SameFile(fi1, fi2 fs.FileInfo) bool
	UserHomeDir() (string, error)

	// Execution
	//Run(*Execution)
	Run(cmd string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) (Execution, error)
}

type LocalSrv struct{}

func (l LocalSrv) Open(name string) (File, error) {
	return os.Open(name)
}

func (l LocalSrv) OpenFile(name string, flag int, perm fs.FileMode) (File, error) {
	return os.OpenFile(name, flag, perm)
}

func (l LocalSrv) Chdir(dir string) error {
	return os.Chdir(dir)
}

func (l LocalSrv) Getwd() (dir string, err error) {
	return os.Getwd()
}

func (l LocalSrv) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (l LocalSrv) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

func (l LocalSrv) SameFile(fi1, fi2 fs.FileInfo) bool {
	return os.SameFile(fi1, fi2)
}

func (l LocalSrv) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

type RemoteSrv struct {
	c    *ssh.Client
	fsc  *client.Conn
	root *client.Fsys
	cwd  string
}

func readPasswd(userAtHost string) func() (secret string, err error) {
	return func() (secret string, err error) {
		fmt.Printf("Password for %s: ", userAtHost)
		bs, err := terminal.ReadPassword(0)
		fmt.Println("")
		return string(bs), err
	}
}

type pipeconn struct {
	*io.PipeReader
	*io.PipeWriter
}

func (*pipeconn) Close() error {
	return nil
}

func StartRemoteSrv(addr string) error {
	useraddr := strings.Split(addr, "@")
	if len(useraddr) != 2 {
		return fmt.Errorf("Expected a remote address in the form user@host:port, but got %s\n", addr)
	}
	user := useraddr[0]
	addr = useraddr[1]
	var rs RemoteSrv
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PasswordCallback(readPasswd(user + "@" + addr)),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO(knusbaum): Don't use insecure host keys
	}

	sshc, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("Failed to dial: %w", err)
	}
	rs.c = sshc

	// Start the 9p session
	s, err := rs.c.NewSession()
	if err != nil {
		return fmt.Errorf("Failed to start ssh session for 9p: %w", err)
	}
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	s.Stdin = r1
	s.Stdout = w2
	s.Stderr = os.Stderr
	go s.Run("bash --login -c 'export9p -v -s -dir /'")
	c, err := client.NewConn(&pipeconn{r2, w1})
	if err != nil {
		return fmt.Errorf("9p client failed initialization: %w", err)
	}
	rs.fsc = c
	root, err := rs.fsc.Attach(nil, user, "")
	if err != nil {
		return fmt.Errorf("9p client failed to attach to root: %w", err)
	}
	rs.root = root
	rs.cwd = "/"
	EdSrv = &rs
	return nil
}

// Open mode 9p file constants
const (
	Oread   = 0
	Owrite  = 1
	Ordwr   = 2
	Oexec   = 3
	None    = 4
	Otrunc  = 0x10
	Orclose = 0x40
)

func convertFlag(mode int) uint8 {
	var m uint8
	switch mode & 0x0F {
	case os.O_RDONLY:
		m = Oread
	case os.O_WRONLY:
		m = Owrite
	case os.O_RDWR:
		m = Ordwr
	}
	if (int(mode) & os.O_TRUNC) > 0 {
		m |= Otrunc
	}
	return m
}

type file9p struct {
	*client.Fid
	dirs []*plan9.Dir
}

func (f *file9p) Stat() (fs.FileInfo, error) {
	d, err := f.Fid.Stat()
	if err != nil {
		return nil, err
	}
	return &fileInfo9p{
		name:    d.Name,
		size:    int64(d.Length),
		mode:    fs.FileMode(d.Mode),
		modTime: time.Unix(int64(d.Mtime), 0),
		isDir:   (d.Mode & plan9.DMDIR) != 0,
		qid:     d.Qid,
	}, nil
}

func (f *file9p) Dirreadall() ([]*plan9.Dir, error) {
	// Note: Cannot use ioutil.ReadAll / io.ReadAll here
	// because it assumes it can read small amounts.
	// Plan 9 requires providing a buffer big enough for
	// at least a single directory entry.
	var dirs []*plan9.Dir
	for {
		d, err := f.Dirread()
		dirs = append(dirs, d...)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return dirs, err
		}
	}
}

func (f *file9p) ReadDir(n int) (fss []fs.DirEntry, errorr error) {
	fmt.Printf("ReadDir(%d) \n", n)
	defer func() { fmt.Printf("ReadDir(%d) -> %v (%v)\n", n, fss, errorr) }()
	if f.dirs == nil {
		// We can make this better. This is memory-inefficient.
		dirs, err := f.Dirreadall()
		if err != nil {
			return nil, err
		}
		fmt.Printf("Got %d directories.\n", len(dirs))
		f.dirs = dirs

	}

	convert := func(ds []*plan9.Dir) []fs.DirEntry {
		ret := make([]fs.DirEntry, len(ds), len(ds))
		for i := range ds {
			ret[i] = dirEntryFromDir(ds[i])
		}
		return ret
	}

	if n >= len(f.dirs) || n <= 0 {
		ret := f.dirs
		f.dirs = f.dirs[:0]
		return convert(ret), nil
	}
	ret := f.dirs[:n]
	f.dirs = f.dirs[n:]
	return convert(ret), nil
}

type dirEntry9p struct {
	name  string
	isDir bool
	fType fs.FileMode
	info  fs.FileInfo
}

func (d *dirEntry9p) Name() string               { return d.name }
func (d *dirEntry9p) IsDir() bool                { return d.isDir }
func (d *dirEntry9p) Type() fs.FileMode          { return d.fType }
func (d *dirEntry9p) Info() (fs.FileInfo, error) { return d.info, nil }

type fileInfo9p struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
	qid     plan9.Qid
}

func (f *fileInfo9p) Name() string       { return f.name }
func (f *fileInfo9p) Size() int64        { return f.size }
func (f *fileInfo9p) Mode() fs.FileMode  { return f.mode }
func (f *fileInfo9p) ModTime() time.Time { return f.modTime }
func (f *fileInfo9p) IsDir() bool        { return f.isDir }
func (f *fileInfo9p) Sys() interface{}   { return nil }

func dirEntryFromDir(d *plan9.Dir) *dirEntry9p {
	de := &dirEntry9p{}
	de.name = d.Name
	de.isDir = (d.Mode & plan9.DMDIR) != 0
	de.fType = fs.FileMode(d.Mode) // Nice. plan9 modes are directly compatible with fs.FileMode
	de.info = &fileInfo9p{
		name:    d.Name,
		size:    int64(d.Length),
		mode:    de.fType,
		modTime: time.Unix(int64(d.Mtime), 0),
		isDir:   de.isDir,
		qid:     d.Qid,
	}
	return de
}

func (l *RemoteSrv) Open(name string) (File, error) {
	f, err := l.root.Open(name, Oread)
	if err != nil {
		return nil, err
	}
	return &file9p{Fid: f}, nil
}

func (l *RemoteSrv) OpenFile(name string, flag int, perm fs.FileMode) (File, error) {
	if flag&os.O_CREATE != 0 {
		f, err := l.root.Create(name, convertFlag(flag), plan9.Perm(perm))
		if err != nil {
			return nil, err
		}
		return &file9p{Fid: f}, nil
	}
	f, err := l.root.Open(name, convertFlag(flag))
	if err != nil {
		return nil, err
	}
	return &file9p{Fid: f}, nil
}

//TODO(knusbaum): relative filenames should be based on cwd. Not sure if this is ever an issue.
func (l *RemoteSrv) Chdir(dir string) error {
	l.cwd = dir
	return nil
}

func (l *RemoteSrv) Getwd() (dir string, err error) {
	return l.cwd, nil
}

func (l *RemoteSrv) MkdirAll(path string, perm fs.FileMode) error {
	return fmt.Errorf("NOT IMPLEMENTED [MkdirAll(%s)]", path)
}

func (l *RemoteSrv) Stat(name string) (fs.FileInfo, error) {
	d, err := l.root.Stat(name)
	if err != nil {
		return nil, err
	}
	f := &fileInfo9p{
		name:    d.Name,
		size:    int64(d.Length),
		mode:    fs.FileMode(d.Mode),
		modTime: time.Unix(int64(d.Mtime), 0),
		isDir:   (d.Mode & plan9.DMDIR) != 0,
		qid:     d.Qid,
	}
	fmt.Printf("Stat(%s) -> %#v\n", name, f)
	return f, nil
}

func (l *RemoteSrv) SameFile(fi1, fi2 fs.FileInfo) bool {
	f9i1, ok := fi1.(*fileInfo9p)
	if !ok {
		return false
	}
	f9i2, ok := fi2.(*fileInfo9p)
	if !ok {
		return false
	}
	fmt.Printf("Comparing %#v with %#v\n", f9i1.qid, f9i2.qid)
	return f9i1.qid == f9i2.qid
}

func (l *RemoteSrv) UserHomeDir() (string, error) {
	s, err := l.c.NewSession()
	if err != nil {
		return "", fmt.Errorf("Failed to start ssh session for homedir: %w", err)
	}
	defer s.Close()
	c, err := s.CombinedOutput("bash --login -c 'echo $HOME'")
	dir := strings.TrimSpace(string(c))
	if dir == "" {
		return "", fmt.Errorf("Failed to obtain home directory.")
	}
	return dir, nil
}

func (l LocalSrv) Run(cmd string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) (Execution, error) {
	c := exec.Command(cmd, args...)
	c.Dir = dir
	c.Stdin = stdin
	c.Stdout = stdout
	c.Stderr = stderr
	c.Env = env
	return &localProc{c: c}, nil
}

type ProcessState interface {
	Execution() Execution
	String() string
	Success() bool
}

type localState struct {
	*os.ProcessState
	e Execution
}

func (s *localState) Execution() Execution {
	return s.e
}

type Execution interface {
	Start() error
	Wait() error // Waits for the process to exit
	Kill() error // Sends a KILL signal to the process
	State() ProcessState
}

type localProc struct {
	c *exec.Cmd
}

func (p *localProc) Start() error {
	return p.c.Start()
}

func (p *localProc) Wait() error {
	return p.c.Wait()
}

func (p *localProc) Kill() error {
	return p.c.Process.Kill()
}

func (p *localProc) State() ProcessState {
	if p.c.ProcessState != nil {
		return &localState{p.c.ProcessState, p}
	}
	return nil
}

type rExecution struct {
	cmd    string
	waited bool
	result error

	// Set by calling Start()
	Session *ssh.Session
}

func (e *rExecution) Start() error {
	fmt.Printf("rSTART\n")
	return e.Session.Start(e.cmd)
}

func (e *rExecution) Wait() error {
	fmt.Printf("rWAIT\n")
	defer func() { e.waited = true }()
	e.result = e.Session.Wait()
	return e.result
}

func (e *rExecution) Kill() error {
	fmt.Printf("rKILL\n")
	defer func() { e.waited = true }()
	return e.Session.Signal(ssh.SIGKILL)
}

func (e *rExecution) State() ProcessState {
	fmt.Printf("rSTATE\n")
	if e.waited {
		fmt.Printf("waited: true [E: %#v]\n", e)
		return &remoteState{e}
	}
	fmt.Printf("waited: false\n")
	return nil
}

type remoteState struct {
	e *rExecution
}

func (s *remoteState) Execution() Execution {
	return s.e
}

func (s *remoteState) String() string {
	if s == nil {
		return "<nil>"
	}
	if s.e.result == nil {
		return "exit status 0"
	}
	return fmt.Sprintf("fail: %s", s.e.result)
}

func (s *remoteState) Success() bool {
	return s.e.result == nil
}

func (l *RemoteSrv) Run(cmd string, args []string, dir string, env []string, stdin io.Reader, stdout, stderr io.Writer) (Execution, error) {
	s, err := l.c.NewSession()
	if err != nil {
		return nil, err
	}
	var command string
	// TODO(knusbaum): Address environment issues
	//for _, kv := range env {
	//	kvs := strings.Split(kv, "=")
	//command += fmt.Sprintf("export %s=%s ; ", kvs[0], shellescape.Quote(kvs[1]))
	// 		kvs := strings.Split(kv, "=")
	// 		//fmt.Printf("KV: %s -> k: [%s] v: [%s]\n", kv, kvs[0], kvs[1])
	// 		err := s.Setenv(kvs[0], kvs[1])
	// 		if err != nil {
	// 			//return nil, err
	// 			log.Printf("Failed to set env: [%s]: %s", kv, err)
	// 		} else {
	// 			log.Printf("Setenv OK [%s] => [%s]", kvs[0], kvs[1])
	// 		}
	//}
	command += fmt.Sprintf("cd '%s' && ", dir)
	// This is required to load profile/path/etc.
	command += "bash --login -c "
	command += cmd + " " + strings.Join(args, " ")
	s.Stdin = stdin
	s.Stdout = stdout
	s.Stderr = stderr
	return &rExecution{cmd: command, Session: s}, nil
}
