package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"9fans.net/go/plumb"
	"github.com/rjkroege/edwood/internal/draw"
	"github.com/rjkroege/edwood/internal/dumpfile"
	"github.com/rjkroege/edwood/internal/frame"
)

var (
	command []*Command

	debugAddr         = flag.String("debug", "", "Serve debug information on the supplied address")
	globalAutoIndent  = flag.Bool("a", false, "Start each window in autoindent mode")
	barflag           = flag.Bool("b", false, "Click to focus window instead of focus follows mouse (Bart's flag)")
	varfontflag       = flag.String("f", defaultVarFont, "Variable-width font")
	fixedfontflag     = flag.String("F", defaultFixedFont, "Fixed-width font")
	mtpt              = flag.String("m", defaultMtpt, "Mountpoint for 9P file server")
	swapScrollButtons = flag.Bool("r", false, "Swap scroll buttons")
	winsize           = flag.String("W", "1024x768", "Window size and position as WidthxHeight[@X,Y]")
)

func main() {

	// rfork(RFENVG|RFNAMEG); TODO(flux): I'm sure these are vitally(?) important.

	runtime.GOMAXPROCS(7)

	var (
		ncol     int
		loadfile string
	)
	flag.IntVar(&ncol, "c", 2, "Number of columns at startup")
	flag.StringVar(&loadfile, "l", "", "Load state from file generated with Dump command")
	flag.Parse()

	if *debugAddr != "" {
		go func() {
			log.Println(http.ListenAndServe(*debugAddr, nil))
		}()
	}

	var err error
	home, err = os.UserHomeDir()
	if err != nil {
		log.Fatalf("could not get user home directory: %v", err)
	}
	acmeshell = os.Getenv("acmeshell")
	p := os.Getenv("tabstop")
	if p != "" {
		mt, _ := strconv.ParseInt(p, 10, 32)
		maxtab = uint(mt)
	}
	if maxtab == 0 {
		maxtab = 4
	}

	b := os.Getenv("tabexpand")
	if b != "" {
		te, _ := strconv.ParseBool(b)
		texpand = te
	} else {
		texpand = false
	}

	var dump *dumpfile.Content

	if loadfile != "" {
		d, err := dumpfile.Load(loadfile) // Overrides fonts selected up to here.
		if err != nil {
			// Maybe it's in legacy format. Try that too.
			d, err = dumpfile.LoadLegacy(loadfile, home)
		}
		if err == nil {
			if d.VarFont != "" {
				*varfontflag = d.VarFont
			}
			if d.FixedFont != "" {
				*fixedfontflag = d.FixedFont
			}
			dump = d
		}
	}

	os.Setenv("font", *varfontflag)

	// TODO(flux): this must be 9p open?  It's unused in the C code after its opening.
	// Is it just somehow to keep it open?
	//snarffd = open("/dev/snarf", OREAD|OCEXEC);

	wdir, _ = os.Getwd()

	draw.Main(func(dd *draw.Device) {
		display, err := dd.NewDisplay(nil, *varfontflag, "edwood", *winsize)
		if err != nil {
			log.Fatalf("can't open display: %v\n", err)
		}
		if err := display.Attach(draw.Refnone); err != nil {
			panic("failed to attach to window")
		}
		display.ScreenImage().Draw(display.ScreenImage().R(), display.White(), nil, image.Point{})

		mousectl = display.InitMouse()
		keyboardctl = display.InitKeyboard()

		tagfont = *varfontflag

		iconinit(display)

		cwait = make(chan ProcessState)
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

		row.Init(display.ScreenImage().R(), display)
		if loadfile == "" || row.Load(dump, loadfile, true) != nil {
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
		ctx := context.Background()
		go mousethread(display)
		go keyboardthread(display)
		go waitthread(ctx)
		go newwindowthread()
		go xfidallocthread(ctx, display)

		signal.Ignore(ignoreSignals...)
		signal.Notify(csignal, hangupSignals...)

		select {
		case <-cexit:
			// Do nothing.
		case <-csignal:
			row.lk.Lock()
			row.Dump("")
			row.lk.Unlock()
		}
		killprocs(fs)
		os.Exit(0)
	})
}

func readfile(c *Column, filename string) {
	w := c.Add(nil, nil, 0)
	abspath, _ := filepath.Abs(filename)
	w.SetName(abspath)
	w.body.Load(0, filename, true)
	w.body.file.Clean()
	w.SetTag()
	w.Resize(w.r, false, true)
	w.body.ScrDraw(w.body.fr.GetFrameFillStatus().Nchars)
	w.tag.SetSelect(w.tag.file.Size(), w.tag.file.Size())
	xfidlog(w, "new")
}

var fontCache = make(map[string]draw.Font)

func fontget(name string, display draw.Display) draw.Font {
	var font draw.Font
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

func iconinit(display draw.Display) {
	//TODO(flux): Probably should de-globalize colors.
	if tagcolors[frame.ColBack] == nil {
		tagcolors[frame.ColBack] = display.AllocImageMix(draw.Palebluegreen, draw.White)
		tagcolors[frame.ColHigh], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palegreygreen)
		tagcolors[frame.ColBord], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Purpleblue)
		tagcolors[frame.ColText] = display.Black()
		tagcolors[frame.ColHText] = display.Black()
		textcolors[frame.ColBack] = display.AllocImageMix(draw.Paleyellow, draw.White)
		textcolors[frame.ColHigh], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Darkyellow)
		textcolors[frame.ColBord], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Yellowgreen)
		textcolors[frame.ColText] = display.Black()
		textcolors[frame.ColHText] = display.Black()
	}

	// ...
	r := image.Rect(0, 0, display.ScaleSize(Scrollwid+ButtonBorder), fontget(tagfont, display).Height()+1)
	button, _ = display.AllocImage(r, display.ScreenImage().Pix(), false, draw.Notacolor)
	button.Draw(r, tagcolors[frame.ColBack], nil, r.Min)
	r.Max.X -= display.ScaleSize(ButtonBorder)
	button.Border(r, display.ScaleSize(ButtonBorder), tagcolors[frame.ColBord], image.Point{})

	r = button.R()
	modbutton, _ = display.AllocImage(r, display.ScreenImage().Pix(), false, draw.Notacolor)
	modbutton.Draw(r, tagcolors[frame.ColBack], nil, r.Min)
	r.Max.X -= display.ScaleSize(ButtonBorder)
	modbutton.Border(r, display.ScaleSize(ButtonBorder), tagcolors[frame.ColBord], image.Point{})
	r = r.Inset(display.ScaleSize(ButtonBorder))
	tmp, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)
	modbutton.Draw(r, tmp, nil, image.Point{})

	r = button.R()
	colbutton, _ = display.AllocImage(r, display.ScreenImage().Pix(), false, draw.Purpleblue)

	but2col, _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xAA0000FF)
	but3col, _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x006600FF)

}

