package main

import (
	"fmt"
	"regexp"
)

// An interface to regexp for acme.

type AcmeRegexp struct {
	re *regexp.Regexp
}

func rxcompile(r []rune) (*AcmeRegexp, error) {
	re, err := regexp.Compile(string(r))
	if err != nil {
		return nil, err
	}
	return &AcmeRegexp{re}, nil
}

type FRuneReader struct {
	buf Texter
	q   int
	eof int
}

type BRuneReader FRuneReader

func NewFRuneReader(b Texter, offset int, eof int) *FRuneReader {
	if eof > b.Nc() {
		eof = b.Nc()
	}
	if eof < 0 {
		eof = b.Nc()
	}
	return &FRuneReader{b, offset, eof}
}

func NewBRuneReader(b Texter, offset int) *BRuneReader {
	frr := NewFRuneReader(b, offset, 0)
	frr.q = offset - 1
	return (*BRuneReader)(frr)
}

func (frr *FRuneReader) ReadRune() (r rune, size int, err error) {
	if frr.q >= frr.eof {
		return 0, 0, fmt.Errorf("end of buffer")
	}
	rr := frr.buf.Read(frr.q, 1)
	frr.q++
	return rr[0], 1, nil
}

func (brr *BRuneReader) ReadRune() (r rune, size int, err error) {
	if brr.q < 0 {
		return 0, 0, fmt.Errorf("end of buffer")
	}
	rr := brr.buf.Read(brr.q, 1)
	brr.q--
	return rr[0], 1, nil
}

// works on Text if present, rune otherwise
func (re *AcmeRegexp) rxexecute(t Texter, r []rune, startp int, eof int, nmatch int) (rp RangeSet) {
	var source Texter
	if t != nil {
		source = t
	} else {
		source = &TextBuffer{0, 0, r}
	}

	rngs := RangeSet([]Range{})
	for len(rngs) < nmatch {
		reader := NewFRuneReader(source, int(startp), int(eof))
		loc := re.re.FindReaderIndex(reader)
		if loc == nil {
			return rngs
		}
		rng := Range{loc[0] + startp, loc[1] + startp}
		rngs = append(rngs, rng)
		startp += loc[1]
	}
	return rngs
}

func (re *AcmeRegexp) rxbexecute(t Texter, startp int, nmatch int) (rp RangeSet) {
	source := t

	rngs := RangeSet([]Range{})
	for startp >= 0 && len(rngs) < nmatch {
		reader := NewBRuneReader(source, int(startp))
		loc := re.re.FindReaderIndex(reader)
		if loc == nil {
			return rngs
		}
		rng := Range{startp - loc[1], startp - loc[0]}
		rngs = append(rngs, rng)
		startp = startp - loc[1] //TODO(flux): Does this follow "end" semantics, or do I need loc[0]-1
	}
	return rngs
}
