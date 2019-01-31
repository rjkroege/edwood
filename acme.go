package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"9fans.net/go/draw"
	"9fans.net/go/plumb"
	"github.com/rjkroege/edwood/frame"
)

var (
	command           *Command
	swapscrollButtons bool
)

// var threaddebuglevel = flag.Int("D", 0, "Thread Debug Level") // TODO(flux): Unused?
var globalautoindentflag = flag.Bool("a", false, "Global AutoIntent")
var bartflagflag = flag.Bool("b", false, "Bart's Flag")
var ncolflag = flag.Int("c", -1, "Number of columns (> 0)")
var varfontflag = flag.String("f", defaultVarFont, "Variable Width Font")
var fixedfontflag = flag.String("F", "/lib/font/bit/lucm/unicode.9.font", "Fixed Width Font")
var loadfileflag = flag.String("l", "", "Load file name")
var mtptflag = flag.String("m", defaultMtpt, "Mountpoint")
var swapscrollbuttonsflag = flag.Bool("r", false, "Swap scroll buttons")
var winsize = flag.String("W", "1024x768", "Window Size (WidthxHeight)")
var ncol = 2

var mainpid int

func main() {

	// rfork(RFENVG|RFNAMEG); TODO(flux): I'm sure these are vitally(?) important.

	runtime.GOMAXPROCS(7)

	flag.Parse()
	ncol = *ncolflag
	globalautoindent = *globalautoindentflag
	loadfile := *loadfileflag
	mtpt = *mtptflag
	bartflag = *bartflagflag
	swapscrollbuttons = *swapscrollbuttonsflag

	cputype = os.Getenv("cputype")
	objtype = os.Getenv("objtype")
	home = os.Getenv("HOME")
	acmeshell = os.Getenv("acmeshell")
	p := os.Getenv("tabstop")
	if p != "" {
		mt, _ := strconv.ParseInt(p, 10, 32)
		maxtab = uint(mt)
	}
	if maxtab == 0 {
		maxtab = 4
	}

	if loadfile != "" {
		fontnames := LoadFonts(loadfile) // Overrides fonts selected up to here.
		if len(fontnames) == 2 {
			*varfontflag = fontnames[0]
			*fixedfontflag = fontnames[1]
		}
	}

	os.Setenv("font", *varfontflag)

	// TODO(flux): this must be 9p open?  It's unused in the C code after its opening.
	// Is it just somehow to keep it open?
	//snarffd = open("/dev/snarf", OREAD|OCEXEC);

	wdir, _ = os.Getwd()

	var err error
	var display *draw.Display
	display, err = draw.Init(nil, *varfontflag, "edwood", *winsize)
	if err != nil {
		log.Fatalf("can't open display: %v\n", err)
	}
	if err := display.Attach(draw.Refnone); err != nil {
		panic("failed to attach to window")
	}
	display.ScreenImage.Draw(display.ScreenImage.R, display.White, nil, image.ZP)

	mousectl = display.InitMouse()
	keyboardctl = display.InitKeyboard()
	mainpid = os.Getpid()

	tagfont = *varfontflag

	iconinit(display)

	cwait = make(chan *os.ProcessState)
	ccommand = make(chan *Command)
	ckill = make(chan string)
	cxfidalloc = make(chan *Xfid)
	cxfidfree = make(chan *Xfid)
	cnewwindow = make(chan *Window)
	csignal = make(chan os.Signal, 1)
	cerr = make(chan error)
	cedit = make(chan int)
	cexit = make(chan struct{})
	cwarn = make(chan uint)

	mousectl = display.InitMouse()
	mouse = &mousectl.Mouse

	startplumbing()
	fs := fsysinit()

	// disk = NewDisk()  TODO(flux): Let's be sure we'll avoid this paging stuff

	const WindowsPerCol = 6

	row.Init(display.ScreenImage.R, display)
	if loadfile == "" || row.Load(loadfile, true) != nil {
		// Open the files from the command line, up to WindowsPerCol each
		files := flag.Args()
		if ncol < 0 {
			if len(files) == 0 {
				ncol = 2
			} else {
				ncol = (len(files) + (WindowsPerCol - 1)) / WindowsPerCol
				if ncol < 2 {
					ncol = 2
				}
			}
		}
		if ncol == 0 {
			ncol = 2
		}
		for i := 0; i < ncol; i++ {
			row.Add(nil, -1)
		}
		rightmostcol := row.col[len(row.col)-1]
		if len(files) == 0 {
			readfile(row.col[len(row.col)-1], wdir)
		} else {
			for i, filename := range files {
				// guide  always goes in the rightmost column
				if filepath.Base(filename) == "guide" || i/WindowsPerCol >= len(row.col) {
					readfile(rightmostcol, filename)
				} else {
					readfile(row.col[i/WindowsPerCol], filename)
				}
			}
		}
	}
	display.Flush()

	// After row is initialized
	go mousethread(display)
	go keyboardthread(display)
	go waitthread()
	go newwindowthread()
	go xfidallocthread(display)

	signal.Ignore(ignoreSignals...)
	signal.Notify(csignal, hangupSignals...)
	for {
		select {
		case <-cexit:
			shutdown(os.Interrupt, fs)

		case s := <-csignal:
			shutdown(s, fs)
		}
	}

}

