package main

import (
	"crypto/sha1"
	"os"
)

type File struct {
	b       Buffer
	delta   Buffer
	epsilon Buffer
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

func (f *File) Load(q0 uint, fd *os.File) (n uint, h [sha1.Size]byte, hasNulls bool, err error) {
	n, h, hasNulls, err = f.b.Load(q0, fd)
	return n, h, hasNulls, err
}

func (f *File) AddText(t *Text) *File {
	f.text = append(f.text, t)
	f.curtext = t
	return f
}

func (f *File) DelText(t *Text) {
	Unimpl()
}

func (f *File) Insert(p0 uint, s []rune) {
	if p0 > f.b.nc() {
		panic("internal error: fileinsert")
	}
	if f.seq > 0 {
		// f.Uninsert(&f.delta, p0, len(s))  TODO(flux): Here we start dealing with Undo operations
	}
	f.b.Insert(p0, s)
	if len(s) != 0 {
		f.mod = true
	}
}

func (f *File) Uninsert(delta *Buffer, q0, ns uint) {
	Unimpl()
}

func (f *File) Delete(p0, p1 uint) {
	Unimpl()
}

func (f *File) Undelete(delta *Buffer, p0, p1 uint) {
	Unimpl()
}

func (f *File) SetName(name string) {
	if f.seq > 0 {
		// f.UnsetName(f, &f.delta) TODO(flux): Undo
	}
	f.name = name
	f.unread = true
}

func (f *File) UnsetName(delta *Buffer) {
	Unimpl()
}

func NewFile(filename string) *File {
	return &File{
		b: NewBuffer(),
		/*	delta     Buffer
			epsilon   Buffer
		*/
		elog: MakeElog(),
		name: filename,
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
		b: NewBuffer(),
		/*	delta     Buffer
			epsilon   Buffer
		*/
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

func (f *File) RedoSeq() uint {
	Unimpl()
	return 0
}

func (f *File) Undo(isundo bool, q0p, q1p *uint) {
	Unimpl()
}

func (f *File) Reset() {
	Unimpl()

}

func (f *File) Close() {
	Unimpl()

}

func (f *File) Mark() {
	if f.epsilon.nc() != 0 {
		f.epsilon.Delete(0, f.epsilon.nc())
	}
	f.seq = seq
}
