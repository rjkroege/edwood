package rich

import (
	"github.com/rjkroege/edwood/draw"
)

// WithBoldFont is an Option that sets the bold font variant for the frame.
func WithBoldFont(f draw.Font) Option {
	return func(fi *frameImpl) {
		fi.boldFont = f
	}
}

// WithItalicFont is an Option that sets the italic font variant for the frame.
func WithItalicFont(f draw.Font) Option {
	return func(fi *frameImpl) {
		fi.italicFont = f
	}
}

// WithBoldItalicFont is an Option that sets the bold-italic font variant for the frame.
func WithBoldItalicFont(f draw.Font) Option {
	return func(fi *frameImpl) {
		fi.boldItalicFont = f
	}
}

// WithScaledFont is an Option that sets a scaled font for a specific scale factor.
// The frame stores a map of scale factors to fonts.
// Common scale factors: 2.0 for H1, 1.5 for H2, 1.25 for H3.
func WithScaledFont(scale float64, f draw.Font) Option {
	return func(fi *frameImpl) {
		if fi.scaledFonts == nil {
			fi.scaledFonts = make(map[float64]draw.Font)
		}
		fi.scaledFonts[scale] = f
	}
}
