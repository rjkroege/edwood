package rich

import (
	"github.com/rjkroege/edwood/draw"
)

// WithDisplay is an Option that sets the display for the frame.
func WithDisplay(d draw.Display) Option {
	return func(f *frameImpl) {
		f.display = d
	}
}

// WithBackground is an Option that sets the background image for the frame.
func WithBackground(b draw.Image) Option {
	return func(f *frameImpl) {
		f.background = b
	}
}

// WithFont is an Option that sets the font for the frame.
func WithFont(f draw.Font) Option {
	return func(fi *frameImpl) {
		fi.font = f
	}
}

// WithTextColor is an Option that sets the text color image for the frame.
func WithTextColor(c draw.Image) Option {
	return func(fi *frameImpl) {
		fi.textColor = c
	}
}

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

// WithCodeFont is an Option that sets the monospace font for code spans.
func WithCodeFont(f draw.Font) Option {
	return func(fi *frameImpl) {
		fi.codeFont = f
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

// WithSelectionColor is an Option that sets the selection highlight color image.
func WithSelectionColor(c draw.Image) Option {
	return func(fi *frameImpl) {
		fi.selectionColor = c
	}
}

// WithImageCache is an Option that sets the image cache for the frame.
// When set, the frame will use this cache to load images during layout.
// If nil, images will not be loaded and will display placeholders.
func WithImageCache(cache *ImageCache) Option {
	return func(fi *frameImpl) {
		fi.imageCache = cache
	}
}

// WithBasePath is an Option that sets the base path for resolving relative image paths.
// This should be the path to the source file (e.g., markdown file) containing image references.
// When combined with WithImageCache, relative image paths like "images/photo.png" will be
// resolved relative to this base path's directory.
func WithBasePath(path string) Option {
	return func(fi *frameImpl) {
		fi.basePath = path
	}
}
