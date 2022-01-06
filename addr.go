package main

import (
	"strings"

	"github.com/rjkroege/edwood/sam"
)

// These constants indicates the direction of regular expresssion search.
// TODO(fhs): Introduce a new type for these constants.
const (
	None = iota
	Fore = '+' // Forward
	Back = '-' // Backward
)

const (
	Char = iota
	Line
)

// Return if r is valid character in an address
func isaddrc(r rune) bool {
	return strings.ContainsRune("0123456789+-/$.#,;?", r)
}

// quite hard: could be almost anything but white space, but we are a little conservative,
// aiming for regular expressions of alphanumerics and no white space
func isregexc(r rune) bool {
	if r == 0 {
		return false
	}
	if isalnum(r) {
		return true
	}
	if strings.ContainsRune("^+-.*?#,;[]()$", r) {
		return true
	}
	return false
}

// nlcounttopos starts at q0 and advances nl lines,
// being careful not to walk past the end of the text,
// and then nr chars, being careful not to walk past
// the end of the current line.
// It returns the final position in runes.
func nlcounttopos(t sam.Texter, q0 int, nl int, nr int) int {
	for nl > 0 && q0 < t.Nc() {
		if t.ReadC(q0) == '\n' {
			nl--
		}
		q0++
	}
	if nl > 0 {
		return q0
	}
	for nr > 0 && q0 < t.Nc() && t.ReadC(q0) != '\n' {
		q0++
		nr--
	}
	return q0
}

func number(showerr bool, t sam.Texter, r Range, line int, dir int, size int) (Range, bool) {
	var q0, q1 int

	if size == Char {
		if dir == Fore {
			line = r.q1 + line
		} else {
			if dir == Back {
				if r.q0 == 0 && line > 0 {
					r.q0 = t.Nc()
				}
				line = r.q0 - line
			}
		}
		if line < 0 || line > t.Nc() {
			goto Rescue
		}
		return Range{line, line}, true
	}
	q0 = r.q0
	q1 = r.q1
	switch dir {
	case None:
		q0 = 0
		q1 = 0
		for line > 0 && q1 < t.Nc() {
			if t.ReadC(q1) == '\n' || q1 == t.Nc() {
				line--
				if line > 0 {
					q0 = q1 + 1
				}
			}
			q1++
		}
		if line == 1 && q1 == t.Nc() { // 6 goes to end of 5-line file
			break
		}
		if line > 0 {
			goto Rescue
		}
	case Fore:
		if q1 > 0 {
			for q1 < t.Nc() && t.ReadC(q1-1) != '\n' {
				q1++
			}
		}
		q0 = q1
		for line > 0 && q1 < t.Nc() {
			if t.ReadC(q1) == '\n' || q1 == t.Nc() {
				line--
				if line > 0 {
					q0 = q1 + 1
				}
			}
			q1++
		}
		if line == 1 && q1 == t.Nc() { // 6 goes to end of 5-line file
			break
		}
		if line > 0 {
			goto Rescue
		}
	case Back:
		if q0 < t.Nc() {
			for q0 > 0 && t.ReadC(q0-1) != '\n' {
				q0--
			}
		}
		q1 = q0
		for line > 0 && q0 > 0 {
			if t.ReadC(q0-1) == '\n' {
				line--
				if line >= 0 {
					q1 = q0
				}
			}
			q0--
		}
		// :1-1 is :0 = #0, but :1-2 is an error
		if line > 1 {
			goto Rescue
		}
		for q0 > 0 && t.ReadC(q0-1) != '\n' {
			q0--
		}
	}
	return Range{q0, q1}, true

Rescue:
	if showerr {
		warning(nil, "address out of range\n")
	}
	return r, false
}

var pattern *AcmeRegexp