func readfile(c *Column, filename string) {
	w := c.Add(nil, nil, 0)
	abspath, _ := filepath.Abs(filename)
	w.SetName(abspath)
	w.body.Load(0, filename, true)
	w.body.file.Unmodded()
	w.SetTag()
	w.Resize(w.r, false, true)
	w.body.ScrDraw(w.body.fr.GetFrameFillStatus().Nchars)
	w.tag.SetSelect(w.tag.file.Size(), w.tag.file.Size())
	xfidlog(w, "new")
}

var fontCache = make(map[string]*draw.Font)

func fontget(name string, display *draw.Display) *draw.Font {
	var font *draw.Font
	var ok bool
	if font, ok = fontCache[name]; !ok {
		f, err := display.OpenFont(name)
		if err != nil {
			warning(nil, "can't open font file %s: %v\n", name, err)
			return nil
		}
		fontCache[name] = f
		font = f
	}
	return font
}

var boxcursor = draw.Cursor{
	Point: image.Point{-7, -7},
	Clr: [32]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xF8, 0x1F, 0xF8, 0x1F, 0xF8, 0x1F,
		0xF8, 0x1F, 0xF8, 0x1F, 0xF8, 0x1F, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
	Set: [32]byte{0x00, 0x00, 0x7F, 0xFE, 0x7F, 0xFE, 0x7F, 0xFE,
		0x70, 0x0E, 0x70, 0x0E, 0x70, 0x0E, 0x70, 0x0E,
		0x70, 0x0E, 0x70, 0x0E, 0x70, 0x0E, 0x70, 0x0E,
		0x7F, 0xFE, 0x7F, 0xFE, 0x7F, 0xFE, 0x00, 0x00},
}

