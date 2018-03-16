package main

// Texter abstracts the buffering side of Text, allowing testing of Elog Apply
// TODO(flux): This is probably lame and will get re-done when I understand
// how Text stores its text.
type Texter interface {
	Constrain(q0, q1 int) (p0, p1 int)
	Delete(q0, q1 int, tofile bool)
	Insert(q0 int, r []rune, tofile bool)
	Q0() int // Selection start
	SetQ0(int)
	Q1() int // End of selelection
	SetQ1(int)
	nc() int
	Read(q, n int) []rune
}
