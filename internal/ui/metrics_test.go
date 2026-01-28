// Package ui provides user interface utilities for edwood.
package ui

import (
	"testing"
)

// TestLayoutMetricsNew tests that a new LayoutMetrics is properly initialized.
func TestLayoutMetricsNew(t *testing.T) {
	lm := NewLayoutMetrics(16, 14)
	if lm == nil {
		t.Fatal("NewLayoutMetrics returned nil")
	}

	if lm.TagFontHeight() != 16 {
		t.Errorf("TagFontHeight() = %d; want 16", lm.TagFontHeight())
	}
	if lm.BodyFontHeight() != 14 {
		t.Errorf("BodyFontHeight() = %d; want 14", lm.BodyFontHeight())
	}
}

// TestLayoutMetricsZeroHeights tests metrics with zero heights.
func TestLayoutMetricsZeroHeights(t *testing.T) {
	lm := NewLayoutMetrics(0, 0)
	if lm == nil {
		t.Fatal("NewLayoutMetrics returned nil")
	}

	if lm.TagFontHeight() != 0 {
		t.Errorf("TagFontHeight() = %d; want 0", lm.TagFontHeight())
	}
	if lm.BodyFontHeight() != 0 {
		t.Errorf("BodyFontHeight() = %d; want 0", lm.BodyFontHeight())
	}
}

// TestLayoutMetricsSetTagFontHeight tests updating tag font height.
func TestLayoutMetricsSetTagFontHeight(t *testing.T) {
	lm := NewLayoutMetrics(16, 14)
	lm.SetTagFontHeight(20)

	if lm.TagFontHeight() != 20 {
		t.Errorf("TagFontHeight() = %d; want 20", lm.TagFontHeight())
	}
	// Body should be unchanged
	if lm.BodyFontHeight() != 14 {
		t.Errorf("BodyFontHeight() = %d; want 14", lm.BodyFontHeight())
	}
}

// TestLayoutMetricsSetBodyFontHeight tests updating body font height.
func TestLayoutMetricsSetBodyFontHeight(t *testing.T) {
	lm := NewLayoutMetrics(16, 14)
	lm.SetBodyFontHeight(18)

	if lm.BodyFontHeight() != 18 {
		t.Errorf("BodyFontHeight() = %d; want 18", lm.BodyFontHeight())
	}
	// Tag should be unchanged
	if lm.TagFontHeight() != 16 {
		t.Errorf("TagFontHeight() = %d; want 16", lm.TagFontHeight())
	}
}

// TestLayoutMetricsTagLineHeight calculates tag line height including spacing.
func TestLayoutMetricsTagLineHeight(t *testing.T) {
	tests := []struct {
		name       string
		tagHeight  int
		bodyHeight int
		tagLines   int
		want       int
	}{
		{"single line, same heights", 16, 16, 1, 16},
		{"single line, different heights", 18, 14, 1, 18},
		{"multi line tag", 16, 14, 3, 48},
		{"zero tag lines", 16, 14, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutMetrics(tt.tagHeight, tt.bodyHeight)
			got := lm.TagLinesHeight(tt.tagLines)
			if got != tt.want {
				t.Errorf("TagLinesHeight(%d) = %d; want %d", tt.tagLines, got, tt.want)
			}
		})
	}
}

// TestLayoutMetricsBodyLineHeight calculates body line height including spacing.
func TestLayoutMetricsBodyLineHeight(t *testing.T) {
	tests := []struct {
		name       string
		tagHeight  int
		bodyHeight int
		bodyLines  int
		want       int
	}{
		{"single line, same heights", 16, 16, 1, 16},
		{"single line, different heights", 18, 14, 1, 14},
		{"multi line body", 16, 14, 10, 140},
		{"zero body lines", 16, 14, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutMetrics(tt.tagHeight, tt.bodyHeight)
			got := lm.BodyLinesHeight(tt.bodyLines)
			if got != tt.want {
				t.Errorf("BodyLinesHeight(%d) = %d; want %d", tt.bodyLines, got, tt.want)
			}
		})
	}
}

