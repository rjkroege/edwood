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
	// Compare list style fields
	if a.ListItem != b.ListItem || a.ListBullet != b.ListBullet {
		return false
	}
	if a.ListIndent != b.ListIndent || a.ListOrdered != b.ListOrdered {
		return false
	}
	if a.ListNumber != b.ListNumber {
		return false
	}
	// Compare table style fields
	if a.Table != b.Table || a.TableHeader != b.TableHeader {
		return false
	}
	if a.TableAlign != b.TableAlign {
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

func TestListStyleFields(t *testing.T) {
	// Test that list style fields exist and have expected default values
	t.Run("default style has no list fields set", func(t *testing.T) {
		s := DefaultStyle()
		if s.ListItem {
			t.Error("DefaultStyle().ListItem = true, want false")
		}
		if s.ListBullet {
			t.Error("DefaultStyle().ListBullet = true, want false")
		}
		if s.ListIndent != 0 {
			t.Errorf("DefaultStyle().ListIndent = %d, want 0", s.ListIndent)
		}
		if s.ListOrdered {
			t.Error("DefaultStyle().ListOrdered = true, want false")
		}
		if s.ListNumber != 0 {
			t.Errorf("DefaultStyle().ListNumber = %d, want 0", s.ListNumber)
		}
	})

	t.Run("can set unordered list bullet style", func(t *testing.T) {
		s := Style{
			ListBullet: true,
			ListIndent: 0,
			Scale:      1.0,
		}
		if !s.ListBullet {
			t.Error("ListBullet not set")
		}
		if s.ListOrdered {
			t.Error("ListOrdered should be false for unordered bullet")
		}
	})

	t.Run("can set ordered list item style", func(t *testing.T) {
		s := Style{
			ListItem:    true,
			ListOrdered: true,
			ListNumber:  3,
			ListIndent:  1,
			Scale:       1.0,
		}
		if !s.ListItem {
			t.Error("ListItem not set")
		}
		if !s.ListOrdered {
			t.Error("ListOrdered not set")
		}
		if s.ListNumber != 3 {
			t.Errorf("ListNumber = %d, want 3", s.ListNumber)
		}
		if s.ListIndent != 1 {
			t.Errorf("ListIndent = %d, want 1", s.ListIndent)
		}
	})

	t.Run("nested list indent levels", func(t *testing.T) {
		// Test multiple indentation levels
		levels := []int{0, 1, 2, 3}
		for _, level := range levels {
			s := Style{ListItem: true, ListIndent: level, Scale: 1.0}
			if s.ListIndent != level {
				t.Errorf("ListIndent = %d, want %d", s.ListIndent, level)
			}
		}
	})

	t.Run("list styles are comparable", func(t *testing.T) {
		s1 := Style{ListItem: true, ListIndent: 1, Scale: 1.0}
		s2 := Style{ListItem: true, ListIndent: 1, Scale: 1.0}
		s3 := Style{ListItem: true, ListIndent: 2, Scale: 1.0}

		if !stylesEqual(s1, s2) {
			t.Error("identical list styles should be equal")
		}
		if stylesEqual(s1, s3) {
			t.Error("different indent levels should not be equal")
		}
	})
}

func TestTableStyleFields(t *testing.T) {
	// Test that table style fields exist and have expected default values
	t.Run("default style has no table fields set", func(t *testing.T) {
		s := DefaultStyle()
		if s.Table {
			t.Error("DefaultStyle().Table = true, want false")
		}
		if s.TableHeader {
			t.Error("DefaultStyle().TableHeader = true, want false")
		}
		if s.TableAlign != AlignLeft {
			t.Errorf("DefaultStyle().TableAlign = %d, want AlignLeft (0)", s.TableAlign)
		}
	})

	t.Run("can set table cell style", func(t *testing.T) {
		s := Style{
			Table: true,
			Code:  true, // Tables render in code font
			Block: true, // Tables are block elements
			Scale: 1.0,
		}
		if !s.Table {
			t.Error("Table not set")
		}
		if !s.Code {
			t.Error("Code not set")
		}
		if !s.Block {
			t.Error("Block not set")
		}
	})

	t.Run("can set table header style", func(t *testing.T) {
		s := Style{
			Table:       true,
			TableHeader: true,
			Bold:        true, // Headers are typically bold
			Code:        true,
			Block:       true,
			Scale:       1.0,
		}
		if !s.TableHeader {
			t.Error("TableHeader not set")
		}
		if !s.Bold {
			t.Error("Bold not set for header")
		}
	})

	t.Run("table alignment values", func(t *testing.T) {
		// Test each alignment value
		tests := []struct {
			name  string
			align Alignment
		}{
			{"left", AlignLeft},
			{"center", AlignCenter},
			{"right", AlignRight},
		}
		for _, tt := range tests {
			s := Style{Table: true, TableAlign: tt.align, Scale: 1.0}
			if s.TableAlign != tt.align {
				t.Errorf("%s: TableAlign = %d, want %d", tt.name, s.TableAlign, tt.align)
			}
		}
	})

	t.Run("table styles are comparable", func(t *testing.T) {
		s1 := Style{Table: true, TableAlign: AlignCenter, Scale: 1.0}
		s2 := Style{Table: true, TableAlign: AlignCenter, Scale: 1.0}
		s3 := Style{Table: true, TableAlign: AlignRight, Scale: 1.0}

		if !stylesEqual(s1, s2) {
			t.Error("identical table styles should be equal")
		}
		if stylesEqual(s1, s3) {
			t.Error("different alignments should not be equal")
		}
	})
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
