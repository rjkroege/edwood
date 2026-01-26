package rich

import (
	"image/color"
	"testing"
)

func TestDefaultStyle(t *testing.T) {
	style := DefaultStyle()

	// Default style should have scale 1.0 (normal body text)
	if style.Scale != 1.0 {
		t.Errorf("DefaultStyle().Scale = %v, want 1.0", style.Scale)
	}

	// Default style should not be bold or italic
	if style.Bold {
		t.Error("DefaultStyle().Bold = true, want false")
	}
	if style.Italic {
		t.Error("DefaultStyle().Italic = true, want false")
	}

	// Default style should have nil colors (use defaults)
	if style.Fg != nil {
		t.Errorf("DefaultStyle().Fg = %v, want nil", style.Fg)
	}
	if style.Bg != nil {
		t.Errorf("DefaultStyle().Bg = %v, want nil", style.Bg)
	}
}

func TestStyleEquality(t *testing.T) {
	tests := []struct {
		name  string
		a, b  Style
		equal bool
	}{
		{
			name:  "default styles are equal",
			a:     DefaultStyle(),
			b:     DefaultStyle(),
			equal: true,
		},
		{
			name:  "bold styles are equal",
			a:     StyleBold,
			b:     Style{Bold: true, Scale: 1.0},
			equal: true,
		},
		{
			name:  "bold vs italic are not equal",
			a:     StyleBold,
			b:     StyleItalic,
			equal: false,
		},
		{
			name:  "different scales are not equal",
			a:     StyleH1,
			b:     StyleH2,
			equal: false,
		},
		{
			name:  "same colors are equal",
			a:     Style{Fg: color.Black, Bg: color.White, Scale: 1.0},
			b:     Style{Fg: color.Black, Bg: color.White, Scale: 1.0},
			equal: true,
		},
		{
			name:  "different fg colors are not equal",
			a:     Style{Fg: color.Black, Scale: 1.0},
			b:     Style{Fg: color.White, Scale: 1.0},
			equal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stylesEqual(tt.a, tt.b)
			if got != tt.equal {
				t.Errorf("stylesEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.equal)
			}
		})
	}
}

// stylesEqual compares two styles for equality.
// This is a helper for testing; Go struct comparison works for most cases,
// but color.Color comparison requires special handling.
func stylesEqual(a, b Style) bool {
	if a.Bold != b.Bold || a.Italic != b.Italic || a.Scale != b.Scale {
		return false
	}
	if !colorEqual(a.Fg, b.Fg) || !colorEqual(a.Bg, b.Bg) {
		return false
	}
	return true
}

// colorEqual compares two colors for equality, handling nil.
func colorEqual(a, b color.Color) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

func TestLinkStyleColor(t *testing.T) {
	// StyleLink should have a blue foreground color for rendering links
	if StyleLink.Fg == nil {
		t.Fatal("StyleLink.Fg is nil, want blue color")
	}

	// Check that it's blue (high blue component, low red/green)
	r, g, b, _ := StyleLink.Fg.RGBA()
	// Convert from 16-bit to 8-bit for easier comparison
	r8, g8, b8 := r>>8, g>>8, b>>8

	// Blue should be dominant
	if b8 <= r8 || b8 <= g8 {
		t.Errorf("StyleLink.Fg is not blue enough: R=%d, G=%d, B=%d", r8, g8, b8)
	}

	// Blue component should be substantial (at least 128/255)
	if b8 < 128 {
		t.Errorf("StyleLink.Fg blue component too low: %d, want >= 128", b8)
	}

	// StyleLink should have Link=true
	if !StyleLink.Link {
		t.Error("StyleLink.Link = false, want true")
	}

	// StyleLink should have normal scale
	if StyleLink.Scale != 1.0 {
		t.Errorf("StyleLink.Scale = %v, want 1.0", StyleLink.Scale)
	}
}
