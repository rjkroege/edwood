package main

import (
	"testing"
)

func testRegexpForward(t *testing.T) {
	tests := []struct {
		text     string
		re       string
		expected []RangeSet
		nmax     int // Max number of matches
	}{
		{"aaa", "b", []RangeSet{}, 10},
		{"aaa", "a", []RangeSet{{{0, 1}}, {{1, 2}}, {{2, 3}}}, 10},
		{"cba", "ba", []RangeSet{{{2, 3}}}, 10},
		{"aaaaa", "a", []RangeSet{{{0, 1}}, {{1, 2}}}, 2},
	}

	for i, test := range tests {
		are, err := rxcompile(test.re)
		if err != nil {
			t.Errorf("Failed to compile tests[%d].re = '%v'", i, test.re)
		}
		text := &TextBuffer{0, 0, []rune(test.text)}
		rs := are.rxexecute(text, nil, 0, 0x7FFFFFF /*text.nc()*/, test.nmax)
		if len(rs) != len(test.expected) {
			t.Errorf("Mismatch tests[%d] - expected %d elements, got %d", i, len(test.expected), len(rs))
			t.Errorf("\trs = %#v", rs)
		} else {
			for j, r := range rs {
				// TODO(flux): r[0] below assumes only one element coming back in each RangeSet
				if r[0].q0 != test.expected[j][0].q0 {
					t.Errorf("Mismatch tests[%d].expected[%d][0].q0=%d, got %d", i, j, tests[i].expected[j][0].q0, r[0].q0)
				}
				if r[0].q1 != test.expected[j][0].q1 {
					t.Errorf("Mismatch tests[%d].expected[%d][0].q1=%d, got %d", i, j, tests[i].expected[j][0].q1, r[0].q1)
				}
			}
		}
	}
}

// Not expected to pass until rxbexecute is implemented.
/*
func TestRegexpBackward(t *testing.T) {
	tests := []struct {
		text     string
		re       string
		expected RangeSet
		nmax     int
	}{
		{"baa", "ba", RangeSet{{0, 1}}, 10},
		{"aaa", "a", RangeSet{{2, 3}, {1, 2}, {0, 1}}, 10},
		{"cba", "a", RangeSet{{2, 3}}, 10},
		{"aba", "a", RangeSet{{2, 3}, {0, 1}}, 10},
		{"aaaa", "a", RangeSet{{3, 4}, {2, 3}}, 2},
	}
	for i, test := range tests {
		are, err := rxcompile(test.re)
		if err != nil {
			t.Errorf("Failed to compile tests[%d].re = '%v'", i, test.re)
		}
		text := &TextBuffer{0, 0, []rune(test.text)}
		rs := are.rxbexecute(text, text.Nc(), test.nmax)
		if len(rs) != len(test.expected) {
			t.Errorf("Mismatch tests[%d] - expected %d elements, got %d", i, len(test.expected), len(rs))
			t.Errorf("\trs = %#v", rs)
		} else {
			for j, r := range rs {
				if r.q0 != test.expected[j].q0 {
					t.Errorf("Mismatch tests[%d].expected[%d].q0=%d, got %d", i, j, tests[i].expected[j].q0, r.q0)
				}
				if r.q1 != test.expected[j].q1 {
					t.Errorf("Mismatch tests[%d].expected[%d].q1=%d, got %d", i, j, tests[i].expected[j].q1, r.q1)
				}
			}
		}
	}
}
*/