func iconinit(display *draw.Display) {
	//TODO(flux): Probably should de-globalize colors.
	if tagcolors[frame.ColBack] == nil {
		tagcolors[frame.ColBack] = display.AllocImageMix(draw.Palebluegreen, draw.White)
		tagcolors[frame.ColHigh], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage.Pix, true, draw.Palegreygreen)
		tagcolors[frame.ColBord], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage.Pix, true, draw.Purpleblue)
		tagcolors[frame.ColText] = display.Black
		tagcolors[frame.ColHText] = display.Black
		textcolors[frame.ColBack] = display.AllocImageMix(draw.Paleyellow, draw.White)
		textcolors[frame.ColHigh], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage.Pix, true, draw.Darkyellow)
		textcolors[frame.ColBord], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage.Pix, true, draw.Yellowgreen)
		textcolors[frame.ColText] = display.Black
		textcolors[frame.ColHText] = display.Black
	}

	// ...
	r := image.Rect(0, 0, display.ScaleSize(Scrollwid+ButtonBorder), fontget(tagfont, display).Height+1)
	button, _ = display.AllocImage(r, display.ScreenImage.Pix, false, draw.Notacolor)
	button.Draw(r, tagcolors[frame.ColBack], nil, r.Min)
	r.Max.X -= display.ScaleSize(ButtonBorder)
	button.Border(r, display.ScaleSize(ButtonBorder), tagcolors[frame.ColBord], image.ZP)

	r = button.R
	modbutton, _ = display.AllocImage(r, display.ScreenImage.Pix, false, draw.Notacolor)
	modbutton.Draw(r, tagcolors[frame.ColBack], nil, r.Min)
	r.Max.X -= display.ScaleSize(ButtonBorder)
	modbutton.Border(r, display.ScaleSize(ButtonBorder), tagcolors[frame.ColBord], image.ZP)
	r = r.Inset(display.ScaleSize(ButtonBorder))
	tmp, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage.Pix, true, draw.Medblue)
	modbutton.Draw(r, tmp, nil, image.ZP)

	r = button.R
	colbutton, _ = display.AllocImage(r, display.ScreenImage.Pix, false, draw.Purpleblue)

	but2col, _ = display.AllocImage(r, display.ScreenImage.Pix, true, 0xAA0000FF)
	but3col, _ = display.AllocImage(r, display.ScreenImage.Pix, true, 0x006600FF)

}

func ismtpt(filename string) bool {
	if mtpt == "" {
		return false
	}
	s := path.Clean(filename)
	return strings.HasPrefix(s, mtpt) && (mtpt[len(mtpt)-1] == '/' || len(s) == len(mtpt) || s[len(mtpt)] == '/')
}

func mousethread(display *draw.Display) {
	// TODO(rjk): Do we need this?
	runtime.LockOSThread()

	for {
		row.lk.Lock()
		flushwarnings()
		row.lk.Unlock()
		display.Flush()
		select {
		case <-mousectl.Resize:
			if err := display.Attach(draw.Refnone); err != nil {
				panic("failed to attach to window")
			}
			display.ScreenImage.Draw(display.ScreenImage.R, display.White, nil, image.ZP)
			iconinit(display)
			ScrlResize(display)
			row.Resize(display.ScreenImage.R)
		case mousectl.Mouse = <-mousectl.C:
			MovedMouse(mousectl.Mouse)
		case <-cwarn:
			// Do nothing
		case pm := <-cplumb:
			if pm.Type == "text" {
				act := findattr(pm.Attr, "action")
				if act == "" || act == "showfile" {
					plumblook(pm)
				} else if act == "showdata" {
					plumbshow(pm)
				}
			}
		}
	}
}

func findattr(attr *plumb.Attribute, s string) string {
	for attr != nil {
		if attr.Name == s {
			return attr.Value
		}
		attr = attr.Next
	}
	return ""
}