// TestLayoutMetricsWindowHeight calculates total window height.
func TestLayoutMetricsWindowHeight(t *testing.T) {
	tests := []struct {
		name       string
		tagHeight  int
		bodyHeight int
		tagLines   int
		bodyLines  int
		border     int
		want       int
	}{
		// Window height = tag lines height + border + body lines height + 1 (for separator)
		{"basic window", 16, 14, 1, 10, 2, 16 + 2 + 140 + 1},
		{"multi-line tag", 18, 14, 3, 5, 2, 54 + 2 + 70 + 1},
		{"no body lines", 16, 14, 1, 0, 2, 16 + 2 + 0 + 1},
		{"zero border", 16, 14, 1, 10, 0, 16 + 0 + 140 + 1},
		{"same heights", 16, 16, 2, 5, 1, 32 + 1 + 80 + 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutMetrics(tt.tagHeight, tt.bodyHeight)
			got := lm.WindowHeight(tt.tagLines, tt.bodyLines, tt.border)
			if got != tt.want {
				t.Errorf("WindowHeight(%d, %d, %d) = %d; want %d",
					tt.tagLines, tt.bodyLines, tt.border, got, tt.want)
			}
		})
	}
}

// TestLayoutMetricsBodyLinesForHeight calculates body lines that fit in a height.
func TestLayoutMetricsBodyLinesForHeight(t *testing.T) {
	tests := []struct {
		name       string
		tagHeight  int
		bodyHeight int
		height     int
		want       int
	}{
		{"exact fit", 16, 14, 140, 10},
		{"partial line excluded", 16, 14, 147, 10},
		{"zero height", 16, 14, 0, 0},
		{"less than one line", 16, 14, 10, 0},
		{"exact one line", 16, 14, 14, 1},
		{"different heights", 18, 20, 100, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutMetrics(tt.tagHeight, tt.bodyHeight)
			got := lm.BodyLinesForHeight(tt.height)
			if got != tt.want {
				t.Errorf("BodyLinesForHeight(%d) = %d; want %d", tt.height, got, tt.want)
			}
		})
	}
}

// TestLayoutMetricsTagLinesForHeight calculates tag lines that fit in a height.
func TestLayoutMetricsTagLinesForHeight(t *testing.T) {
	tests := []struct {
		name       string
		tagHeight  int
		bodyHeight int
		height     int
		want       int
	}{
		{"exact fit", 16, 14, 48, 3},
		{"partial line excluded", 16, 14, 50, 3},
		{"zero height", 16, 14, 0, 0},
		{"less than one line", 16, 14, 10, 0},
		{"exact one line", 16, 14, 16, 1},
		{"different heights", 20, 14, 100, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutMetrics(tt.tagHeight, tt.bodyHeight)
			got := lm.TagLinesForHeight(tt.height)
			if got != tt.want {
				t.Errorf("TagLinesForHeight(%d) = %d; want %d", tt.height, got, tt.want)
			}
		})
	}
}

// TestLayoutMetricsMinWindowHeight returns minimum height for a window.
func TestLayoutMetricsMinWindowHeight(t *testing.T) {
	tests := []struct {
		name       string
		tagHeight  int
		bodyHeight int
		border     int
		want       int
	}{
		// Min = 1 tag line + border + 1 body line + 1 separator
		{"basic", 16, 14, 2, 16 + 2 + 14 + 1},
		{"larger tag", 20, 14, 2, 20 + 2 + 14 + 1},
		{"larger body", 16, 20, 2, 16 + 2 + 20 + 1},
		{"zero border", 16, 14, 0, 16 + 0 + 14 + 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutMetrics(tt.tagHeight, tt.bodyHeight)
			got := lm.MinWindowHeight(tt.border)
			if got != tt.want {
				t.Errorf("MinWindowHeight(%d) = %d; want %d", tt.border, got, tt.want)
			}
		})
	}
}

