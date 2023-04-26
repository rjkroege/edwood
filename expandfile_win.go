//go:build windows

package main

import (
	"github.com/rjkroege/edwood/regexp"
)

// PAL: If our q1==q0 selection is within ' chars, we check if it's a filename, otherwise fall
// back to our original selection code.  Returns the quoted string.

// Interesting parts: Group 2 is the filename/name, group 10 is the address
const (
	filenameGroup = 2
	addressGroup  = 10
)

// These REs are insufficiently restrictive.  In theory a windows filename
// can have as many spaces as it wants.  But for a good experience we 
// want to limit it.  So we need to pre-process the ranges to chop off
// at repeated whitespace.  This happens in getspan()
var filenameRE = regexp.MustCompileAcme(`((([a-zA-Z]:|((\\\\|//)[a-zA-Z0-9.]*))?((\\|/)?([^<>:*|?"'\n])*)*))(:([0-9]+|(/[^ ']+)))?`)

var quotedfilenameRE = regexp.MustCompileAcme(`('(([a-zA-Z]:|((\\\\|//)[a-zA-Z0-9.]*))?((\\|/)?[^<>:*|?"'\n]*)*)')(:([0-9]+|(/[^ ']+)))?`)

// Return a span of characters bounded by either newlines or
// successive whitespaces.
func getspan(t *Text, q0 int) (qq0, qq1 int) {
	qq0 = q0
	qq1 = q0
	lastwaswhitespace := false
	for qq0 > 0 {
		c := t.ReadC(qq0 - 1)
		if c == '\n' {
			break
		}
		if isfilespace(c) && lastwaswhitespace {
			qq0++
			break
		}
		lastwaswhitespace = isfilespace(c)
		qq0--
	}
	lastwaswhitespace = false
	for qq1 < t.file.Nr() {
		c := t.ReadC(qq1)
		if c == '\n' {
			break
		}
		if isfilespace(c) && lastwaswhitespace {
			qq1--
			break
		}
		lastwaswhitespace = isfilespace(c)
		qq1++
	}
	return qq0, qq1
}

func findquotedcontext(t *Text, q0 int) (qq0, qq1 int) {
	// Let's try a radical departure.  Start by getting a line.
	qq0, qq1 = getspan(t, q0)
	if qq1 == qq0 {
		return q0, q0
	}

	n := qq1 - qq0
	rb := make([]rune, n)
	t.file.Read(qq0, rb[:n])

	found := quotedfilenameRE.FindForward(rb, 0, n, -1)
	for _, pair := range found {
		// qq0 is zero of the range returned.
		// The match is thus at qq0+pair[0:1], and (q0-qq0) must fall in that range
		if q0-qq0 >= pair[0] && q0-qq0 <= pair[1] {
			return pair[0] + qq0, pair[1] + qq0
		}
	}
	return q0, q0
}

func expandfile(t *Text, q0 int, q1 int, e *Expand) (success bool) {
	n := 0
	var found [][]int
	var rb []rune
	if q0 == q1 {
		qq0, qq1 := getspan(t, q0)
		n = qq1 - qq0
		if n == 0 {
			return false
		}
		rb = make([]rune, n)
		t.file.Read(qq0, rb[:n])
		found = quotedfilenameRE.FindForward(rb, 0, n, 1)
		if found == nil {
			found = filenameRE.FindForward(rb, 0, n, 1)
		}
		for i, pair := range found {
			// qq0 is zero of the range returned.
			// The match is thus at qq0+pair[0:1], and (q0-qq0) must fall in that range
			if q0-qq0 >= pair[0] && q0-qq0 <= pair[1] {
				q0, q1 = pair[0]+qq0, pair[1]+qq0
				found = found[i : i+1]
				break
			}
		}
	} else {
		n = q1 - q0
		if n == 0 {
			return false
		}
		rb = make([]rune, n)
		t.file.Read(q0, rb[:n])

		found = quotedfilenameRE.FindForward(rb, 0, n, 1)
		if found == nil {
			found = filenameRE.FindForward(rb, 0, n, 1)
		}
		if found != nil && found[0][0] != 0 {
			return false
		}
	}
	if found == nil {
		return false
	}
	e.q0 = q0
	e.q1 = q1
	amax := q1
	amin := amax
	filename := ""
	if found != nil && found[0][2*filenameGroup] != -1 {
		filename = string(rb[found[0][2*filenameGroup]:found[0][2*filenameGroup+1]])
		e.q0 = q0 + found[0][2*filenameGroup]
		e.q1 = q1 + found[0][2*filenameGroup+1]
	}
	if found != nil && found[0][2*addressGroup] != -1 {
		amin = found[0][2*addressGroup] + q0
		amax = found[0][2*addressGroup+1] + q0
		e.a0 = amin
		e.a1 = amax
	}
	isFile := func(name string) bool {
		e.name = name
		e.at = t
		e.a0 = amin
		_, _, e.a1 = address(true, nil, Range{-1, -1}, Range{0, 0}, e.a0, amax,
			func(q int) rune { return t.ReadC(q) }, false)
		return true
	}
	s := filename
	if amin == q0 {
		return isFile(filename)
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
