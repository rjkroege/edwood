package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"unicode/utf8"

	"9fans.net/go/acme"
	"github.com/creack/pty"
)

const termprog = "win"

func usage() {
	fmt.Fprintf(os.Stderr, "usage: win cmd args...\n")
	os.Exit(0)
}

var debug = false

func debugf(format string, args ...interface{}) {
	if debug {
		fmt.Fprintf(os.Stderr, "Debug: "+format, args...)
	}
}

var blank = &acme.Event{C1: 'M', C2: 'X', Nr: 1, Nb: 1, Text: []byte(" ")}

type winWin struct {
	W *acme.Win

	Q    sync.Mutex
	p, k int

	typing     []rune // typing not yet delivered to shell
	ntypebreak int    // how many lines are not yet delivered
	cook       bool
	password   bool

	rcpty *os.File
	rctty *os.File

	echo    EchoManager
	sysname string
}

func NewWinWin() (*winWin, error) {
	w, err := acme.New()
	if err != nil {
		return nil, err
	}
	// TODO(PAL): sysname
	win := &winWin{W: w, cook: true, typing: []rune{}, echo: NewEchoManager(), sysname: "win"}
	return win, nil
}

func (w *winWin) Read(b []byte) (int, error) {
	n, e := w.W.Read("body", b)
	return n, e
}

func (w *winWin) Write(b []byte) (int, error) {
	n, e := w.W.Write("body", b)
	return n, e
}

func eToS(e *acme.Event) string {
	return fmt.Sprintf("C1:%c C2:%c Q0:%v Q1:%v OQ0:%v OQ1:%v Flag:%v Text:%s Arg:%s Loc:%s",
		e.C1, e.C2, e.Q0, e.Q1, e.OrigQ0, e.OrigQ1, e.Flag, e.Text, e.Arg, e.Loc)
}

func (w *winWin) Printf(file string, format string, args ...interface{}) error {
	_, err := w.W.Write(file, []byte(fmt.Sprintf(format, args...)))
	return err
}

func (w *winWin) israw() bool {
	return (!w.cook && w.password) && !isecho(w.rcpty)
}

// Add text, either in the body, or at the not-yet-sent insertion point
// The text might be in the message, or may need to come from
// the data stream
func (w *winWin) typetext(e *acme.Event) {
	debugf("typetext %v @ w.p=%d\n", eToS(e), w.p)
	buf := make([]byte, 256) // TODO(PAL): Don't want to allocate this every time!
	// The C code puts it on the stack.  Maybe go does too?

	if e.Nr > 0 {
		w.addtype(e.C1, e.Q0-w.p, []rune(string(e.Text)))
	} else {
		// read body[Q0:Q1]
		m := e.Q0
		for m < e.Q1 {
			// Seek to Q0
			w.Printf("addr", "#%d", e.Q0)

			// Read runes from Q0-Q1
			n, err := w.W.Read("data", buf)
			if err != nil || n != len(buf) {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				break
			}
			rbuf := []rune(string(buf))
			w.addtype(e.C1, m-w.p, rbuf)
			m += len(rbuf)
		}
	}
	if w.israw() { // Obscure the text.
		w.Printf("addr", "#%d,#%d", e.Q0, e.Q1)
		w.W.Write("data", []byte(""))
		w.p -= e.Q1 - e.Q0
	}
	w.sendtype()
	if len(e.Text) > 0 && e.Text[len(e.Text)-1] == '\n' {
		w.cook = false
	}
}

// Insert text into typing at p0
func (w *winWin) addtype(c rune, p0 int, text []rune) {
	for _, r := range text {
		if (r == 0x7F || r == 3) && c == 'K' { // del and ^c from keyboard
			w.rcpty.Write([]byte{byte(r)})
			/* toss all typing */
			w.p += len(w.typing) + len(text)
			w.typing = w.typing[0:0]
			w.ntypebreak = 0
			/* buglet:  more than one delete ignored */
			return
		}
		if r == '\n' || r == 0x04 {
			w.ntypebreak++
		}
	}
	w.typing = append(append(w.typing[0:p0], text...), w.typing[p0:]...)
}