// TestLayoutMetricsTotalLines calculates effective total lines accounting for height difference.
func TestLayoutMetricsTotalLines(t *testing.T) {
	tests := []struct {
		name       string
		tagHeight  int
		bodyHeight int
		tagLines   int
		bodyLines  int
		want       int
	}{
		// When heights are the same, total = tagLines + bodyLines
		{"same heights", 16, 16, 2, 10, 12},
		// When tag is taller, its lines count for more
		{"tag taller", 20, 10, 2, 10, 14}, // 2*20/10 + 10 = 4 + 10 = 14 effective body lines
		// When body is taller, tag lines count for less
		{"body taller", 10, 20, 4, 10, 12}, // 4*10/20 + 10 = 2 + 10 = 12 effective body lines
		{"zero tag lines", 16, 14, 0, 10, 10},
		{"zero body lines", 16, 14, 2, 0, 2}, // just the tag contribution
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutMetrics(tt.tagHeight, tt.bodyHeight)
			got := lm.TotalLinesEquivalent(tt.tagLines, tt.bodyLines)
			if got != tt.want {
				t.Errorf("TotalLinesEquivalent(%d, %d) = %d; want %d",
					tt.tagLines, tt.bodyLines, got, tt.want)
			}
		})
	}
}

// TestLayoutMetricsZeroDivision ensures no panic with zero heights.
func TestLayoutMetricsZeroDivision(t *testing.T) {
	// Should not panic even with zero font heights
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("function panicked: %v", r)
		}
	}()

	lm := NewLayoutMetrics(0, 0)

	// These should return 0, not panic
	if got := lm.BodyLinesForHeight(100); got != 0 {
		t.Errorf("BodyLinesForHeight with zero body height = %d; want 0", got)
	}
	if got := lm.TagLinesForHeight(100); got != 0 {
		t.Errorf("TagLinesForHeight with zero tag height = %d; want 0", got)
	}
	if got := lm.TotalLinesEquivalent(5, 10); got != 0 {
		t.Errorf("TotalLinesEquivalent with zero heights = %d; want 0", got)
	}
}

// TestLayoutMetricsEquality tests that metrics with same values are equal.
func TestLayoutMetricsEquality(t *testing.T) {
	lm1 := NewLayoutMetrics(16, 14)
	lm2 := NewLayoutMetrics(16, 14)
	lm3 := NewLayoutMetrics(16, 18)

	if !lm1.Equal(lm2) {
		t.Error("metrics with same values should be equal")
	}
	if lm1.Equal(lm3) {
		t.Error("metrics with different values should not be equal")
	}
}

// TestLayoutMetricsPixelHeightFromLines calculates correct pixel height from lines.
// This tests the fix for the TODO in col.go:394 about variable font heights.
func TestLayoutMetricsPixelHeightFromLines(t *testing.T) {
	tests := []struct {
		name       string
		tagHeight  int
		bodyHeight int
		tagLines   int
		bodyLines  int
		want       int
	}{
		// When tag and body have same height, behaves like simple multiplication
		{"same heights", 16, 16, 2, 10, 2*16 + 10*16},
		// When tag is taller, tag lines contribute more pixels
		{"tag taller", 20, 14, 2, 10, 2*20 + 10*14},
		// When body is taller, body lines contribute more pixels
		{"body taller", 14, 20, 2, 10, 2*14 + 10*20},
		// Single line tag
		{"single tag line", 18, 14, 1, 5, 18 + 5*14},
		// No body lines (collapsed window)
		{"no body lines", 16, 14, 2, 0, 2 * 16},
		// Multi-line tag (expanded)
		{"expanded tag", 16, 14, 5, 10, 5*16 + 10*14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutMetrics(tt.tagHeight, tt.bodyHeight)
			got := lm.PixelHeightFromLines(tt.tagLines, tt.bodyLines)
			if got != tt.want {
				t.Errorf("PixelHeightFromLines(%d, %d) = %d; want %d",
					tt.tagLines, tt.bodyLines, got, tt.want)
			}
		})
	}
}