// acmeregexp searches for regular expression pattern pat in text t.
// If pat is empty, it uses the pattern used in the previous invocation of this function.
// Dir indicates the direction of the search: forward or backward.
// R sets the text position where search begins.
// Lim sets the text position where search ends for forward search
// (set range to {-1, -1} for no limit).
// Warnings will be shown to user if showerr is true.
// It returns the match and whether a match was found.
func acmeregexp(showerr bool, t sam.Texter, lim Range, r Range, pat string, dir int) (retr Range, found bool) {
	if len(pat) == 0 && pattern == nil {
		if showerr {
			warning(nil, "no previous regular expression\n")
		}
		return r, false
	}
	if len(pat) > 0 {
		var err error
		pattern, err = rxcompile(pat)
		if err != nil {
			return r, false
		}
	}

	var sel RangeSet
	if dir == Back {
		sel = pattern.rxbexecute(t, r.q0, 1)
	} else {
		q := -1
		if lim.q0 >= 0 {
			q = lim.q1
		}
		sels := pattern.rxexecute(t, nil, r.q1, q, 1)
		if len(sels) > 0 {
			sel = sels[0]
		} else {
			sel = nil
		}
	}
	if len(sel) == 0 {
		if showerr {
			warning(nil, "no match for regexp\n")
		}
		return Range{-1, -1}, false
	}
	return sel[0], true
}

// address parses an address for text t, where getc takes a closure over
// the address expression and returns the qth rune in the address
// (q0 <= q < q1). If eval is true, the address is also evaluated.
// Lim sets the limits of regular expression search.
// Warnings will be shown to user if showerr is true.
// It returns the updated address range (initially set to ar), whether
// the evaluation was successful, and the position q in the address
// where parsing was stopped.
func address(showerr bool, t sam.Texter, lim Range, ar Range, q0 int, q1 int, getc func(q int) rune, eval bool) (r Range, evalp bool, qp int) {
	var (
		n         int
		prevc, nc rune
		nr        Range
	)
	evalp = eval
	r = ar
	q := q0
	dir := None
	size := Line
	c := rune(0)
	for q < q1 {
		prevc = c
		c = getc(q)
		q++
		switch {
		default:
			return r, evalp, q - 1
		case c == ';':
			ar = r
			fallthrough
		case c == ',':
			if prevc == 0 { // lhs defaults to 0
				r.q0 = 0
			}
			if q >= q1 && t != nil { // rhs defaults to $
				r.q1 = t.Nc()
			} else {
				nr, evalp, q = address(showerr, t, lim, ar, q, q1, getc, evalp)
				r.q1 = nr.q1
			}
			return r, evalp, q
		case c == '+':
			fallthrough
		case c == '-':
			if q < q1 {
				nc = getc(q)
			} else {
				nc = 0
			}
			if evalp && (prevc == '+' || prevc == '-') &&
				(nc != '#' && nc != '/' && nc != '?') {
				r, evalp = number(showerr, t, r, 1, int(prevc), Line) // do previous one
			}
			dir = int(c)
		case c == '.':
			fallthrough
		case c == '$':
			if q != q0+1 {
				return r, evalp, q - 1
			}
			if evalp {
				if c == '.' {
					r = ar
				} else {
					r = Range{t.Nc(), t.Nc()}
				}
			}
			if q < q1 {
				dir = Fore
			} else {
				dir = None
			}
		case c == '#':
			if q >= q1 {
				return r, evalp, q - 1
			}
			c = getc(q)
			q++
			if c < '0' || '9' < c {
				return r, evalp, q - 1
			}
			size = Char
			fallthrough
		case c >= '0' && c <= '9':
			n = int(c - '0')
			for q < q1 {
				c = getc(q)
				q++
				if c < '0' || '9' < c {
					q--
					break
				}
				n = n*10 + int(c-'0')
			}
			if evalp {
				r, evalp = number(showerr, t, r, n, dir, size)
			}
			dir = None
			size = Line
		case c == '?':
			dir = Back
			fallthrough
		case c == '/':
			pat := ""
			for q < q1 {
				c = getc(q)
				q++
				switch c {
				case '\n':
					q--
					goto out
				case '\\':
					pat = pat + string(c)
					if q == q1 {
						goto out
					}
					c = getc(q)
					q++
				case '/':
					goto out
				}
				pat = pat + string(c)
			}
		out:
			if evalp {
				r, evalp = acmeregexp(showerr, t, lim, r, pat, dir)
			}
			dir = None
			size = Line
		}
	}
	if evalp && dir != None {
		r, evalp = number(showerr, t, r, 1, dir, Line) // do previous one
	}
	return r, evalp, q
}
