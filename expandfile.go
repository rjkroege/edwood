//go:build !windows

package main

// PAL: If our q1==q0 selection is within ' chars, we check if it's a filename, otherwise fall
// back to our original selection code.  Returns the quoted string.
func findquotedcontext(t *Text, q0 int) (qq0, qq1 int) {
	qq0 = q0
	qq1 = q0
	foundquote := false
	foundcolon := false
	for qq0 >= 0 {
		c := t.ReadC(qq0)
		if c == ':' {
			// We are looking for the case where the click
			// happened in an address, in which case we aren't (yet)
			// in a quoted context.
			foundcolon = true
			qq0--
			continue
		}
		if foundcolon && !foundquote && c != '\'' {
			qq0++
			break
		}
		if c == '\'' {
			if foundcolon {
				foundquote = true
				foundcolon = false // we no longer care about the colon
				qq1 = qq0          // Make the search rightward start from within the quotes.
				qq0--
				continue // Keep marching leftwards;
			}
			foundquote = true
			break
		}
		if !isfilec(c) && !isfilespace(c) {
			return q0, q0 // No quote found leftward.
		}
		qq0--
	}
	if !foundquote {
		return q0, q0
	}
	for qq1 < t.file.Nr() {
		c := t.ReadC(qq1 - 1)
		if c == '\'' {
			break
		}
		if !isfilec(c) && !isfilespace(c) {
			return q0, q0 // No quote found rightwards.
		}
		qq1++
	}
	return qq0, qq1
}

func expandfile(t *Text, q0 int, q1 int, e *Expand) (success bool) {
	amax := q1
	if q1 == q0 {
		// Check for being in a quoted string, find out if its a file.
		qq0, qq1 := findquotedcontext(t, q0)
		if qq0 != qq1 {
			// Invariant: qq0 and qq1-1 are '
			if expandfile(t, qq0+1, qq1-1, e) {
				// We have a file.  If we have a colon following our qq1+1 quote
				// we have to get it and add it to Expand.
				cq1 := qq1
				c := t.ReadC(cq1)
				if c != ':' { // We don't have any address information here.  Just return e.
					e.q0 = qq0
					e.q1 = qq1
					return true
				}
				cq1++
				// collect the address
				e.a0 = cq1
				for cq1 < t.file.Nr() {
					c := t.ReadC(cq1)
					if !isaddrc(c) && !isregexc(c) && c != '\'' {
						break
					}
					cq1++
				}
				e.a1 = cq1
				q0 = qq0
				q1 = cq1
				e.q0 = q0
				e.q1 = q1
				return true
			}
		} else {
			colon := int(-1)
			// TODO(rjk): utf8 conversion work.
			for q1 < t.file.Nr() {
				c := t.ReadC(q1)
				if !isfilec(c) {
					break
				}
				if c == ':' {
					colon = q1
					break
				}
				q1++
			}
			for q0 > 0 {
				c := t.ReadC(q0 - 1)
				if !isfilec(c) && !isaddrc(c) && !isregexc(c) {
					break
				}
				if colon < 0 && c == ':' {
					colon = q0 - 1
				}
				q0--
			}
			// if it looks like it might begin file: , consume address chars after :
			// otherwise terminate expansion at :
			if colon >= 0 {
				q1 = colon
				if colon < t.file.Nr()-1 {
					c := t.ReadC(colon + 1)
					if isaddrc(c) {
						q1 = colon + 1
						for q1 < t.file.Nr() {
							c := t.ReadC(q1)
							if !isaddrc(c) {
								break
							}
							q1++
						}
					}
				}
			}
			if q1 > q0 {
				if colon >= 0 { // stop at white space
					for amax = colon + 1; amax < t.file.Nr(); amax++ {
						c := t.ReadC(amax)
						if c == ' ' || c == '\t' || c == '\n' {
							break
						}
					}
				} else {
					amax = t.file.Nr()
				}
			}
		}
	}
	amin := amax
	e.q0 = q0
	e.q1 = q1
	n := q1 - q0
	if n == 0 {
		return false
	}
	// see if it's a file name
	rb := make([]rune, n)
	t.file.Read(q0, rb[:n])
	// first, does it have bad chars?
	nname := -1
	for i, c := range rb {
		if c == ':' && nname < 0 {
			if q0+i+1 >= t.file.Nr() {
				return false
			}
			if i != n-1 {
				if cc := t.ReadC(q0 + i + 1); !isaddrc(cc) {
					return false
				}
			}
			amin = q0 + i
			nname = i
		}
	}
	if nname == -1 {
		nname = n
	}
	for i := 0; i < nname; i++ {
		if !isfilec(rb[i]) && rb[i] != ' ' {
			return false
		}
	}
	isFile := func(name string) bool {
		e.name = name
		e.at = t
		e.a0 = amin + 1
		_, _, e.a1 = address(true, nil, Range{-1, -1}, Range{0, 0}, e.a0, amax,
			func(q int) rune { return t.ReadC(q) }, false)
		return true
	}
	s := string(rb[:nname])
	if amin == q0 {
		return isFile(s)
	}
	dname := t.DirName(s)
	// if it's already a window name, it's a file
	if lookfile(dname) != nil {
		return isFile(dname)
	}
	// if it's the name of a file, it's a file
	if ismtpt(dname) || !access(dname) {
		return false
	}
	return isFile(dname)
}
