// Package drawutil contains draw related utility functions.
package drawutil

import (
	"os"
	"strconv"
	"sync"
)

var scrollSizeOnce sync.Once
var scrollLines int
var scrollPercent float64

// MouseScrollSize computes the number of lines of text that should be
// scrolled in response to a mouse scroll wheel click. Maxlines is the
// number of lines visible in the text window.
//
// The default scroll increment is one line. This default can be overridden
// by setting the $mousescrollsize environment variable to an integer,
// which specifies a constant number of lines, or to a real number followed
// by a percent character, indicating that the scroll increment should be a
// percentage of the total number of lines in the window. For example,
// setting $mousescrollsize to 50% causes a half-window scroll increment.
func MouseScrollSize(maxlines int) int {
	return mouseScrollSize(&scrollSizeOnce, maxlines)
}

type doer interface {
	Do(func())
}

func mouseScrollSize(once doer, maxlines int) int {
	once.Do(func() {
		s := os.Getenv("mousescrollsize")
		if s == "" {
			return
		}
		if s[len(s)-1] == '%' {
			pcnt, err := strconv.ParseFloat(s[:len(s)-1], 32)
			if err != nil || pcnt <= 0 {
				return
			}
			if pcnt > 100 {
				pcnt = 100
			}
			scrollPercent = pcnt
			return
		}
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			return
		}
		scrollLines = n
	})
	if scrollLines > 0 {
		return scrollLines
	}
	if scrollPercent > 0 {
		return int(scrollPercent * float64(maxlines) / 100.0)
	}
	return 1
}
