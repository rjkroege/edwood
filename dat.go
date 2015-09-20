package main

import (
	"9fans.net/go/draw"
	"bytes"
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"os/user"
	"sync"
	"unicode/utf8"
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

	Blockincr = 256
	MaxBlock  = 8 * 1024
	NRange    = 10
	Infinity  = 0x7FFFFFFF

	STACK = 65536

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
)

var (
	blist *Block

	globalincref bool
	seq          uint
	maxtab       uint /*size of a tab, in units of the '0' character */

	display     *draw.Display
	screen      *draw.Image
	font        *draw.Font
	mouse       *draw.Mouse
	mousectl    *draw.Mousectl
	keyboardctl *draw.Keyboardctl

	reffont   Reffont
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

	bartflag          int
	swapscrollbuttons int
	activewin         *Window
	activecol         *Column
	snarfbuf          Buffer
	nullrect          image.Rectangle
	fsyspid           int
	cputype           string
	objtype           string
	acmeshell         string
	tagcols           [NCOL]*draw.Image
	textcols          [NCOL]*draw.Image
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
)

type Range struct {
	q0, q1 int
}

type Block struct {
	addr uint   // disk address in bytes
	n    uint   // number of used runes in block
	next *Block // pointer to next in free list
}

type Disk struct {
	fd   *os.File
	addr uint
	free [MaxBlock/Blockincr + 1]*Block
}

func NewDisk() *Disk {
	d := new(Disk)
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	tmp, err := ioutil.TempFile("/tmp", fmt.Sprintf("X%d.%.4sacme", os.Getpid(), u.Username))
	if err != nil {
		panic(err)
	}
	d.fd = tmp
	return d
}

func ntosize(n uint) (uint, uint) {
	if n > MaxBlock {
		panic("internal error: ntosize")
	}
	size := n
	if size&(Blockincr-1) != 0 {
		size += Blockincr - (size & (Blockincr - 1))
	}

	// last bucket holds blocks of exactly Maxblock
	ip := size / Blockincr
	return size, ip
}

func (d *Disk) NewBlock(n uint) *Block {
	size, i := ntosize(n)
	b := d.free[i]
	if b != nil {
		d.free[i] = b.next
	} else {
		if blist == nil {
			bl := new(Block)
			blist = bl
			for j := 0; j < 100-1; j++ {
				bl.next = new(Block)
				bl = bl.next
			}
		}
		b = blist
		blist = b.next
		b.addr = d.addr
		d.addr += size
	}
	b.n = n
	return b
}

func (d *Disk) Release(b *Block) {
	_, i := ntosize(b.n)
	b.next = d.free[i]
	d.free[i] = b
}

func (d *Disk) Read(b *Block, r []rune, n uint) {
	if n > b.n {
		panic("internal error: disk.Read")
	}
	// this is a simplified way of checking that b.n < MaxBlock
	_, _ = ntosize(b.n)
	buf := make([]byte, n)
	if m, err := d.fd.ReadAt(buf, int64(b.addr)); err != nil {
		panic(err)
	} else if m != len(r) {
		panic("read error from temp file")
	}
	copy(r, bytes.Runes(buf))
}

func (d *Disk) Write(bp **Block, r []rune, n uint) {
	bl := *bp
	size, _ := ntosize(bl.n)
	nsize, _ := ntosize(n)
	if size != nsize {
		d.Release(bl)
		bl = d.NewBlock(n)
		*bp = bl
	}
	if m, err := d.fd.WriteAt([]byte(string(r)), int64(bl.addr)); err != nil {
		panic(err)
	} else if m != len(r) {
		panic("write error to temp file")
	}
	bl.n = n
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

type Reffont struct {
	ref Ref
	f   *draw.Font
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
