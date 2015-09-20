package main

type File struct {
	b         Buffer
	delta     Buffer
	epsilon   Buffer
	elogbuf   *Buffer
	elog      Elog
	name      []rune
	qidpath   uint64
	mtime     uint64
	dev       int
	unread    bool
	editclean bool
	seq       int
	mod       int

	curtext *Text
	text    **Text
	ntext   int
	dumpid  int
}

func (f *File) AddText(t *Text) *File {
	return nil
}

func (f *File) DelText(t *Text) {

}

func (f *File) Insert(q0 uint, s []rune, ns uint) {

}

func (f *File) Uninsert(delta *Buffer, q0, ns uint) {

}

func (f *File) Delete(p0, p1 uint) {

}

func (f *File) Undelete(delta *Buffer, p0, p1 uint) {

}

func (f *File) SetName(name string, n int) {

}

func (f *File) UnsetName(delta *Buffer) {

}

func (f *File) Load(p0 uint, fd int, nulls *int) uint {
	return 0
}

func (f *File) RedoSeq() uint {
	return 0
}

func (f *File) Undo(isundo bool, q0p, q1p *uint) {

}

func (f *File) Reset() {

}

func (f *File) Close() {

}

func (f *File) Mark() {

}