// Send to the process
func (w *winWin) sendtype() {
	raw := w.israw()
lineloop:
	for w.ntypebreak != 0 || (raw && len(w.typing) > 0) {
		for i, r := range w.typing {
			if r == '\n' || r == 0x04 || (i == len(w.typing)-1 && raw) {
				if (r == '\n' || r == 0x04) && w.ntypebreak > 0 {
					w.ntypebreak--
				}
				n := i + 1
				if !raw {
					w.echo.Echoed(w.typing)
				}
				n, err := w.rcpty.Write([]byte(string(w.typing[0:n])))
				if n != i+1 || err != nil {
					fmt.Fprintf(os.Stderr, "sending to program")
				}
				w.p += len([]rune(string(w.typing[0:n])))
				copy(w.typing[0:len(w.typing)-n], w.typing[n:])
				w.typing = w.typing[0 : len(w.typing)-n]
				continue lineloop
			}
		}
		fmt.Fprintf(os.Stderr, "no breakchar\n")
		w.ntypebreak = 0
	}
}

func (w *winWin) delete(e *acme.Event) int {
	deltap := 0

	q0 := e.Q0
	q1 := e.Q1
	if q1 <= w.p {
		return e.Q1 - e.Q0
	}
	if q0 >= w.p+len(w.typing) {
		return 0
	}
	deltap = 0
	if q0 < w.p {
		deltap = w.p - q0
		q0 = 0
	} else {
		q0 -= w.p
	}
	if q1 > w.p+len(w.typing) {
		q1 = len(w.typing)
	} else {
		q1 -= w.p
	}
	w.deltype(q0, q1)
	return deltap
}

func (w *winWin) deltype(p0, p1 int) {
	for _, r := range w.typing[p0:p1] {
		if r == '\n' || r == 0x04 {
			w.ntypebreak--
		}
	}
	w.typing = append(w.typing[:p0], w.typing[p1:]...)
}

func (w *winWin) sendbs(n int) {
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = 0x08
	}
	for n > 0 {
		nw, err := w.rcpty.Write(buf)
		if err != nil {
			panic("Bad write")
		}
		if nw != len(buf) {
			buf = buf[nw:]
		}
	}
}

func events(win *winWin) {
	for e := range win.W.EventChan() {
		win.Q.Lock()

		switch e.C1 {
		case 'E': // write to body or tag; can't affect us
			switch e.C2 {
			case 'I', 'D':
				win.p += e.Q1 - e.Q0 // Track the output point
			case 'i', 'd': // Tag
			default:
				// Unknown?
				fmt.Fprintf(os.Stderr, "Unknown event: %v\n", e)
			}
		case 'F': // Generated by our own actions, ignore
		case 'K', 'M':
			switch e.C2 {
			case 'I':
				if e.Nr == 1 && e.Text[0] == 0x7F { // One key, it's 0x7F, delete
					win.Printf("addr", "#%d,#%d", e.Q0, e.Q1)
					win.Printf("data", "")
					buf := []byte{0x7F}
					win.rcpty.Write(buf)
					break
				}
				if e.Q0 < win.p {
					//if(debug)
					//	fprint(2, "shift typing %d... ", e.q1-e.q0)
					win.p += e.Q1 - e.Q0
				} else if e.Q0 <= win.p+len(win.typing) {
					//if(debug)
					//	fprint(2, "type... ");
					win.typetext(e)
				}

			case 'D':
				n := win.delete(e)
				win.p -= n
				if win.israw() && e.Q1 >= win.p+n {
					win.sendbs(n)
				}

			case 'X', 'x':
				/*
					var e2, e3 *acme.Event
					if e.Flag&2 != 0 {
						e2 = <-win.W.EventChan()
					}
					if e.Flag&8 != 0 {
						e3 = <-win.W.EventChan()
						_ = <-win.W.EventChan()
					}
				*/
				if (e.Flag&1 != 0) || (e.C2 == 'x' && e.Nr == 0 /*&& e2.Nr == 0*/) {
					/* send it straight back */
					//fsfidprint(efd, "%c%c%d %d\n", e.c1, e.c2, e.q0, e.q1);
					win.W.WriteEvent(e)
				}
				/*
					if e.Q0 == e.Q1 && (e.Flag&2 != 0) {
						e2.Flag = e.Flag
						e = e2
					}
				*/
				switch {
				case string(e.Text) == "cook":
					win.cook = true
				case string(e.Text) == "nocook":
					win.cook = false
				case e.Flag&8 != 0:
					if e.Q1 != e.Q0 {
						win.W.WriteEvent(e)
						//	win.W.WriteEvent(blank)
						//sende(&e, fd0, cfd, afd, dfd, 0);
						//sende(&blank, fd0, cfd, afd, dfd, 0);
					}
					//win.W.WriteEvent(e3)
					//sende(&e3, fd0, cfd, afd, dfd, 1)
				default:
					if e.Q1 != e.Q0 {
						win.W.WriteEvent(e)
						//sende(&e, fd0, cfd, afd, dfd, 1)
					}
				}

			case 'l', 'L':
				/* just send it back */
				win.W.WriteEvent(e)

			case 'd', 'i':

			default:
				fmt.Fprintf(os.Stderr, "Unknown event: %v\n", e)

			}
		}
		debugf("%#v\n", eToS(e))
		//win.WriteEvent(e)

		win.Q.Unlock()
	}
	os.Exit(0)
}

