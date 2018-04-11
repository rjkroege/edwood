package main

const (
	None = iota
	Fore = '+'
	Back = '-'
)

const (
	Char = iota
	Line
)

// Return if r is valid character in an address
func isaddrc(r rune) bool {
	if utfrune([]rune("0123456789+-/$.#,;?"), r) != -1 {
		return true
	}
	return false
}

//* quite hard: could be almost anything but white space, but we are a little conservative,
//* aiming for regular expressions of alphanumerics and no white space

func isregexc(r rune) bool {
	if r == 0 {
		return false
	}
	if isalnum(rune(r)) {
		return true
	}
	if utfrune([]rune("^+-.*?#,;[]()$"), r) != -1 {
		return true
	}
	return false
}

// nlcounttopos starts at q0 and advances nl lines,
// being careful not to walk past the end of the text,
// and then nr chars, being careful not to walk past
// the end of the current line.
// It returns the final position.
func nlcounttopos(t Texter, q0 int, nl int, nr int) int {
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

func number(showerr bool, t Texter, r Range, line int, dir int, size int) (Range, bool) {
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
		break
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
		break
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

func acmeregexp(showerr bool, t Texter, lim Range, r Range, pat string, dir int) (retr Range, foundp bool) {
	var (
		sel RangeSet
		q   int
		err error
	)
	if len(pat) == 0 && pattern == nil {
		if showerr {
			warning(nil, "no previous regular expression\n")
		}
		return r, false
	}
	if len(pat) > 0 {
		pattern, err = rxcompile(pat)
		if err != nil {
			return r, false
		}
	}
	if dir == Back {
		sel = pattern.rxbexecute(t, r.q0, 1)
	} else {
		if lim.q0 < 0 {
			q = -1
		} else {
			q = lim.q1
		}
		sels := pattern.rxexecute(t, nil, r.q1, q, 1)
		if len(sels) > 0 { sel = sels[0] } else { sel = nil }
	}
	if len(sel) == 0 && showerr {
		warning(nil, "no match for regexp\n")
		return Range{-1, -1}, false
	}
	return sel[0], true
}

// getc takes a closure over the address expression and returns the qth rune.
func address(showerr bool, t Texter, lim Range, ar Range, q0 int, q1 int, getc func(q int) rune, eval bool) (r Range, evalp bool, qp int) {
	var (
		dir, size    int
		n            int
		prevc, c, nc rune
		q            int
		pat          string
		nr           Range
	)
	evalp = eval
	r = ar
	q = q0
	dir = None
	size = Line
	c = 0
	for q < q1 {
		prevc = c
		c = getc(q)
		q++
		switch {
		default:
			return r, evalp, q - 1
		case c == ';':
			ar = r
			// fall through
		case c == ',':
			if prevc == 0 { // lhs defaults to 0
				r.q0 = 0
			}
			text, ok := t.(*Text)
			if q >= q1 && t != nil && ok && text.file != nil { // rhs defaults to $
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
			break
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
			break
		case c == '#':
			if q == q1 {
				return r, evalp, q - 1
			}
			if q < q1 {
				c = getc(q)
			} else {
				c = 0
			}
			q++
			if c < '0' || '9' < c {
				return r, evalp, q - 1
			}
			size = Char
			fallthrough
		case c >= '0' && c <= '9':
			n = int(c - '0')
			for q < q1 {
				if q < q1 {
					c = getc(q)
				} else {
					c = 0
				}

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
			break
		case c == '?':
			dir = Back
			fallthrough
		case c == '/':
			pat = ""
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
					break
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
			break
		}
	}
	if evalp && dir != None {
		r, evalp = number(showerr, t, r, 1, dir, Line) // do previous one
	}
	return r, evalp, q
}
