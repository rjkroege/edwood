package main

import (
	"flag"
	"image"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"fmt"

	"9fans.net/go/draw"
	"github.com/paul-lalonde/acme/frame"
)

const (
	NSnarf = 1000
)

var (
	snarfrune [NSnarf + 1]rune

	fontnames = [2]string{
		"/lib/font/bit/lucsans/euro.8.font",
		"/lib/font/bit/lucm/unicode.9.font",
	}

	command           *Command
	swapscrollButtons bool
)

func timefmt( /*Fmt* */ ) int {
	return 0
}

func main2() {
	var cols [5]*draw.Image
	errch := make(chan<- error)
	display, err := draw.Init(errch, "", "acme", "1024x720")
	if err != nil {
		panic(err)
	}
	img, err := display.AllocImage(image.Rect(0, 0, 1024, 720), draw.RGB16, true, draw.Cyan)
	if err != nil {
		panic(err)
	}
	f := frame.NewFrame(image.Rect(0, 0, 500, 600), display.DefaultFont, img, cols)

	for {
		f.Tick(image.ZP, true)
	}
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
var winsize = flag.String("W", "1024x768", "Window Size (WidthxHeight)") // TODO(flux): Unused?

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
	display, err = draw.Init(nil, fontnames[0], "acme", *winsize) // TODO(flux):
	if err != nil {
		log.Fatal(err)
	}
	mousectl = display.InitMouse()
	keyboardctl = display.InitKeyboard()
	mainpid = os.Getpid()

	// TODO(flux): Original Acme does a bunch of font cache setup here.
	// I suspect it's not useful in the modern world.
	tagfont = display.DefaultFont

	// TODO(flux)
	iconinit()
	//	timerinit()
	//	rxinit();

	// cplumb = make(chan *Plumbmsg) TODO(flux): There must be a plumber library in go...
	// cwait = make(chan Waitmsg)
	ccommand = make(chan Command)
	ckill = make(chan []rune)
	cxfidalloc = make(chan *Xfid)
	cxfidfree = make(chan *Xfid)
	cnewwindow = make(chan chan interface{})
	mouseexit0 = make(chan int)
	cexit = make(chan int)
	cerr = make(chan string)
	cedit = make(chan int)
	cwarn = make(chan uint) /* TODO(flux): (really chan(unit)[1]) */

	mousectl = display.InitMouse()
	mouse = &mousectl.Mouse

	// startplumbing() // TODO(flux): plumbing
	// fsysinit()  // TODO(flux): I don't even have a clue to the design of Fsys here.

	// disk = NewDisk()  TODO(flux): Let's be sure we'll avoid this paging stuff

	const WindowsPerCol = 6

	row.Init(display.ScreenImage.R)
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
		rightmostcol := row.col[row.ncol()-1]
		if len(files) == 0 {
			readfile(row.col[row.ncol()-1], wdir)
		} else {
			for i, filename := range files {
				// guide  always goes in the rightmost column
				if filepath.Base(filename) == "guide" || uint(i)/WindowsPerCol >= row.ncol() {
					readfile(rightmostcol, filename)
				} else {
					readfile(row.col[i/WindowsPerCol], filename)
				}
			}
		}
	}
	display.Flush()

	// After row is initialized
	go mousethread()

	select {
	case <-cexit:
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
	// textscrdraw(&w->body)  // TODO(flux): Scroll bars
	// w.tag.SetSelect(w.tag.file.b.nc(), w.tag.file.b.nc()) //  TODO(flux): tag text uninitialized here.
	// xfidlog(w, "new")  // TODO(flux): Wish I knew what the xfid log did
}

var fontCache map[string]*draw.Font = make(map[string]*draw.Font)

// TODO(flux): I don't refcount the fonts.  They aren't so large that we can't
// keep them around until the end of time.
func fontget(fix int, save bool, setfont bool, name string) (font *draw.Font) {
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
		tagfont = font
		iconinit()
	}
	return font
}

func iconinit() {
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

func mousethread() {
	runtime.LockOSThread()

	for {
		// TODO(flux) lock row and flush warnings?

		select {
		case <-mousectl.Resize:
			if err := display.Attach(draw.Refnone); err != nil {
				panic("failed to attach to window")
			}
			fmt.Println("RESIZE!")
			display.ScreenImage.Draw(display.ScreenImage.R, display.White, nil, image.ZP)
			iconinit()
			row.Resize(display.ScreenImage.R)
		case mousectl.Mouse = <-mousectl.C:
			MovedMouse(mousectl.Mouse)
		}
	}
}

func MovedMouse(m draw.Mouse) {
	row.lk.Lock()
	defer row.lk.Unlock()

	t := row.Which(m.Point)

	if t != mousetext && t != nil && t.w != nil &&
		(mousetext == nil || mousetext.w == nil || t.w.id != mousetext.w.id) {
		// xfidlog(t.w, "focus");  TODO(flux)
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
	but := uint(0)
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

func keyboardthread() {

}

func waitthread() {

}

func xfidallocthread() {

}

func newwindowthread() {

}

func plumbproc() {

}
