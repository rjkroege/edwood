package main

import (
	"image"
	"log"
	"os"
	"strconv"

	"9fans.net/go/plumb"
	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/frame"
)

// TODO(rjk): Document what each of these are.
type globals struct {
	globalincref bool

	seq       int  // undo/redo sequence across all file.OEBs
	maxtab    uint // size of a tab, in units of the '0' character
	tabexpand bool // defines whether to expand tab to spaces

	tagfont     string
	mouse       *draw.Mouse
	mousectl    *draw.Mousectl
	keyboardctl *draw.Keyboardctl

	modbutton draw.Image
	colbutton draw.Image
	button    draw.Image
	but2col   draw.Image
	but3col   draw.Image

	//	boxcursor Cursor
	row Row

	seltext   *Text
	argtext   *Text
	mousetext *Text // global because Text.Close needs to clear it
	typetext  *Text // global because Text.Close needs to clear it
	barttext  *Text // shared between mousethread and keyboardthread

	activewin  *Window
	activecol  *Column
	snarfbuf     []byte
	snarfContext *SelectionContext
	home       string
	acmeshell  string
	tagcolors  [frame.NumColours]draw.Image
	textcolors [frame.NumColours]draw.Image
	wdir       string
	editing    int

	cplumb     chan *plumb.Message
	cwait      chan ProcessState
	ccommand   chan *Command
	ckill      chan string
	cxfidalloc chan *Xfid
	cxfidfree  chan *Xfid
	cnewwindow chan *Window
	cexit      chan struct{}
	csignal    chan os.Signal
	cerr       chan error
	cedit      chan int
	cwarn      chan uint

	editoutlk chan bool

	WinID int
}

// Singleton global object.
var global *globals

// Preserve existing global semantics.
// TODO(rjk): Remove this *eventually*.
func init() {
	global = makeglobals()
}

func makeglobals() *globals {
	g := &globals{
		acmeshell:  os.Getenv("acmeshell"),
		editing:    Inactive,
		editoutlk:  make(chan bool, 1),
		cwait:      make(chan ProcessState),
		ccommand:   make(chan *Command),
		ckill:      make(chan string),
		cxfidalloc: make(chan *Xfid),
		cxfidfree:  make(chan *Xfid),
		cnewwindow: make(chan *Window),
		csignal:    make(chan os.Signal, 1),
		cerr:       make(chan error),
		cedit:      make(chan int),
		cexit:      make(chan struct{}),
		cwarn:      make(chan uint),
	}

	if home, err := os.UserHomeDir(); err == nil {
		g.home = home
	} else {
		log.Fatalf("could not get user home directory: %v", err)
	}

	if pwd, err := os.Getwd(); err == nil {
		g.wdir = pwd
	} else {
		log.Fatalf("could not get working directory: %v", err)
	}

	p := os.Getenv("tabstop")
	if p != "" {
		mt, _ := strconv.ParseInt(p, 10, 32)
		g.maxtab = uint(mt)
	}
	if g.maxtab == 0 {
		g.maxtab = 4
	}

	b := os.Getenv("tabexpand")
	if b != "" {
		te, _ := strconv.ParseBool(b)
		g.tabexpand = te
	} else {
		g.tabexpand = false
	}

	return g
}

// TODO(rjk): Can separate this out even better.
func (g *globals) iconinit(display draw.Display) {
	if g.tagcolors[frame.ColBack] == nil {
		g.tagcolors[frame.ColBack] = display.AllocImageMix(draw.Palebluegreen, draw.White)
		g.tagcolors[frame.ColHigh], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palegreygreen)
		g.tagcolors[frame.ColBord], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Purpleblue)
		g.tagcolors[frame.ColText] = display.Black()
		g.tagcolors[frame.ColHText] = display.Black()
		g.textcolors[frame.ColBack] = display.AllocImageMix(draw.Paleyellow, draw.White)
		g.textcolors[frame.ColHigh], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Darkyellow)
		g.textcolors[frame.ColBord], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Yellowgreen)
		g.textcolors[frame.ColText] = display.Black()
		g.textcolors[frame.ColHText] = display.Black()
	}

	// ...
	r := image.Rect(0, 0, display.ScaleSize(Scrollwid+ButtonBorder), fontget(g.tagfont, display).Height()+1)
	g.button, _ = display.AllocImage(r, display.ScreenImage().Pix(), false, draw.Notacolor)
	g.button.Draw(r, g.tagcolors[frame.ColBack], nil, r.Min)
	r.Max.X -= display.ScaleSize(ButtonBorder)
	g.button.Border(r, display.ScaleSize(ButtonBorder), g.tagcolors[frame.ColBord], image.Point{})

	r = g.button.R()
	g.modbutton, _ = display.AllocImage(r, display.ScreenImage().Pix(), false, draw.Notacolor)
	g.modbutton.Draw(r, g.tagcolors[frame.ColBack], nil, r.Min)
	r.Max.X -= display.ScaleSize(ButtonBorder)
	g.modbutton.Border(r, display.ScaleSize(ButtonBorder), g.tagcolors[frame.ColBord], image.Point{})
	r = r.Inset(display.ScaleSize(ButtonBorder))
	tmp, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)
	g.modbutton.Draw(r, tmp, nil, image.Point{})

	r = g.button.R()
	g.colbutton, _ = display.AllocImage(r, display.ScreenImage().Pix(), false, draw.Purpleblue)

	g.but2col, _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xAA0000FF)
	g.but3col, _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x006600FF)
}