func startProcess(arg string, args []string, w *winWin) {
	cmd := exec.Command(arg, args...)
	var err error
	cmd.Env = append(os.Environ(), []string{"TERM=dumb",
		fmt.Sprintf("winid=%d", w.W.ID())}...)
	/*
		w.rcpty, w.rctty, err = termios.Pty()
		if err != nil {
			panic(err)
		}
		cmd.Stdout = w.rctty
		cmd.Stderr = w.rctty
		cmd.Stdin = w.rctty
		err = cmd.Start()
	*/
	w.rcpty, err = pty.Start(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running: %v", err)
	}

	debugf("%s on pid %d\n", arg, cmd.Process.Pid)
}

func main() {
	//usage()
	//signal.Notify
	win, err := NewWinWin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "win: Failed to open acme window\n")
		os.Exit(0)
	}

	pwd, _ := os.Getwd()
	pwdSlash := strings.TrimSuffix(pwd, "/") + "/"
	err = win.W.Name(pwdSlash + "+win")
	if err != nil {
		fmt.Fprintf(os.Stderr, "win: Failed to set name\n")
		os.Exit(0)
	}

	win.W.Write("tag", []byte("Send"))

	stty := exec.Command("stty", "stty", "tabs", "-onlcr", "icanon", "echo", "erase", "^h", "intr", "^?")
	stty.Run()

	// TODO(PAL): Better selection of shell.
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "rc"
	}
	startProcess(shell, []string{"-i"}, win)
	go win.stdoutproc()
	events(win)
}

// TODO(PAL): Somehow redirection into /dev/tty isn't getting picked up by the read.
func (w *winWin) stdoutproc() {
	buf := make([]byte, 8192)

	// I read on win and write it to data.
	var partialRune []byte
	for {
		n, err := w.rcpty.Read(buf[len(partialRune):])
		if err == io.EOF {
			continue
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "win: error reading rcw: %v\n", err)
			panic(err)
		}

		r, m := utf8.DecodeLastRune(buf[:n+len(partialRune)])
		if m != 0 && r == utf8.RuneError {
			// Partial rune at end of read buffer
			partialRune = buf[n-m : n]
		}

		// TODO(PAL): squash nuls, password
		// Partial runes clue: unicode/utf8.DecodeLastRuneInString returns RuneError and count.
		if n > 0 {
			input := []rune(string(buf[0 : n-len(partialRune)]))

			input = w.echo.Cancel(input)

			input = dropcrnl(input)

			input = dropcr(input)

			input = squashnulls(input)

			input = w.label(input) // Processes the awd cookie

			w.password = false
			istring := string(input)
			if strings.Contains(strings.ToLower(istring), "password") || strings.Contains(strings.ToLower(istring), "passphrase") {
				// remove trailing spaces
				istring = strings.TrimRight(istring, " ")
				w.password = len(istring) > 0 && istring[len(istring)-1] == ':'
				debugf("password elision: %v\n", w.password)
				input = []rune(istring)
			}

			w.Q.Lock()
			//err := w.Printf("addr", "#%d", w.p)
			err := w.Printf("addr", "$")
			if err != nil {
				fmt.Fprintf(os.Stderr, "TODO(PAL): reset addr: %v", err)
				w.Printf("addr", "$")
				// Go to $, read the address back, set to p.
			}
			n, err := w.W.Write("data", []byte(string(input)))
			if err != nil || n != len([]byte(string(input))) {
				fmt.Fprintf(os.Stderr, "Problem flushing body")
			}
			w.p += len([]rune(string(input)))
			debugf("w.p == %d\n", w.p)
			// Copy the partial to the front of the buffer
			copy(buf, partialRune)
			partialRune = partialRune[0:0]
			w.Q.Unlock()
		}
	}
}

