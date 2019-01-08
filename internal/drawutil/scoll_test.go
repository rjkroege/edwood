package drawutil

import (
	"os"
	"testing"
)

func TestMouseScrollSize(t *testing.T) {
	const key = "mousescrollsize"
	mss, ok := os.LookupEnv(key)
	if ok {
		defer os.Setenv(key, mss)
	} else {
		defer os.Unsetenv(key)
	}

	tt := []struct {
		s        string
		maxlines int
		n        int
	}{
		{"", 200, 1},
		{"0", 200, 1},
		{"-1", 200, 1},
		{"-42", 200, 1},
		{"two", 200, 1},
		{"1", 200, 1},
		{"42", 200, 42},
		{"123", 200, 123},
		{"%", 200, 1},
		{"0%", 200, 1},
		{"-1%", 200, 1},
		{"-42%", 200, 1},
		{"five%", 200, 1},
		{"123%", 200, 200},
		{"10%", 200, 20},
		{"100%", 200, 200},
	}
	for _, tc := range tt {
		os.Setenv(key, tc.s)
		scrollLines = 0
		scrollPercent = 0
		n := mouseScrollSize(always{}, tc.maxlines)
		if n != tc.n {
			t.Errorf("mousescrollsize of %v for %v lines is %v; expected %v",
				tc.s, tc.maxlines, n, tc.n)
		}
	}
}

type always struct{}

func (a always) Do(f func()) { f() }
