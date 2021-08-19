// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package regexp

import (
	"regexp/syntax"
)

// FindBackward is similar to FindAllSubmatchIndex but searches
// backwards in r[start:end], taking care to match ^ and $ correctly.
func (re *Regexp) FindBackward(r []rune, start int, end int, n int) [][]int {
	if n < 0 {
		n = len(r) + 1
	}
	if end < 0 {
		end = len(r)
	}
	var result [][]int
	var prevMatch []int
	re.allMatchesRunesBackward(r, start, end, n, func(match []int) {
		if result == nil {
			result = make([][]int, 0, startSize)
		}
		switch {
		case prevMatch != nil && prevMatch[0] == prevMatch[1] && match[1] == prevMatch[1]:
			// Previous match was empty and current match ends with it.
			// Replace previous match with current one.
			result[len(result)-1] = match

		case prevMatch != nil && match[1] > prevMatch[0] && match[1] <= prevMatch[1]:
			// Match overlaps with previous one.
			// Replace previous match with current one.
			result[len(result)-1] = match

		case prevMatch != nil && match[1] > prevMatch[0]:
			// TODO(fhs): Is this possible?
			// Match overlaps with previous one
			// and possibly the match before the previous one.
			// Do nothing.

		default:
			result = append(result, match)
		}
		prevMatch = match
	})
	return result
}

// allMatchesRunesBackward calls deliver at most n times
// with the location of successive matches in the input text.
func (re *Regexp) allMatchesRunesBackward(r []rune, start int, end int, n int, deliver func([]int)) {
	ri := &inputRunes{
		str:   r,
		start: start,
		end:   end,
	}
	for pos, i, prevMatchStart := end, 0, -1; i < n && pos >= start; {
		matches := re.doExecuteInput1(ri, pos, re.prog.NumCap, nil)
		if len(matches) == 0 {
			pos--
			continue
		}

		accept := true
		if matches[1] == pos {
			// We've found an empty match.
			if matches[0] == prevMatchStart {
				// We don't allow an empty match right
				// after a previous match, so ignore it.
				accept = false
			}
		}
		pos--
		prevMatchStart = matches[0]

		if accept {
			deliver(re.pad(matches))
			i++
		}
	}
}

// doExecuteInput1 finds the match in the input that begins at pos (if there is one),
// and appends the position of its subexpressions to dstCap and returns dstCap.
//
// nil is returned if no matches are found and non-nil if matches are found.
func (re *Regexp) doExecuteInput1(i input, pos int, ncap int, dstCap []int) []int {
	if dstCap == nil {
		// Make sure 'return dstCap' is non-nil.
		dstCap = arrayNoInts[:0:0]
	}
	// TODO(fhs): we should use onepass and backtrack matcher here
	// but they take []byte, string, or io.RuneReader for input.

	m := re.get()
	m.init(ncap)
	if !m.match1(i, pos) {
		re.put(m)
		return nil
	}
	dstCap = append(dstCap, m.matchcap...)
	re.put(m)
	return dstCap
}

// match1 runs the machine over the input starting at pos.
// It reports whether a match was found.
// If so, m.matchcap holds the submatch information.
//
// Compared to match method, the match fails if it doesn't begin at given pos.
// We don't look for match starting at pos+1, pos+2, etc.
// (Prefix fast search is not used and m.p.Start PC is added only once.)
func (m *machine) match1(i input, pos int) bool {
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
	var started bool
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
		}
		if !m.matched && !started {
			if len(m.matchcap) > 0 {
				m.matchcap[0] = pos
			}
			m.add(runq, uint32(m.p.Start), pos, m.matchcap, &flag, nil)
			started = true
		}
		flag = newLazyFlag(r, r1)
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
