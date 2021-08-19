// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package regexp

import (
	"regexp/syntax"

	"github.com/rjkroege/edwood/runes"
)

// CompileAcme is like Compile but treats ^ and $ as only matching
// beginning and end of lines respectively.
func CompileAcme(expr string) (*Regexp, error) {
	return compile(expr, syntax.Perl&^syntax.OneLine, false)
}

// FindForward is similar to FindAllSubmatchIndex but searches
// r[start:end], taking care to match ^ and $ correctly.
func (re *Regexp) FindForward(r []rune, start int, end int, n int) [][]int {
	if n < 0 {
		n = len(r) + 1
	}
	if end < 0 {
		end = len(r)
	}
	var result [][]int
	re.allMatchesRunes(r, start, end, n, func(match []int) {
		if result == nil {
			result = make([][]int, 0, startSize)
		}
		result = append(result, match)
	})
	return result
}

// allMatchesRunes calls deliver at most n times
// with the location of successive matches in the input text.
func (re *Regexp) allMatchesRunes(r []rune, start int, end int, n int, deliver func([]int)) {
	ri := &inputRunes{
		str:   r,
		start: start,
		end:   end,
	}
	for pos, i, prevMatchEnd := start, 0, -1; i < n && pos <= end; {
		matches := re.doExecuteInput(ri, pos, re.prog.NumCap, nil)
		if len(matches) == 0 {
			break
		}

		accept := true
		if matches[1] == pos {
			// We've found an empty match.
			if matches[0] == prevMatchEnd {
				// We don't allow an empty match right
				// after a previous match, so ignore it.
				accept = false
			}
			// TODO: use step()
			pos++
		} else {
			pos = matches[1]
		}
		prevMatchEnd = matches[1]

		if accept {
			deliver(re.pad(matches))
			i++
		}
	}
}

// doExecuteInput finds the leftmost match in the input, appends the position
// of its subexpressions to dstCap and returns dstCap.
//
// nil is returned if no matches are found and non-nil if matches are found.
func (re *Regexp) doExecuteInput(i input, pos int, ncap int, dstCap []int) []int {
	if dstCap == nil {
		// Make sure 'return dstCap' is non-nil.
		dstCap = arrayNoInts[:0:0]
	}
	// TODO(fhs): we should use onepass and backtrack matcher here
	// but they take []byte, string, or io.RuneReader for input.

	m := re.get()
	m.init(ncap)
	if !m.matchRunes(i, pos) {
		re.put(m)
		return nil
	}
	dstCap = append(dstCap, m.matchcap...)
	re.put(m)
	return dstCap
}

// matchRunes runs the machine over the input starting at pos.
// It reports whether a match was found.
// If so, m.matchcap holds the submatch information.
//
// Only change compared to the match method is that
// we use i.context to create the lazyFlag inside the loop,
// for correct handling of $.
func (m *machine) matchRunes(i input, pos int) bool {
	startCond := m.re.cond
	if startCond == ^syntax.EmptyOp(0) { // impossible
		return false
	}
	m.matched = false
	for i := range m.matchcap {
		m.matchcap[i] = -1
	}
	runq, nextq := &m.q0, &m.q1
	r, r1 := endOfText, endOfText
	width, width1 := 0, 0
	r, width = i.step(pos)
	if r != endOfText {
		r1, width1 = i.step(pos + width)
	}
	var flag lazyFlag
	if pos == 0 {
		flag = newLazyFlag(-1, r)
	} else {
		flag = i.context(pos)
	}
	for {
		if len(runq.dense) == 0 {
			if startCond&syntax.EmptyBeginText != 0 && pos != 0 {
				// Anchored match, past beginning of text.
				break
			}
			if m.matched {
				// Have match; finished exploring alternatives.
				break
			}
			if len(m.re.prefix) > 0 && r1 != m.re.prefixRune && i.canCheckPrefix() {
				// Match requires literal prefix; fast search for it.
				advance := i.index(m.re, pos)
				if advance < 0 {
					break
				}
				pos += advance
				r, width = i.step(pos)
				r1, width1 = i.step(pos + width)
			}
		}
		if !m.matched {
			if len(m.matchcap) > 0 {
				m.matchcap[0] = pos
			}
			m.add(runq, uint32(m.p.Start), pos, m.matchcap, &flag, nil)
		}
		flag = i.context(pos + width)
		m.step(runq, nextq, pos, pos+width, r, &flag)
		if width == 0 {
			break
		}
		if len(m.matchcap) == 0 && m.matched {
			// Found a match and not paying attention
			// to where it is, so any match will do.
			break
		}
		pos += width
		r, width = r1, width1
		if r != endOfText {
			r1, width1 = i.step(pos + width)
		}
		runq, nextq = nextq, runq
	}
	m.clear(nextq)
	return m.matched
}

// inputRunes scans a rune sub-slice: str[start:end].
type inputRunes struct {
	str        []rune
	start, end int
}

func (i *inputRunes) step(pos int) (rune, int) {
	if pos < i.end {
		return i.str[pos], 1
	}
	return endOfText, 0
}

func (i *inputRunes) canCheckPrefix() bool {
	return true
}

func (i *inputRunes) hasPrefix(re *Regexp) bool {
	return runes.HasPrefix(i.str[i.start:i.end], []rune(re.prefix))
}

func (i *inputRunes) index(re *Regexp, pos int) int {
	return runes.Index(i.str[pos:i.end], []rune(re.prefix))
}

func (i *inputRunes) context(pos int) lazyFlag {
	r1, r2 := endOfText, endOfText
	// 0 < pos && pos <= len(i.str)
	if uint(pos-1) < uint(len(i.str)) {
		r1 = i.str[pos-1]
	}
	// 0 <= pos && pos < len(i.str)
	if uint(pos) < uint(len(i.str)) {
		r2 = i.str[pos]
	}
	return newLazyFlag(r1, r2)
}