func ismtpt(filename string) bool {
	m := *mtpt
	if m == "" {
		return false
	}
	s := path.Clean(filename)
	return strings.HasPrefix(s, m) && (m[len(m)-1] == '/' || len(s) == len(m) || s[len(m)] == '/')
}

func mousethread(display draw.Display) {
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
			display.ScreenImage().Draw(display.ScreenImage().R(), display.White(), nil, image.Point{})
			iconinit(display)
			ScrlResize(display)
			row.Resize(display.ScreenImage().R())
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
			if *swapScrollButtons {
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
				// This may replicate work done elsewhere.
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

func keyboardthread(display draw.Display) {
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
					// In a set of zeroxes, the last typed-in body becomes the curtext.
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

func waitthread(ctx context.Context) {
	// There is a race between process exiting and our finding out it was ever created.
	// This structure keeps a list of processes that have exited we haven't heard of.
	exited := make(map[int]ProcessState)

	Freecmd := func(c *Command) {
		if c != nil {
			if c.iseditcommand {
				cedit <- 0
			}
			mnt.DecRef(c.md) // mnt.Add in fsysmount
		}
	}
	for {
		select {
		case <-ctx.Done():
			return

		case err := <-cerr:
			row.lk.Lock()
			warning(nil, "%s", err)
			row.display.Flush()
			row.lk.Unlock()

		case cmd := <-ckill:
			found := false
			for _, c := range command {
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
			var (
				i int
				c *Command
			)
			pid := w.Pid()
			for i, c = range command {
				if c.pid == pid {
					command = append(command[:i], command[i+1:]...)
					break
				}
			}
			row.lk.Lock()
			t := &row.tag
			t.Commit()
			if c == nil {
				// command exited before we had a chance to add it to command list
				exited[pid] = w
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
			Freecmd(c)

		case c := <-ccommand:
			// has this command already exited?
			if p, ok := exited[c.pid]; ok {
				if msg := p.String(); msg != "" {
					warning(c.md, "%s\n", msg)
				}
				delete(exited, c.pid)
				Freecmd(c)
				break
			}
			command = append(command, c)
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
func xfidallocthread(ctx context.Context, d draw.Display) {
	xfree := (*Xfid)(nil)
	for {
		select {
		case <-ctx.Done():
			return
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
	for _, c := range command {
		c.proc.Kill()
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
