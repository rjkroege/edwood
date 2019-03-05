package frame

import (
	"github.com/rjkroege/edwood/internal/draw"
)

// All the dependencies on 9fans.net/go/draw along with mocking
// interfaces.

// Fontmetrics lets tests mock the calls into draw for measuring the
// width of UTF8 slices.
type Fontmetrics interface {
	BytesWidth([]byte) int
	DefaultHeight() int
	Impl() *draw.Font
	StringWidth(string) int
	RunesWidth([]rune) int
}

type frfont struct {
	*draw.Font
}

func (ff *frfont) DefaultHeight() int {
	return ff.Font.Height
}

func (ff *frfont) Impl() *draw.Font {
	return ff.Font
}
