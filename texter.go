package main

// Texter abstracts the buffering side of Text, allowing testing of Elog Apply
// TODO(flux): This is probably lame and will get re-done when I understand
// how Text stores its text.
type Texter interface {
	Constrain(q0, q1 uint) (p0, p1 uint)
	Delete(q0, q1 uint, tofile bool)
	Insert(q0 uint, r []rune, tofile bool)
	Q0() uint // Selection start
	SetQ0(uint)
	Q1() uint // End of selelection
	SetQ1(uint)
}
