package regexp

import (
	"fmt"
	"reflect"
	"testing"
)

type runesTest struct {
	text       string
	start, end int
	re         string
	expected   [][]int
	nmax       int // Max number of matches
}

// runesTests should work for forward search, and for backwards search
// after expected matches are reversed.
var runesTests = []runesTest{
	{"aaa", 0, -1, "b", [][]int(nil), 10},
	{"aaa", 0, -1, "a", [][]int{{0, 1}, {1, 2}, {2, 3}}, 10},
	{"aaaaa", 0, -1, "a+", [][]int{{0, 5}}, 10},
	{"aaaaaxyz", 0, -1, "xyz", [][]int{{5, 8}}, 10},
	{"aba", 0, -1, "a", [][]int{{0, 1}, {2, 3}}, 10},
	{"baaaaaa", 0, -1, "ba", [][]int{{0, 2}}, 10},
	{"cba", 0, -1, "cba", [][]int{{0, 3}}, 10},
	{"cba", 0, -1, "ba", [][]int{{1, 3}}, 10},
	{"cba", 0, -1, "a", [][]int{{2, 3}}, 10},
	{"two", 0, -1, "(one|two|three)", [][]int{{0, 3, 0, 3}}, 10},
	{"abcd\nbcd\n", 1, -1, "^b", [][]int{{5, 6}}, 10},
	{"abcd\nabcd\n", 1, -1, "^", [][]int{{5, 5}, {10, 10}}, 10},
	{"01234\nabcd\nabcx\n", 0, -1, "^abc", [][]int{{6, 9}, {11, 14}}, 10},
	{"01234\nabcd\nabcx\n", 0, -1, "^", [][]int{{0, 0}, {6, 6}, {11, 11}, {16, 16}}, 10},
	{"01234^abcd\n", 0, -1, "^", [][]int{{0, 0}, {11, 11}}, 10},
	{"01234\nabcd\nwxyz\n", 7, 13, "^", [][]int{{11, 11}}, 10},
	{"01234\nabcd\nwxyz\n", 7, 13, "$", [][]int{{10, 10}}, 10},
	{"aaa\naa", 0, -1, ".*", [][]int{{0, 3}, {4, 6}}, 10},
	{"<html></html>", 0, -1, "<.*>", [][]int{{0, 13}}, 10},
	{"! 世界 ! 世界 ! 世界 !", 0, -1, "!", [][]int{{0, 1}, {5, 6}, {10, 11}, {15, 16}}, -1},
	{"α 世界 α 世界 α 世界 α", 0, -1, "α", [][]int{{0, 1}, {5, 6}, {10, 11}, {15, 16}}, -1},
	{"世界 αβδ 世界", 0, -1, "αβδ", [][]int{{3, 6}}, -1},
}

func TestRegexpForward(t *testing.T) {
	tt := []runesTest{
		{"aaaaa", 0, -1, "a", [][]int{{0, 1}, {1, 2}}, 2},
		{"ab000ab000ab000", 0, -1, "ab", [][]int{{0, 2}, {5, 7}}, 2},
	}
	tt = append(tt, runesTests...)

	runRunesTests(t, tt, func(re *Regexp, tc *runesTest) [][]int {
		return re.FindForward([]rune(tc.text), tc.start, tc.end, tc.nmax)
	})
}

func TestRegexpBackward(t *testing.T) {
	tt := []runesTest{
		{"aaaaa", 0, -1, "a", [][]int{{4, 5}, {3, 4}}, 2},
		{"ab000ab000ab000", 0, -1, "ab", [][]int{{10, 12}, {5, 7}}, 2},
	}
	for _, tc := range runesTests {
		tc.expected = reverseMatches(tc.expected)
		tt = append(tt, tc)
	}
	runRunesTests(t, tt, func(re *Regexp, tc *runesTest) [][]int {
		return re.FindBackward([]rune(tc.text), tc.start, tc.end, tc.nmax)
	})
}

func runRunesTests(t *testing.T, tt []runesTest, matcher func(*Regexp, *runesTest) [][]int) {
	for i, tc := range tt {
		t.Run(fmt.Sprintf("test-%02d", i), func(t *testing.T) {
			re, err := CompileAcme(tc.re)
			if err != nil {
				t.Fatalf("failed to compile regular expression %q", tc.re)
			}
			rs := matcher(re, &tc)
			if !reflect.DeepEqual(rs, tc.expected) {
				t.Errorf("regexp %q incorrectly matches %q[%v:%v]:\nexpected: %#v\ngot: %#v",
					tc.re, tc.text, tc.start, tc.end, tc.expected, rs)
			}
		})
	}
}

func reverseMatches(m [][]int) [][]int {
	if m == nil {
		return nil
	}
	b := make([][]int, len(m))
	for i, s := range m {
		b[len(m)-i-1] = s
	}
	return b
}
