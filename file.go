package main

import (
	"crypto/sha1"
	"fmt"
	"os"
)

type File struct {
	b       Buffer
	delta   []*Undo
	epsilon []*Undo
	elogbuf *Buffer
	elog    Elog
	name    string //[]rune
	qidpath string // TODO(flux): Gross hack to use filename instead of qidpath for file uniqueness
	mtime   int64
	// dev       int
	unread    bool
	editclean bool
	seq       int
	mod       bool

	curtext *Text
	text    []*Text
	dumpid  int

	sha1 [sha1.Size]byte // Used to check if the file has changed on disk since loaded
}

type Undo struct {
	t   int
	mod bool
	seq int
	p0  int
	n   int
	buf []rune
}

func (f *File) Load(q0 int, fd *os.File) (n int, h [sha1.Size]byte, hasNulls bool, err error) {
	n, h, hasNulls, err = f.b.Load(q0, fd)
	return n, h, hasNulls, err
}

func (f *File) AddText(t *Text) *File {
	f.text = append(f.text, t)
	f.curtext = t
	return f
}

func (f *File) DelText(t *Text) {

	for i, text := range f.text {
		if text == t {
			f.text[i] = f.text[len(f.text)-1]
			f.text = f.text[:len(f.text)-1]
			if len(f.text) == 0 {
				return
			}
			if t == f.curtext {
				f.curtext = f.text[0]
			}
		}
	}
	acmeerror("can't find text in File.DelText", nil)
}

func (f *File) Insert(p0 int, s []rune) {
	if p0 > f.b.nc() {
		panic("internal error: fileinsert")
	}
	if f.seq > 0 {
		f.Uninsert(&f.delta, p0, len(s))
	}
	f.b.Insert(p0, s)
	if len(s) != 0 {
		f.mod = true
	}
}

func (f *File) Uninsert(delta *[]*Undo, q0, ns int) {
	var u Undo
	/* undo an insertion by deleting */
	u.t = Delete
	u.mod = f.mod
	u.seq = f.seq
	u.p0 = q0
	u.n = ns
	(*delta) = append(*delta, &u)
}

func (f *File) Delete(p0, p1 int) {
	if !(p0 <= p1 && p0 <= f.b.nc() && p1 <= f.b.nc()) {
		acmeerror("internal error: filedelete", nil)
	}
	if f.seq > 0 {
		f.Undelete(&f.delta, p0, p1)
	}
	f.b.Delete(p0, p1)
	if p1 > p0 {
		f.mod = true
	}
}

func (f *File) Undelete(delta *[]*Undo, p0, p1 int) {
	/* undo a deletion by inserting */
	var u Undo
	u.t = Insert
	u.mod = f.mod
	u.seq = f.seq
	u.p0 = p0
	u.n = p1 - p0
	u.buf = f.b.Read(p0, u.n)
	(*delta) = append(*delta, &u)
}

func (f *File) SetName(name string) {
	if f.seq > 0 {
		// f.UnsetName(f, &f.delta) TODO(flux): Undo
	}
	f.name = name
	f.unread = true
}

func (f *File) UnsetName(delta *[]*Undo) {
	var u Undo
	/* undo a file name change by restoring old name */
	u.t = Filename
	u.mod = f.mod
	u.seq = f.seq
	u.p0 = 0 /* unused */
	u.n = len(f.name)
	u.buf = []rune(f.name)
	(*delta) = append(*delta, &u)
}

func NewFile(filename string) *File {
	return &File{
		b:       NewBuffer(),
		delta:   []*Undo{},
		epsilon: []*Undo{},
		elog:    MakeElog(),
		name:    filename,
		//	qidpath   uint64
		//	mtime     uint64
		//	dev       int
		unread:    true,
		editclean: true,
		//	seq       int
		mod: false,

		curtext: nil,
		text:    []*Text{},
		//	ntext   int
		//	dumpid  int
	}
}

func NewTagFile() *File {

	return &File{
		b:       NewBuffer(),
		delta:   []*Undo{},
		epsilon: []*Undo{},

		elog: MakeElog(),
		name: "",
		//	qidpath   uint64
		//	mtime     uint64
		//	dev       int
		unread:    true,
		editclean: true,
		//	seq       int
		mod: false,

		//	curtext *Text
		//	text    **Text
		//	ntext   int
		//	dumpid  int
	}
}

func (f *File) RedoSeq() int {
	delta := &f.epsilon
	if len(*delta) == 0 {
		return 0
	}
	u := (*delta)[len(*delta)-1]
	return u.seq
}

func (f *File) Undo(isundo bool) (q0p, q1p int) {
	var (
		stop           int
		delta, epsilon *[]*Undo
	)
	if isundo {
		/* undo; reverse delta onto epsilon, seq decreases */
		delta = &f.delta
		epsilon = &f.epsilon
		stop = f.seq
	} else {
		/* redo; reverse epsilon onto delta, seq increases */
		delta = &f.epsilon
		epsilon = &f.delta
		stop = 0 /* don't know yet */
	}

	for len(*delta) > 0 {
		u := (*delta)[len(*delta)-1]
		if isundo {
			if u.seq < stop {
				f.seq = u.seq
				return
			}
		} else {
			if stop == 0 {
				stop = u.seq
			}
			if u.seq > stop {
				return
			}
		}
		switch u.t {
		default:
			panic(fmt.Sprintf("undo: 0x%x\n", u.t))
		case Delete:
			f.seq = u.seq
			f.Undelete(epsilon, u.p0, u.p0+u.n)
			f.mod = u.mod
			f.b.Delete(u.p0, u.p0+u.n)
			for _, text := range f.text {
				text.Delete(u.p0, u.p0+u.n, false)
			}
			q0p = u.p0
			q1p = u.p0

		case Insert:
			f.seq = u.seq
			f.Uninsert(epsilon, u.p0, u.n)
			f.mod = u.mod
			f.b.Insert(u.p0, u.buf)
			for _, text := range f.text {
				text.Insert(u.p0, u.buf, false)
			}
			q0p = u.p0
			q1p = u.p0 + u.n

		case Filename:
			f.seq = u.seq
			f.UnsetName(epsilon)
			f.mod = u.mod
			if u.n == 0 {
				f.name = ""
			} else {
				f.name = string(u.buf)
			}
			break
		}
		(*delta) = (*delta)[0 : len(*delta)-1]
	}
	if isundo {
		f.seq = 0
	}
	return q0p, q1p
}

func (f *File) Reset() {
	f.delta = f.delta[0:0]
	f.epsilon = f.epsilon[0:0]
	f.seq = 0
}

func (f *File) Close() {
	f.b.Close()
	elogclose(f)
}

func (f *File) Mark() {
	f.epsilon = f.epsilon[0:0]
	f.seq = seq
}