func MovedMouse(m draw.Mouse) {
	row.lk.Lock()
	defer row.lk.Unlock()

	t := row.Which(m.Point)

	if t != mousetext && t != nil && t.w != nil &&
		(mousetext == nil || mousetext.w == nil || t.w.id != mousetext.w.id) {
		xfidlog(t.w, "focus")
	}

	if t != mousetext && mousetext != nil && mousetext.w != nil {
		mousetext.w.Lock('M')
		mousetext.eq0 = ^0
		mousetext.w.Commit(mousetext)
		mousetext.w.Unlock()
	}
	mousetext = t
	if t == nil {
		return
	}
	w := t.w
	if t == nil || m.Buttons == 0 {
		return
	}
	but := 0
	switch m.Buttons {
	case 1:
		but = 1
	case 2:
		but = 2
	case 4:
		but = 3
	}
	barttext = t
	if t.what == Body && m.Point.In(t.scrollr) {
		if but != 0 {
			if swapscrollButtons {
				switch but {
				case 1:
					but = 3
				case 3:
					but = 1
				}
			}
			w.Lock('M')
			defer w.Unlock()
			t.eq0 = ^0
			t.Scroll(but)
		}
		return
	}
	// scroll Buttons, wheels, etc.
	if w != nil && (m.Buttons&(8|16)) != 0 {
		if m.Buttons&8 != 0 {
			but = Kscrolloneup
		} else {
			but = Kscrollonedown
		}
		w.Lock('M')
		defer w.Unlock()
		t.eq0 = ^0
		t.Type(rune(but))
		return
	}
	if m.Point.In(t.scrollr) {
		if but != 0 {
			switch t.what {
			case Columntag:
				row.DragCol(t.col, but)
			case Tag:
				t.col.DragWin(t.w, but)
				if t.w != nil {
					barttext = &t.w.body
				}
			}
			if t.col != nil {
				activecol = t.col
			}
		}
		return
	}
	if m.Buttons != 0 {
		if w != nil {
			w.Lock('M')
			defer w.Unlock()
		}
		t.eq0 = ^0
		if w != nil {
			w.Commit(t)
		} else {
			t.Commit()
		}
		switch {
		case m.Buttons&1 != 0:
			t.Select()
			if w != nil {
				w.SetTag()
			}
			argtext = t
			seltext = t
			if t.col != nil {
				activecol = t.col // button 1 only
			}
			if t.w != nil && t == &t.w.body {
				activewin = t.w
			}
		case m.Buttons&2 != 0:
			if q0, q1, argt, ok := t.Select2(); ok {
				execute(t, q0, q1, false, argt)
			}
		case m.Buttons&4 != 0:
			if q0, q1, ok := t.Select3(); ok {
				look3(t, q0, q1, false)
			}
		}
		return
	}
}

func keyboardthread(display *draw.Display) {
	var (
		timer *time.Timer
		t     *Text
	)
	emptyTimer := make(<-chan time.Time)
	timerchan := emptyTimer
	typetext := (*Text)(nil)
	for {
		select {
		case <-timerchan:
			t = typetext
			if t != nil && t.what == Tag {
				t.w.Lock('K')
				t.w.Commit(t)
				t.w.Unlock()
				display.Flush()
			}
		case r := <-keyboardctl.C:
			for {
				typetext = row.Type(r, mouse.Point)
				t = typetext
				if t != nil && t.col != nil && !(r == draw.KeyDown || r == draw.KeyLeft || r == draw.KeyRight) { // scrolling doesn't change activecol
					activecol = t.col
				}
				if t != nil && t.w != nil {
					t.w.body.file.curtext = &t.w.body
				}
				if timer != nil {
					timer.Stop()
				}
				if t != nil && t.what == Tag { // Wait 500 msec to commit a tag.
					timer = time.NewTimer(500 * time.Millisecond)
					timerchan = timer.C
				} else {
					timer = nil
					timerchan = emptyTimer
				}
				select {
				case r = <-keyboardctl.C:
					continue
				default:
					display.Flush()
				}
				break
			}
		}
	}

}

// There is a race between process exiting and our finding out it was ever created.
// This structure keeps a list of processes that have exited we haven't heard of.
type Pid struct {
	pid  int
	msg  string
	next *Pid // TODO(flux) turn this into a slice of Pid
}

