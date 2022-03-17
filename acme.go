package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"9fans.net/go/plumb"
	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/dumpfile"
)

var (
	command []*Command

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

	// TODO(rjk): Unlimited concurrency please.
	runtime.GOMAXPROCS(7)

	var (
		ncol     int
		loadfile string
	)
	flag.IntVar(&ncol, "c", 2, "Number of columns at startup")
	flag.StringVar(&loadfile, "l", "", "Load state from file generated with Dump command")
	flag.Parse()

	startProfiler()

	// Implicit to preserve existing semantics.
	// TODO(rjk): Do this here.
	// global = makeglobals()
	g := global

	// TODO(rjk): Push this code into a separate function.
	var dump *dumpfile.Content

	if loadfile != "" {
		d, err := dumpfile.Load(loadfile) // Overrides fonts selected up to here.
		if err != nil {
			// Maybe it's in legacy format. Try that too.
			d, err = dumpfile.LoadLegacy(loadfile, g.home)
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

	g.tagfont = *varfontflag
	os.Setenv("font", *varfontflag)

	draw.Main(func(dd *draw.Device) {
		display, err := dd.NewDisplay(nil, *varfontflag, "edwood", *winsize)
		if err != nil {
			log.Fatalf("can't open display: %v\n", err)
		}
		if err := display.Attach(draw.Refnone); err != nil {
			panic("failed to attach to window")
		}
		display.ScreenImage().Draw(display.ScreenImage().R(), display.White(), nil, image.Point{})

		g.mousectl = display.InitMouse()
		g.mouse = &g.mousectl.Mouse
		g.keyboardctl = display.InitKeyboard()

		g.iconinit(display)

		startplumbing()
		fs := fsysinit()

		const WindowsPerCol = 6

		g.row.Init(display.ScreenImage().R(), display)
		if loadfile == "" || g.row.Load(dump, loadfile, true) != nil {
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
				g.row.Add(nil, -1)
			}
			rightmostcol := g.row.col[len(g.row.col)-1]
			if len(files) == 0 {
				readfile(g.row.col[len(g.row.col)-1], g.wdir)
			} else {
				for i, filename := range files {
					// guide  always goes in the rightmost column
					if filepath.Base(filename) == "guide" || i/WindowsPerCol >= len(g.row.col) {
						readfile(rightmostcol, filename)
					} else {
						readfile(g.row.col[i/WindowsPerCol], filename)
					}
				}
			}
		}
		display.Flush()

		// After row is initialized
		// TODO(rjk): put the globals *in* the ctx?
		ctx := context.Background()
		go mousethread(g, display)
		go keyboardthread(g, display)
		go waitthread(g, ctx)
		go newwindowthread(g)
		go xfidallocthread(g, ctx, display)

		signal.Ignore(ignoreSignals...)
		signal.Notify(g.csignal, hangupSignals...)

		select {
		case <-g.cexit:
			// Do nothing.
		case <-g.csignal:
			g.row.lk.Lock()
			g.row.Dump("")
			g.row.lk.Unlock()
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
	w.Resize(w.r, false, true)
	w.body.ScrDraw(w.body.fr.GetFrameFillStatus().Nchars)
	w.tag.SetSelect(w.tag.file.Nr(), w.tag.file.Nr())
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

func ismtpt(filename string) bool {
	m := *mtpt
	if m == "" {
		return false
	}
	s := path.Clean(filename)
	return strings.HasPrefix(s, m) && (m[len(m)-1] == '/' || len(s) == len(m) || s[len(m)] == '/')
}

func mousethread(g *globals, display draw.Display) {
	// TODO(rjk): Do we need this?
	runtime.LockOSThread()

	for {
		g.row.lk.Lock()
		flushwarnings()
		g.row.lk.Unlock()
		display.Flush()
		select {
		case <-g.mousectl.Resize:
			if err := display.Attach(draw.Refnone); err != nil {
				panic("failed to attach to window")
			}
			display.ScreenImage().Draw(display.ScreenImage().R(), display.White(), nil, image.Point{})
			// TODO(rjk): We appear to have already done this.
			g.iconinit(display)
			ScrlResize(display)
			g.row.Resize(display.ScreenImage().R())
		case g.mousectl.Mouse = <-g.mousectl.C:
			MovedMouse(g, g.mousectl.Mouse)
		case <-g.cwarn:
			// Do nothing
		case pm := <-g.cplumb:
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

func MovedMouse(g *globals, m draw.Mouse) {
	g.row.lk.Lock()
	defer g.row.lk.Unlock()

	t := g.row.Which(m.Point)

	if t != g.mousetext && t != nil && t.w != nil &&
		(g.mousetext == nil || g.mousetext.w == nil || t.w.id != g.mousetext.w.id) {
		xfidlog(t.w, "focus")
	}

	if t != g.mousetext && g.mousetext != nil && g.mousetext.w != nil {
		g.mousetext.w.Lock('M')
		g.mousetext.eq0 = ^0
		g.mousetext.w.Commit(g.mousetext)
		g.mousetext.w.Unlock()
	}
	g.mousetext = t
	if t == nil {
		return
	}
	w := t.w
	if m.Buttons == 0 {
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
	g.barttext = t
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
				g.row.DragCol(t.col, but)
			case Tag:
				t.col.DragWin(t.w, but)
				if t.w != nil {
					g.barttext = &t.w.body
				}
			}
			if t.col != nil {
				g.activecol = t.col
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
			g.argtext = t
			g.seltext = t
			if t.col != nil {
				g.activecol = t.col // button 1 only
			}
			if t.w != nil && t == &t.w.body {
				g.activewin = t.w
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

func keyboardthread(g *globals, display draw.Display) {
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
		case r := <-g.keyboardctl.C:
			for {
				typetext = g.row.Type(r, g.mouse.Point)
				t = typetext
				if t != nil && t.col != nil && !(r == draw.KeyDown || r == draw.KeyLeft || r == draw.KeyRight) { // scrolling doesn't change activecol
					g.activecol = t.col
				}
				if t != nil && t.w != nil {
					// In a set of zeroxes, the last typed-in body becomes the currobserver.
					t.w.body.file.SetCurObserver(&t.w.body)
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
				case r = <-g.keyboardctl.C:
					continue
				default:
					display.Flush()
				}
				break
			}
		}
	}

}

func waitthread(g *globals, ctx context.Context) {
	// There is a race between process exiting and our finding out it was ever created.
	// This structure keeps a list of processes that have exited we haven't heard of.
	exited := make(map[int]ProcessState)

	Freecmd := func(c *Command) {
		if c != nil {
			if c.iseditcommand {
				g.cedit <- 0
			}
			mnt.DecRef(c.md) // mnt.Add in fsysmount
		}
	}
	for {
		select {
		case <-ctx.Done():
			return

		case err := <-g.cerr:
			g.row.lk.Lock()
			warning(nil, "%s", err)
			g.row.display.Flush()
			g.row.lk.Unlock()

		case cmd := <-g.ckill:
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

		case w := <-g.cwait:
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
			g.row.lk.Lock()
			t := &g.row.tag
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
				g.row.display.Flush()
			}
			g.row.lk.Unlock()
			Freecmd(c)

		case c := <-g.ccommand:
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
			g.row.lk.Lock()
			t := &g.row.tag
			t.Commit()
			t.Insert(0, []rune(c.name), true)
			t.SetSelect(0, 0)
			g.row.display.Flush()
			g.row.lk.Unlock()
		}
	}
}

// maintain a linked list of Xfid
// TODO(flux): It would be more idiomatic to prep one up front, and block on sending
// it instead of using a send and a receive to get one.
// Frankly, it would be more idiomatic to let the GC take care of them,
// though that would require an exit signal in xfidctl.
func xfidallocthread(g *globals, ctx context.Context, d draw.Display) {
	xfree := (*Xfid)(nil)
	for {
		select {
		case <-ctx.Done():
			return
		case <-g.cxfidalloc:
			x := xfree
			if x != nil {
				xfree = x.next
			} else {
				x = &Xfid{}
				x.c = make(chan func(*Xfid))
				go xfidctl(x, d)
			}
			g.cxfidalloc <- x
		case x := <-g.cxfidfree:
			x.next = xfree
			xfree = x
		}
	}

}

func newwindowthread(g *globals) {
	var w *Window

	for {
		// only fsysproc is talking to us, so synchronization is trivial
		<-g.cnewwindow

		// TODO(rjk): Should this be in a row lock?
		w = makenewwindow(nil)
		xfidlog(w, "new")
		g.cnewwindow <- w
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
		global.cerr <- fmt.Errorf(string(data))
	}
	return
}

// Close exists only to satisfy io.WriteCloser interface.
func (w errorWriter) Close() error {
	return nil
}

const MAXSNARF = 10 * 1024

func acmeputsnarf() {
	global.row.display.WriteSnarf(global.snarfbuf)
}

func acmegetsnarf() {
	// log.Println("acmegetsnarf")
	// defer log.Println("end acmegetsnarf")
	// TODO(rjk): use the non-blocking interface on platforms that have one
	// for big snarfs
	b := make([]byte, MAXSNARF)

	n, sz, err := global.row.display.ReadSnarf(b)
	//	log.Println(n, sz)
	if err != nil {
		warning(nil, "can't readsnarf %v", err)
		return
	}
	if n < len(b) && n == sz {
		global.snarfbuf = b[0:n]
		return
	}

	b = make([]byte, sz)
	n, _, err = global.row.display.ReadSnarf(b)
	//	log.Println("second call", n, sz)
	if err != nil {
		warning(nil, "can't readsnarf %v", err)
		return
	}

	// Trim it: it might have shortened.
	global.snarfbuf = b[0:n]
}
