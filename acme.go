package main

import (
	"flag"
	"image"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"fmt"

	"9fans.net/go/draw"
	"github.com/rjkroege/edwood/frame"
)

const (
	NSnarf = 1000
)

var (
	snarfrune [NSnarf + 1]rune

	// git clone https://go.googlesource.com/image and install
	// image/font/gofont/ttfs
	fontnames = [2]string{
		"/mnt/font/GoRegular/17a/font",
		"/mnt/font/GoRegular/17a/font",
	}

	command           *Command
	swapscrollButtons bool
)

func timefmt( /*Fmt* */ ) int {
	return 0
}

// var threaddebuglevel = flag.Int("D", 0, "Thread Debug Level") // TODO(flux): Unused?
var globalautoindentflag = flag.Bool("a", false, "Global AutoIntent")
var bartflagflag = flag.Bool("b", false, "Bart's Flag")
var ncolflag = flag.Int("c", -1, "Number of columns (> 0)")
var fixedfontflag = flag.String("f", fontnames[0], "Variable Width Font")
var varfontflag = flag.String("F", fontnames[1], "Fixed Width Font")
var loadfileflag = flag.String("l", "", "Load file name")
var mtptflag = flag.String("m", "", "Mountpoint")
var swapscrollbuttonsflag = flag.Bool("r", false, "Swap scroll buttons")
var winsize = flag.String("W", "1024x768", "Window Size (WidthxHeight)")
var ncol int = 2

var mainpid int

func main() {

	// rfork(RFENVG|RFNAMEG); TODO(flux): I'm sure these are vitally(?) important.

	runtime.GOMAXPROCS(7)

	flag.Parse()
	ncol = *ncolflag
	globalautoindent = *globalautoindentflag
	fontnames[0] = *fixedfontflag
	fontnames[1] = *varfontflag
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
		// TODO(flux)
		// rowloadfonts(loadfile) // Overrides fonts selected up to here.
	}

	os.Setenv("font", fontnames[0])

	// TODO(flux): this must be 9p open?  It's unused in the C code after its opening.
	// Is it just somehow to keep it open?
	//snarffd = open("/dev/snarf", OREAD|OCEXEC);

	wdir, _ = os.Getwd()

	var err error
	var display *draw.Display
	// TODO(rjk): Upstream draw does not have a cexit parameter.
	display, err = draw.Init(nil, fontnames[0], "acme", *winsize)
	if err != nil {
		log.Fatal(err)
	}
	mousectl = display.InitMouse()
	keyboardctl = display.InitKeyboard()
	mainpid = os.Getpid()

	// TODO(flux): Original Acme does a bunch of font cache setup here.
	// I suspect it's not useful in the modern world.
	tagfont = display.DefaultFont

	iconinit(display)
	// rxinit(); // TODO(flux) looks unneeded now

	//cplumb = make(chan *Plumbmsg)
	// cwait = make(chan Waitmsg)
	ccommand = make(chan *Command)
	ckill = make(chan []rune)
	cxfidalloc = make(chan *Xfid)
	cxfidfree = make(chan *Xfid)
	cnewwindow = make(chan *Window)
	mouseexit0 = make(chan int)
	csignal = make(chan os.Signal, 1)
	cerr = make(chan error)
	cedit = make(chan int)
	cexit = make(chan struct {})
	cwarn = make(chan uint) /* TODO(flux): (really chan(unit)[1]) */

	mousectl = display.InitMouse()
	mouse = &mousectl.Mouse

	// startplumbing() // TODO(flux): plumbing
	fsysinit()

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
	go newwindowthread()
	go xfidallocthread(display)

	signal.Notify(csignal /*, hangupsignals...*/)
	for {
		select {
		case <-cexit:
			shutdown(os.Interrupt)

		case s := <-csignal:
			shutdown(s)
		}
	}

}

func readfile(c *Column, filename string) {
	w := c.Add(nil, nil, 0)
	abspath, _ := filepath.Abs(filename)
	w.SetName(abspath)
	w.body.Load(0, filename, true)
	w.body.file.mod = false
	w.dirty = false
	w.SetTag()
	w.Resize(w.r, false, true)
	w.body.ScrDraw()
	w.tag.SetSelect(w.tag.file.b.nc(), w.tag.file.b.nc())
	xfidlog(w, "new")
}

var fontCache map[string]*draw.Font = make(map[string]*draw.Font)

func fontget(fix int, save bool, setfont bool, name string, display *draw.Display) (font *draw.Font) {
	font = nil
	if name == "" {
		name = fontnames[fix]
	}
	var ok bool
	if font, ok = fontCache[name]; !ok {
		f, err := display.OpenFont(name)
		if err != nil {
			warning(nil, "can't open font file %s: %r\n", name)
			return nil
		}
		fontCache[name] = f
		font = f
	}
	if save {
		reffonts[fix] = font
		fontnames[fix] = name
	}
	if setfont {
		tagfont = font // TODO(flux): Global font stuff is just nasty.
		iconinit(display)
	}
	return font
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
	r := image.Rect(0, 0, display.ScaleSize(Scrollwid+ButtonBorder), tagfont.Height+1)
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

	return strings.HasPrefix(filename, mtpt) && (mtpt[len(mtpt)-1] == '/' || filename[len(mtpt)] == '/' || len(filename) == len(mtpt))
}

func mousethread(display *draw.Display) {
	runtime.LockOSThread()

	for {
		// TODO(flux) lock row and flush warnings?
		row.lk.Lock()
		flushwarnings()
		row.lk.Unlock()
		display.Flush()
		select {
		case <-mousectl.Resize:
			if err := display.Attach(draw.Refnone); err != nil {
				panic("failed to attach to window")
			}
			fmt.Println("RESIZE!")
			display.ScreenImage.Draw(display.ScreenImage.R, display.White, nil, image.ZP)
			iconinit(display)
			row.Resize(display.ScreenImage.R)
		case mousectl.Mouse = <-mousectl.C:
			MovedMouse(mousectl.Mouse)
		case <-cwarn:
			break
			//case pm := <-cplumb:
		}
	}
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
	/* scroll Buttons, wheels, etc. */
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
			t.Commit(true)
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
				activecol = t.col /* button 1 only */
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
	return
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
			fmt.Printf("Keypress: %v\n", r)
			for {
				typetext = row.Type(r, mouse.Point)
				t = typetext
				if t != nil && t.col != nil && !(r == draw.KeyDown || r == draw.KeyLeft || r == draw.KeyRight) { /* scrolling doesn't change activecol */
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

func waitthread() {

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
			break
		}
	}

}

func newwindowthread() {
	var w *Window

	for {
		/* only fsysproc is talking to us, so synchronization is trivial */
		<-cnewwindow
		w = makenewwindow(nil)
		w.SetTag()
		xfidlog(w, "new")
		cnewwindow <- w
	}

}

func plumbproc() {

}

func killprocs() {
	fsysclose()
	/*
		for _, c := range command {
			c.Signal(os.Interrupt)
		}
	*/
}

var dumping bool

var hangupsignals = []os.Signal{
	os.Signal(syscall.SIGINT),
	os.Signal(syscall.SIGHUP),
	os.Signal(syscall.SIGQUIT),
	os.Signal(syscall.SIGSTOP),
}

func shutdown(s os.Signal) {
	fmt.Println("Exiting!", s)
	killprocs()
	if !dumping && os.Getpid() == mainpid {
		dumping = true
		row.Dump("")
	}
	for _, sig := range hangupsignals {
		if sig == s {
			os.Exit(0)
		}
	}
	return
}
