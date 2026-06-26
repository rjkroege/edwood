package main

import (
	"image"
	"log"
	"os"
	"strconv"

	"9fans.net/go/plumb"
	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/frame"
	"github.com/rjkroege/edwood/theme"
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
	snarfbuf   []byte
	home       string
	acmeshell  string
	tagcolors  [frame.NumColours]draw.Image
	textcolors [frame.NumColours]draw.Image
	palette    theme.Palette
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

// allocColor allocates a single-pixel replicating image for a ColorSpec.
// If cs.Mix is non-zero, AllocImageMix is used; otherwise AllocImage.
func (g *globals) allocColor(display draw.Display, cs theme.ColorSpec) draw.Image {
	if cs.Mix != 0 {
		return display.AllocImageMix(cs.Color, cs.Mix)
	}
	img, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, cs.Color)
	return img
}

// applyMode allocates all colour images from g.palette and stores them in
// g.tagcolors, g.textcolors, and the chrome button images.
// No mode conditionals — the palette already encodes the chosen theme.
func (g *globals) applyMode(display draw.Display) {
	p := g.palette
	g.tagcolors[frame.ColBack]  = g.allocColor(display, p[theme.TagBack])
	g.tagcolors[frame.ColHigh]  = g.allocColor(display, p[theme.TagHigh])
	g.tagcolors[frame.ColBord]  = g.allocColor(display, p[theme.TagBord])
	g.tagcolors[frame.ColText]  = g.allocColor(display, p[theme.TagText])
	g.tagcolors[frame.ColHText] = g.allocColor(display, p[theme.TagHText])
	g.tagcolors[frame.ColTick]  = g.allocColor(display, p[theme.TagTick])

	g.textcolors[frame.ColBack]  = g.allocColor(display, p[theme.TextBack])
	g.textcolors[frame.ColHigh]  = g.allocColor(display, p[theme.TextHigh])
	g.textcolors[frame.ColBord]  = g.allocColor(display, p[theme.TextBord])
	g.textcolors[frame.ColText]  = g.allocColor(display, p[theme.TextText])
	g.textcolors[frame.ColHText] = g.allocColor(display, p[theme.TextHText])
	g.textcolors[frame.ColTick]  = g.allocColor(display, p[theme.TextTick])
}

// TODO(rjk): Can separate this out even better.
func (g *globals) iconinit(display draw.Display) {
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
	tmp := g.allocColor(display, g.palette[theme.ChromeModButton])
	g.modbutton.Draw(r, tmp, nil, image.Point{})

	r = g.button.R()
	g.colbutton, _ = display.AllocImage(r, display.ScreenImage().Pix(), false, g.palette[theme.ChromeColButton].Color)

	g.but2col = g.allocColor(display, g.palette[theme.ChromeBut2])
	g.but3col = g.allocColor(display, g.palette[theme.ChromeBut3])
}
