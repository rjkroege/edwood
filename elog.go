package main

type ElogType byte

const (
	DeleteType ElogType = iota
	InsertType
	FilenameType
)

type Elog struct {
	t  ElogType // Delete, Insert, Filename
	q0 uint     // location of change (unused in f)
	nd uint     // number of deleted characters
	nr uint     // number of runes in string or filename
	r  []rune
}

func (e *Elog) Term(f *File) {

}

func (e *Elog) Close(f *File) {

}

func (e *Elog) Insert(f *File, q0 uint, r []rune) {

}

func (e *Elog) Delete(f *File, q0, q1 int) {

}

func (e *Elog) Apply(f *File) {

}

func (e *Elog) Replace(f *File, q0, q1 int, r []rune) {

}
