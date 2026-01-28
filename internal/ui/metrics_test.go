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