// TestLayoutMetricsBodyLinesFromPixelHeight converts pixel height to body lines.
// This addresses the col.go:481 TODO about using correct frame font height.
func TestLayoutMetricsBodyLinesFromPixelHeight(t *testing.T) {
	tests := []struct {
		name       string
		tagHeight  int
		bodyHeight int
		tagLines   int
		pixelHeight int
		wantBody   int
	}{
		// With 1 tag line (16px), remaining 84px = 6 body lines (14px each)
		{"basic", 16, 14, 1, 100, 6},
		// With 2 tag lines (32px), remaining 68px = 4 body lines
		{"multi-tag", 16, 14, 2, 100, 4},
		// Different font heights - 1 tag (20px), remaining 80px = 4 body lines (20px each)
		{"larger body font", 20, 20, 1, 100, 4},
		// Zero remaining height after tag
		{"no room for body", 20, 14, 5, 100, 0},
		// Large body font, 1 tag line
		{"large body font", 14, 25, 1, 100, 3}, // (100-14)/25 = 3.44 -> 3
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutMetrics(tt.tagHeight, tt.bodyHeight)
			got := lm.BodyLinesFromPixelHeight(tt.tagLines, tt.pixelHeight)
			if got != tt.wantBody {
				t.Errorf("BodyLinesFromPixelHeight(%d, %d) = %d; want %d",
					tt.tagLines, tt.pixelHeight, got, tt.wantBody)
			}
		})
	}
}

// TestLayoutMetricsProportionalResize handles resize calculations.
// This tests the logic needed for col.go:399-403 where lines are distributed.
func TestLayoutMetricsProportionalResize(t *testing.T) {
	tests := []struct {
		name            string
		tagHeight       int
		bodyHeight      int
		tagLines        int
		currentBodyLines int
		availableHeight int
		wantBodyLines   int
	}{
		// Shrink: window with 10 body lines, new space only fits 5
		{"shrink", 16, 14, 1, 10, 16 + 5*14, 5},
		// Grow: window with 5 body lines, new space fits 10
		{"grow", 16, 14, 1, 5, 16 + 10*14, 10},
		// Different tag/body heights: 2 tag lines (20px each = 40px), remaining 60px = 3 body lines (20px each)
		{"different heights", 20, 20, 2, 5, 100, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutMetrics(tt.tagHeight, tt.bodyHeight)
			got := lm.BodyLinesFromPixelHeight(tt.tagLines, tt.availableHeight)
			if got != tt.wantBodyLines {
				t.Errorf("BodyLinesFromPixelHeight(%d, %d) = %d; want %d",
					tt.tagLines, tt.availableHeight, got, tt.wantBodyLines)
			}
		})
	}
}

// TestLayoutMetricsTotalPixelHeight computes total window height including border.
// This is useful for col.go packColumn which needs to compute complete window sizes.
func TestLayoutMetricsTotalPixelHeight(t *testing.T) {
	tests := []struct {
		name       string
		tagHeight  int
		bodyHeight int
		tagLines   int
		bodyLines  int
		border     int
		separator  int
		want       int
	}{
		// Standard window: 1 tag line + border + body lines + separator
		{"standard", 16, 14, 1, 10, 2, 1, 16 + 2 + 10*14 + 1},
		// Multi-line tag
		{"multi-tag", 18, 14, 3, 5, 2, 1, 3*18 + 2 + 5*14 + 1},
		// Different fonts
		{"different fonts", 20, 16, 2, 8, 2, 1, 2*20 + 2 + 8*16 + 1},
		// Minimal window (1 tag, 1 body line)
		{"minimal", 16, 14, 1, 1, 2, 1, 16 + 2 + 14 + 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutMetrics(tt.tagHeight, tt.bodyHeight)
			got := lm.TotalPixelHeight(tt.tagLines, tt.bodyLines, tt.border, tt.separator)
			if got != tt.want {
				t.Errorf("TotalPixelHeight(%d, %d, %d, %d) = %d; want %d",
					tt.tagLines, tt.bodyLines, tt.border, tt.separator, got, tt.want)
			}
		})
	}
}

