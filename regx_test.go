package main

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/rjkroege/edwood/sam"
)

func TestRegexpForward(t *testing.T) {
	tt := []struct {
		text     string
		re       string
		expected []RangeSet
		nmax     int // Max number of matches
	}{
		{"aaa", "b", nil, 10},
		{"aaa", "a", []RangeSet{{{0, 1}}, {{1, 2}}, {{2, 3}}}, 10},
		{"cba", "ba", []RangeSet{{{1, 3}}}, 10},
		{"aaaaa", "a", []RangeSet{{{0, 1}}, {{1, 2}}}, 2},
	}
	for i, tc := range tt {
		t.Run(fmt.Sprintf("test-%02d", i), func(t *testing.T) {
			re, err := rxcompile(tc.re)
			if err != nil {
				t.Fatalf("failed to compile regular expression %q", tc.re)
			}
			text := sam.NewTextBuffer(0, 0, []rune(tc.text))
			rs := re.rxexecute(text, nil, 0, text.Nc(), tc.nmax)
			if !reflect.DeepEqual(rs, tc.expected) {
				t.Errorf("regexp %q incorrectly matches %q:\nexpected: %v\ngot: %v",
					tc.re, tc.text, tc.expected, rs)
			}
		})
	}
}

func TestRegexpBackward(t *testing.T) {
	tt := []struct {
		text     string
		re       string
		expected RangeSet
		nmax     int // Max number of matches
	}{
		{"baa", "ba", RangeSet{{0, 2}}, 10},
		{"aaa", "a", RangeSet{{2, 3}, {1, 2}, {0, 1}}, 10},
		{"cba", "a", RangeSet{{2, 3}}, 10},
		{"aba", "a", RangeSet{{2, 3}, {0, 1}}, 10},
		{"aaaa", "a", RangeSet{{3, 4}, {2, 3}}, 2},
	}
	for i, tc := range tt {
		t.Run(fmt.Sprintf("test-%02d", i), func(t *testing.T) {
			re, err := rxcompile(tc.re)
			if err != nil {
				t.Fatalf("failed to compile regular expression %q", tc.re)
			}
			text := sam.NewTextBuffer(0, 0, []rune(tc.text))
			rs := re.rxbexecute(text, text.Nc(), tc.nmax)
			if !reflect.DeepEqual(rs, tc.expected) {
				t.Errorf("regexp %q incorrectly matches %q:\nexpected: %v\ngot: %v",
					tc.re, tc.text, tc.expected, rs)
			}
		})
	}
}
