package main

import (
	"fmt"
	"image"
	"runtime/debug"
	"strings"
	"sync"
	"unicode/utf8"

	"9fans.net/go/draw"
	"github.com/paul-lalonde/acme/frame"
)

const (
	Qdir int = iota
	Qacme
	Qcons
	Qconsctl
	Qdraw
	Qeditout
	Qindex
	Qlabel
	Qnew
	QWaddr
	QWbody
	QWctl
	QWdata
	QWeditout
	QWerrors
	QWevent
	QWrdsel
	QWwrsel
	QWtag
	QWxdata
	QMAX

	NRange = 10
	//	Infinity  = 0x7FFFFFFF

	//	STACK = 65536
	EVENTSIZE = 256

	Empty    = 0
	Null     = '-'
	Delete   = 'd'
	Insert   = 'i'
	Replace  = 'r'
	Filename = 'f'

	Inactive   = 0
	Inserting  = 1
	Collecting = 2

	NCOL = 5

	// Always apply display scalesize to these.
	Border       = 2
	ButtonBorder = 2
	Scrollwid    = 12
	Scrollgap    = 8

	KF             = 0xF000 // Start of private unicode space
	Kscrolloneup   = KF | 0x20
	Kscrollonedown = KF | 0x21
)

var (
	globalincref bool
	seq          int
	maxtab       uint /*size of a tab, in units of the '0' character */

	display     *draw.Display
	screen      *draw.Image
	tagfont     *draw.Font
	mouse       *draw.Mouse
	mousectl    *draw.Mousectl
	keyboardctl *draw.Keyboardctl

	reffont   *draw.Font
	reffonts  [2]*draw.Font
	modbutton *draw.Image
	colbutton *draw.Image
	button    *draw.Image
	but2col   *draw.Image
	but3col   *draw.Image

	//	boxcursor Cursor
	row Row

	timerpid  int
	disk      *Disk
	seltext   *Text
	argtext   *Text
	mousetext *Text
	typetext  *Text
	barttext  *Text

	bartflag          bool
	swapscrollbuttons bool
	activewin         *Window
	activecol         *Column
	snarfbuf          Buffer
	nullrect          image.Rectangle
	fsyspid           int
	cputype           string
	objtype           string
	home              string
	acmeshell         string
	tagcolors         [frame.NumColours]*draw.Image
	textcolors        [frame.NumColours]*draw.Image
	wdir              string
	editing           bool
	erroutfd          int
	messagesize       int
	globalautoindent  bool
	dodollarsigns     bool
	mtpt              string

	//	cplumb chan *Plumbmsg
	//	cwait chan Waitmsg
	ccommand   chan Command
	ckill      chan []rune
	cxfidalloc chan *Xfid
	cxfidfree  chan *Xfid
	cnewwindow chan chan interface{}
	mouseexit0 chan int
	mouseexit1 chan int
	cexit      chan int
	cerr       chan string
	cedit      chan int
	cwarn      chan uint

	editoutlk *sync.Mutex

	WinId int = 0
)

type Range struct {
	q0, q1 int
}

type Command struct {
	pid           int
	name          []rune
	text          string
	av            []string
	iseditcommand bool
	md            *MntDir
	next          *Command
}

type DirTab struct {
	name string
	t    byte
	qid  uint
	perm uint
}

type MntDir struct {
	id    int
	ref   int
	dir   string
	ndir  int
	next  *MntDir
	nincl int
	incl  []string
}

type Fid struct {
	fid  int
	busy int
	open int
	//qid Qid
	w      *Window
	dir    *DirTab
	next   *Fid
	mntdir *MntDir
	nrpart int
	rpart  [utf8.UTFMax]byte
}

type Xfid struct {
	arg interface{}
	//	fcall Fcall
	next    *Xfid
	c       chan func(*Xfid)
	f       *Fid
	buf     []byte
	flushed bool
}

type RangeSet [NRange]Range

type Dirlist struct {
	r   []rune
	nr  int
	wid int
}

type Expand struct {
	q0    uint
	q1    uint
	name  string
	bname string
	jump  int
	at    *Text
	ar    []rune
	agetc func(interface{}, uint) int
	a0    int
	a1    int
}

type Ref int

func (r *Ref) Inc() {
	*r++
}

func (r *Ref) Dec() {
	*r--
}

func Unimpl() {
	stack := strings.Split(string(debug.Stack()), "\n")
	for i, l := range stack {
		if l == "main.Unimpl()" {
			fmt.Printf("Unimplemented: %v: %v\n", stack[i+2], strings.TrimLeft(stack[i+3], " \t"))
			break
		}
	}
}