// TestLayoutMetricsLinesTotalFromPixels converts total pixel height back to line count.
// This addresses the col.go:399-403 TODO where the code incorrectly adds taglines-1 to maxlines.
func TestLayoutMetricsLinesTotalFromPixels(t *testing.T) {
	tests := []struct {
		name       string
		tagHeight  int
		bodyHeight int
		tagLines   int
		totalPixels int
		border     int
		separator  int
		wantBody   int
	}{
		// 100px total - 16px tag - 2 border - 1 sep = 81px for body = 5 lines (14px each)
		{"standard", 16, 14, 1, 100, 2, 1, 5},
		// Multi-line tag: 100px - 48px (3*16) - 2 - 1 = 49px = 3 body lines
		{"multi-tag", 16, 14, 3, 100, 2, 1, 3},
		// Different fonts: 100px - 40px (2*20) - 2 - 1 = 57px = 3 body lines (16px each)
		{"different fonts", 20, 16, 2, 100, 2, 1, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutMetrics(tt.tagHeight, tt.bodyHeight)
			got := lm.BodyLinesFromTotalPixels(tt.tagLines, tt.totalPixels, tt.border, tt.separator)
			if got != tt.wantBody {
				t.Errorf("BodyLinesFromTotalPixels(%d, %d, %d, %d) = %d; want %d",
					tt.tagLines, tt.totalPixels, tt.border, tt.separator, got, tt.wantBody)
			}
		})
	}
}

// TestLayoutMetricsEffectiveLines converts tag+body lines to a common unit.
// This is needed for col.go:399-403 where the code needs to sum lines across windows
// with potentially different font heights for distribution calculations.
func TestLayoutMetricsEffectiveLines(t *testing.T) {
	tests := []struct {
		name       string
		tagHeight  int
		bodyHeight int
		tagLines   int
		bodyLines  int
		want       int // effective lines in body-line units
	}{
		// Same heights: 2 tag + 10 body = 12 effective lines
		{"same heights", 16, 16, 2, 10, 12},
		// Tag taller (20px vs 10px): 2 tag lines = 4 body-equivalent, total = 14
		{"tag taller", 20, 10, 2, 10, 14},
		// Body taller (10px vs 20px): 4 tag lines = 2 body-equivalent, total = 12
		{"body taller", 10, 20, 4, 10, 12},
		// Single tag line
		{"single tag", 16, 14, 1, 10, 11}, // 16/14 = 1.14 -> 1, total 11
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLayoutMetrics(tt.tagHeight, tt.bodyHeight)
			got := lm.TotalLinesEquivalent(tt.tagLines, tt.bodyLines)
			if got != tt.want {
				t.Errorf("TotalLinesEquivalent(%d, %d) = %d; want %d",
					tt.tagLines, tt.bodyLines, got, tt.want)
			}
		})
	}
}

// TestLayoutMetricsZeroDivisionNewMethods tests new methods don't panic with zero heights.
func TestLayoutMetricsZeroDivisionNewMethods(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("function panicked: %v", r)
		}
	}()

	lm := NewLayoutMetrics(0, 0)

	// These should return 0 or handle gracefully, not panic
	if got := lm.PixelHeightFromLines(2, 10); got != 0 {
		t.Errorf("PixelHeightFromLines with zero heights = %d; want 0", got)
	}
	if got := lm.BodyLinesFromPixelHeight(1, 100); got != 0 {
		t.Errorf("BodyLinesFromPixelHeight with zero body height = %d; want 0", got)
	}
	if got := lm.TotalPixelHeight(1, 10, 2, 1); got != 3 {
		t.Errorf("TotalPixelHeight with zero heights = %d; want 3 (just border+separator)", got)
	}
	if got := lm.BodyLinesFromTotalPixels(1, 100, 2, 1); got != 0 {
		t.Errorf("BodyLinesFromTotalPixels with zero heights = %d; want 0", got)
	}
}