func waitthread() {
	var lc, c *Command
	var pids *Pid
	Freecmd := func() {
		if c != nil {
			if c.iseditcommand {
				cedit <- 0
			}
			fsysdelid(c.md)
		}
	}
	for {
	Switch:
		select {
		case err := <-cerr:
			row.lk.Lock()
			warning(nil, "%s", err)
			row.display.Flush()
			row.lk.Unlock()

		case cmd := <-ckill:
			found := false
			for c = command; c != nil; c = c.next {
				if c.name == cmd+" " {
					if err := c.proc.Kill(); err != nil {
						warning(nil, "kill %v: %v\n", cmd, err)
					}
					found = true
				}
			}
			if !found {
				warning(nil, "Kill: no process %v\n", cmd)
			}

		case w := <-cwait:
			pid := w.Pid()
			for c = command; c != nil; c = c.next {
				if c.pid == pid {
					if lc != nil {
						lc.next = c.next
					} else {
						command = c.next
					}
					break
				}
				lc = c
			}
			row.lk.Lock()
			t := &row.tag
			t.Commit()
			if c == nil {
				// helper processes use this exit status
				// TODO(flux): I don't understand what this libthread code is doing
				Untested()
				if strings.HasPrefix(w.String(), "libthread") {
					p := &Pid{}
					p.pid = pid
					p.msg = w.String()
					p.next = pids
					pids = p
				}
			} else {
				if search(t, []rune(c.name)) {
					t.Delete(t.q0, t.q1, true)
					t.SetSelect(0, 0)
				}
				if !w.Success() {
					warning(c.md, "%s: %s\n", c.name, w.String())
				}
				row.display.Flush()
			}
			row.lk.Unlock()
			Freecmd()

		case c = <-ccommand:
			// has this command already exited?
			lastp := (*Pid)(nil)
			for p := pids; p != nil; p = p.next {
				if p.pid == c.pid {
					if p.msg != "" {
						warning(c.md, "%s\n", p.msg)
					}
					if lastp == nil {
						pids = p.next
					} else {
						lastp.next = p.next
					}
					Freecmd()
					break Switch
				}
				lastp = p
			}
			c.next = command
			command = c
			row.lk.Lock()
			t := &row.tag
			t.Commit()
			t.Insert(0, []rune(c.name), true)
			t.SetSelect(0, 0)
			row.display.Flush()
			row.lk.Unlock()
		}
	}
}

// maintain a linked list of Xfid
// TODO(flux): It would be more idiomatic to prep one up front, and block on sending
// it instead of using a send and a receive to get one.
// Frankly, it would be more idiomatic to let the GC take care of them,
// though that would require an exit signal in xfidctl.
func xfidallocthread(d *draw.Display) {
	xfree := (*Xfid)(nil)
	for {
		select {
		case <-cxfidalloc:
			x := xfree
			if x != nil {
				xfree = x.next
			} else {
				x = &Xfid{}
				x.c = make(chan func(*Xfid))
				go xfidctl(x, d)
			}
			cxfidalloc <- x
		case x := <-cxfidfree:
			x.next = xfree
			xfree = x
		}
	}

}

func newwindowthread() {
	var w *Window

	for {
		// only fsysproc is talking to us, so synchronization is trivial
		<-cnewwindow
		w = makenewwindow(nil)
		w.SetTag()
		xfidlog(w, "new")
		cnewwindow <- w
	}

}

func killprocs(fs *fileServer) {
	fs.close()
	for c := command; c != nil; c = c.next {
		c.proc.Kill()
	}
}

var dumping bool

// TODO(rjk): I'm not sure that this is the right thing to do? It fails to
// handle the situation that is most interesting: trying to save the state
// if we would otherwise crash. It's also conceivably racy.
func shutdown(s os.Signal, fs *fileServer) {
	if !dumping && os.Getpid() == mainpid {
		killprocs(fs)
		dumping = true
		row.Dump("")
	} else {
		os.Exit(0)
	}
}

type errorWriter struct{}

func (w errorWriter) Write(data []byte) (n int, err error) {
	n = len(data)
	if n > 0 {
		cerr <- fmt.Errorf(string(data))
	}
	return
}

// Close exists only to satisfy io.WriteCloser interface.
func (w errorWriter) Close() error {
	return nil
}

const MAXSNARF = 100 * 1024

func acmeputsnarf() {
	r := make([]rune, snarfbuf.nc())
	snarfbuf.Read(0, r[:snarfbuf.nc()])
	row.display.WriteSnarf([]byte(string(r)))
}

func acmegetsnarf() {
	b := make([]byte, MAXSNARF)
	n, _, _ := row.display.ReadSnarf(b)
	r, _, _ := cvttorunes(b, n)
	snarfbuf.Reset()
	snarfbuf.Insert(0, r)
}
