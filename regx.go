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
	buf Buffer
	q int
	eof int
}

type BRuneReader FRuneReader

func NewFRuneReader(b Buffer, offset int, eof int) *FRuneReader {
	if eof > b.nc() { eof = b.nc() }
	if eof < 0 { eof = b.nc() }
	return &FRuneReader{b, offset, eof}
}

func NewBRuneReader(b Buffer, offset int) *BRuneReader {
	frr := NewFRuneReader(b, offset, 0)
	frr.q = offset-1
	return (*BRuneReader)(frr)
}

func (frr *FRuneReader)ReadRune()(r rune, size int, err error) {
	if frr.q >= frr.eof { return 0,0, fmt.Errorf("end of buffer") }
	rr := frr.buf.Read(frr.q, 1)
	frr.q++
	return rr[0], 4, nil
}

func (brr *BRuneReader)ReadRune()(r rune, size int, err error) {
	if brr.q < 0 { return 0,0, fmt.Errorf("end of buffer") }
	rr := brr.buf.Read(brr.q, 1)
	brr.q--
	return rr[0], 4, nil
}

// works on Text if present, rune otherwise
func (re *AcmeRegexp)rxexecute(t *Text, r []rune, startp uint, eof uint) (rp RangeSet) {
	var source Buffer
	if t != nil {
		source = t.file.b
	} else {
		source = Buffer(r)
	}
	

	rngs := RangeSet([]Range{})
	for {
		reader := NewFRuneReader(source, int(startp), int(eof))
		loc := re.re.FindReaderIndex(reader)
		if loc == nil { return rngs }
		rng := Range{loc[0], loc[1]}
		rngs = append(rngs, rng)
		startp = uint(loc[1])
	}
}

func (re *AcmeRegexp)rxbexecute(t *Text, startp uint) (rp RangeSet) {
	var source Buffer
	source = t.file.b

	rngs := RangeSet([]Range{})
	for {
		reader := NewBRuneReader(source, int(startp))
		loc := re.re.FindReaderIndex(reader)
		if loc == nil { return rngs }
		rng := Range{loc[0], loc[1]}
		rngs = append(rngs, rng)
		startp = uint(loc[0]) //TODO(flux): Does this follow "end" semantics, or do I need loc[0]-1
	}
}