package main

import (
	"regexp"
)

// An interface to regexp for acme.

type AcmeRegexp struct {
	re *regexp.Regexp
	exception rune // ^ or $ or 0
}

func rxcompile(r string) (*AcmeRegexp, error) {
	re, err := regexp.Compile("(?m)"+r)
	if err != nil {
		return nil, err
	}
	are := &AcmeRegexp{re, 0}
	switch r {
	case "^": are.exception = '^'
	case "$": are.exception = '$'
	}
	return are, nil
}

// works on Text if present, rune otherwise
func (re *AcmeRegexp) rxexecute(t Texter, r []rune, startp int, eof int, nmatch int) (rp []RangeSet) {
	var source Texter
	if t != nil {
		source = t
	} else {
		source = &TextBuffer{0, 0, r}
	}

	if eof == -1 {
		eof = source.Nc()
	}
	view := source.View(startp, eof)
	rngs := []RangeSet{}
	locs := re.re.FindAllSubmatchIndex(view, nmatch)
loop:
	for _, loc := range locs {
		// Filter out ^ not at start of a line, $ not at end
		if len(loc) != 0  && loc[0] == loc[1] { 
			switch {
			case re.exception == '^' &&  loc[0] + startp == 0: // start of text is star-of-line
				break
			case re.exception == '^' &&  t.ReadC(loc[0]+startp-1) == '\n': // ^ after newline
				break
			case re.exception == '$' &&  loc[0] == t.Nc()-startp: // $ at end of text
				break
			case re.exception == '$' &&  t.ReadC(loc[0]+startp) == '\n': // $ at newline
				break
			default: 
				continue loop
			}
		}
		rs := RangeSet([]Range{})
		for i := 0; i < len(loc); i += 2 {
			rng := Range{loc[i] + startp, loc[i+1] + startp}
			rs = append(rs, rng)
		}
		rngs = append(rngs, rs)
	}
	return rngs
}

func (re *AcmeRegexp) rxbexecute(t Texter, startp int, nmatch int) (rp RangeSet) {
	Unimpl()
	return []Range{}
}
/* TODO(flux): This is broken, I'm pretty sure.  You can'd just read backwards,
you also need the backwards regexp

	source := t

	rngs := RangeSet([]Range{})
	for startp >= 0 && len(rngs) < nmatch {
		reader := NewBRuneReader(source, int(startp))
		locs := re.re.FindReaderSubmatchIndex(reader)
		if locs == nil {
			return rngs
		}
		for i := 0; i < len(locs); i += 2 {
			rng := Range{startp - locs[i+1], startp - locs[i]}
			rngs = append(rngs, rng)
		}
		startp = startp - locs[1] //TODO(flux): Does this follow "end" semantics, or do I need loc[0]-1
	}
	return rngs
}
*/
