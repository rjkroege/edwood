package main

import (
	"github.com/rjkroege/edwood/regexp"
	"github.com/rjkroege/edwood/sam"
)

// TODO(rjk): Regexps should stream. We need a forward/back Rune streaming interface.

// AcmeRegexp is the representation of a compiled regular expression for acme.
type AcmeRegexp struct {
	*regexp.Regexp
}

// rxcompile parses a regular expression and returns a regular expression object
// that can be used to match against text.
func rxcompile(r string) (*AcmeRegexp, error) {
	re, err := regexp.CompileAcme(r)
	if err != nil {
		return nil, err
	}
	return &AcmeRegexp{
		Regexp: re,
	}, nil
}

// rxexecute searches forward in r[start:end] (from beginning of the slice to the end)
// and returns at most n matches. If r is nil, it is derived from t.
func (re *AcmeRegexp) rxexecute(t sam.Texter, r []rune, start int, end int, n int) []RangeSet {
	if r == nil {
		// TODO(rjk): This is horrible. Stream here instead.
		r = make([]rune, t.Nc())
		t.ReadB(0, r[:t.Nc()])
	}
	return matchesToRangeSets(re.FindForward(r, start, end, n))
}

// rxbexecute derives the full rune slice r from t and searches backwards in r[:end]
// (from end of the slice to the beginning) and returns at most n matches.
func (re *AcmeRegexp) rxbexecute(t sam.Texter, end int, n int) RangeSet {
	// TODO(rjk): This is horrible. Stream here instead.
	r := make([]rune, t.Nc())
	t.ReadB(0, r[:t.Nc()])
	matches := re.FindBackward(r, 0, end, n)
	var rs RangeSet
	for _, m := range matches {
		rs = append(rs, Range{
			q0: m[0],
			q1: m[1],
		})
	}
	return rs
}

func matchesToRangeSets(matches [][]int) []RangeSet {
	var out []RangeSet
	for _, m := range matches {
		var rs RangeSet
		for k := 0; k < len(m); k += 2 {
			rs = append(rs, Range{
				q0: m[k],
				q1: m[k+1],
			})
		}
		out = append(out, rs)
	}
	return out
}