type EchoManager struct {
	sync.Mutex
	buf []rune
}

func NewEchoManager() EchoManager {
	return EchoManager{buf: make([]rune, 0)}
}

// Things typed are recorded so we can suppress the
// echo that comes back from the pty.
func (echo *EchoManager) Echoed(input []rune) {
	echo.Lock()
	defer echo.Unlock()
	echo.buf = append(echo.buf, input...)
}

func min(l, r int) int {
	if l < r {
		return l
	}
	return r
}

// Suppress matching input from the echo manager buffer.
func (echo *EchoManager) Cancel(input []rune) []rune {
	echo.Lock()
	defer echo.Unlock()
	var i, r int
	for i = 0; i < len(input); i++ {
		if r < len(echo.buf) {
			if echo.buf[r] == input[i] {
				r++
				continue
			}
			if echo.buf[r] == '\n' && input[i] == '\r' {
				continue
			}
			if input[i] == 0x08 { // backspace?
				if i+2 <= len(input) && input[i+1] == ' ' && input[i+2] == 0x08 {
					i += 2
				}
				continue
			}
		}
		break
	}
	copy(echo.buf, echo.buf[r:])
	echo.buf = echo.buf[0 : len(echo.buf)-r]

	return input[i:]
}

// TODO(PAL): Doesn't handle the "\b \b" pattern which I don't understand.
func dropcrnl(p []rune) []rune {
	s := string(p)
	return []rune(strings.Replace(s, "\r\n", "\n", -1))
}

func squashnulls(p []rune) []rune {
	s := string(p)
	return []rune(strings.Replace(s, "\x00", "", -1))
}

func dropcr(p []rune) []rune {
	var r, w int
	for i := 0; i < len(p); i++ {
		switch p[r] {
		case '\b':
			if w > 0 {
				w--
			}
		case '\r':
			for r < len(p)-2 && p[r+1] == '\r' {
				r++
				i++
			}
			if r < len(p)-1 && p[r+1] != '\n' {
				q := r
				for q > 0 && p[q-1] != '\n' {
					q--
				}
				if q > 0 {
					w = q
					break
				}
			}
			p[w] = '\n'
			w++
		default:
			p[w] = p[r]
			w++
		}
		r++
	}
	return p[:w]
}

// echo testname | awk '{printf("\033];%s\007", $0);}' >/dev/tty
func (w *winWin) label(input []rune) []rune {
	// Strip out the last segment of the form "\033];%s\007", form a window label from it,
	// and send to ctl.

	i := len(input) - 1
	for ; i > 3 && input[i] != '\007'; i-- {
	}
	if i <= 3 {
		return input
	}
	endOfString := i
	foundName := false
	for ; i >= 0; i-- {
		if input[i] == '-' {
			foundName = true
		}
		if input[i] == '\033' && input[i+1] == ']' && input[i+2] == ';' {
			windowname := input[i+3 : endOfString]
			if !foundName {
				windowname = append(windowname, '/', '-')
				windowname = append(windowname, []rune(w.sysname)...)
			}
			input = append(input[0:i], input[endOfString+1:]...)
			w.Printf("ctl", "name %s\n", string(windowname))
			return input
		}
	}

	return input
}
