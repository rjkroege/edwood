package markdown

import (
	"testing"
)

// TestSourceMapSimple tests source mapping for plain text (1:1 mapping).
func TestSourceMapSimple(t *testing.T) {
	input := "Hello, World!"
	_, sm := ParseWithSourceMap(input)

	// Plain text should have 1:1 mapping
	// Rendered position 0 should map to source position 0
	srcStart, srcEnd := sm.ToSource(0, 1)
	if srcStart != 0 || srcEnd != 1 {
		t.Errorf("ToSource(0, 1) = (%d, %d), want (0, 1)", srcStart, srcEnd)
	}

	// Rendered position 5 should map to source position 5
	srcStart, srcEnd = sm.ToSource(5, 6)
	if srcStart != 5 || srcEnd != 6 {
		t.Errorf("ToSource(5, 6) = (%d, %d), want (5, 6)", srcStart, srcEnd)
	}

	// Entire string
	srcStart, srcEnd = sm.ToSource(0, 13)
	if srcStart != 0 || srcEnd != 13 {
		t.Errorf("ToSource(0, 13) = (%d, %d), want (0, 13)", srcStart, srcEnd)
	}
}

// TestSourceMapBold tests source mapping for bold text (**text**).
// The rendered text "bold" (4 chars) maps to source "**bold**" (8 chars).
func TestSourceMapBold(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		renderedPos  int
		renderedEnd  int
		wantSrcStart int
		wantSrcEnd   int
	}{
		{
			name:         "just bold text",
			input:        "**bold**",
			renderedPos:  0,
			renderedEnd:  4, // "bold"
			wantSrcStart: 0,
			wantSrcEnd:   8, // "**bold**"
		},
		{
			name:         "bold at start of line",
			input:        "**bold** text",
			renderedPos:  0,
			renderedEnd:  4, // "bold"
			wantSrcStart: 0,
			wantSrcEnd:   8, // "**bold**"
		},
		{
			name:         "plain after bold",
			input:        "**bold** text",
			renderedPos:  4,
			renderedEnd:  9, // " text"
			wantSrcStart: 8,
			wantSrcEnd:   13, // " text"
		},
		{
			name:         "bold in middle",
			input:        "some **bold** text",
			renderedPos:  5,
			renderedEnd:  9, // "bold"
			wantSrcStart: 5,
			wantSrcEnd:   13, // "**bold**"
		},
		{
			name:         "text before bold",
			input:        "some **bold** text",
			renderedPos:  0,
			renderedEnd:  5, // "some "
			wantSrcStart: 0,
			wantSrcEnd:   5, // "some "
		},
		{
			name:         "text after bold",
			input:        "some **bold** text",
			renderedPos:  9,
			renderedEnd:  14, // " text"
			wantSrcStart: 13,
			wantSrcEnd:   18, // " text"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm := ParseWithSourceMap(tt.input)
			srcStart, srcEnd := sm.ToSource(tt.renderedPos, tt.renderedEnd)
			if srcStart != tt.wantSrcStart || srcEnd != tt.wantSrcEnd {
				t.Errorf("ToSource(%d, %d) = (%d, %d), want (%d, %d)",
					tt.renderedPos, tt.renderedEnd, srcStart, srcEnd,
					tt.wantSrcStart, tt.wantSrcEnd)
			}
		})
	}
}

// TestSourceMapHeading tests source mapping for headings (# text).
// The rendered text "Heading" (7 chars) maps to source "# Heading" (9 chars).
func TestSourceMapHeading(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		renderedPos  int
		renderedEnd  int
		wantSrcStart int
		wantSrcEnd   int
	}{
		{
			name:         "h1 heading",
			input:        "# Heading",
			renderedPos:  0,
			renderedEnd:  7, // "Heading"
			wantSrcStart: 0,
			wantSrcEnd:   9, // "# Heading"
		},
		{
			name:         "h2 heading",
			input:        "## Heading",
			renderedPos:  0,
			renderedEnd:  7, // "Heading"
			wantSrcStart: 0,
			wantSrcEnd:   10, // "## Heading"
		},
		{
			name:         "h3 heading",
			input:        "### Heading",
			renderedPos:  0,
			renderedEnd:  7, // "Heading"
			wantSrcStart: 0,
			wantSrcEnd:   11, // "### Heading"
		},
		{
			name:         "partial heading selection",
			input:        "# Heading",
			renderedPos:  0,
			renderedEnd:  4, // "Head"
			wantSrcStart: 0,
			wantSrcEnd:   6, // "# Head" - maps to start of heading through selected text
		},
		{
			name:         "heading with trailing newline",
			input:        "# Title\nBody",
			renderedPos:  0,
			renderedEnd:  5, // "Title"
			wantSrcStart: 0,
			wantSrcEnd:   7, // "# Title"
		},
		{
			name:         "body after heading",
			input:        "# Title\nBody",
			renderedPos:  6, // newline + Body start (rendered: "Title\nBody")
			renderedEnd:  10,
			wantSrcStart: 8,
			wantSrcEnd:   12, // "Body"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm := ParseWithSourceMap(tt.input)
			srcStart, srcEnd := sm.ToSource(tt.renderedPos, tt.renderedEnd)
			if srcStart != tt.wantSrcStart || srcEnd != tt.wantSrcEnd {
				t.Errorf("ToSource(%d, %d) = (%d, %d), want (%d, %d)",
					tt.renderedPos, tt.renderedEnd, srcStart, srcEnd,
					tt.wantSrcStart, tt.wantSrcEnd)
			}
		})
	}
}

// TestSourceMapItalic tests source mapping for italic text (*text*).
func TestSourceMapItalic(t *testing.T) {
	input := "*italic*"
	_, sm := ParseWithSourceMap(input)

	// Rendered "italic" (6 chars) maps to source "*italic*" (8 chars)
	srcStart, srcEnd := sm.ToSource(0, 6)
	if srcStart != 0 || srcEnd != 8 {
		t.Errorf("ToSource(0, 6) = (%d, %d), want (0, 8)", srcStart, srcEnd)
	}
}

// TestSourceMapMixed tests source mapping for mixed content.
func TestSourceMapMixed(t *testing.T) {
	// "# Title\nSome **bold** text\n"
	// Rendered: "Title\nSome bold text\n"
	input := "# Title\nSome **bold** text\n"
	_, sm := ParseWithSourceMap(input)

	// "Title" rendered at 0-5, source "# Title" at 0-7
	srcStart, srcEnd := sm.ToSource(0, 5)
	if srcStart != 0 || srcEnd != 7 {
		t.Errorf("Title: ToSource(0, 5) = (%d, %d), want (0, 7)", srcStart, srcEnd)
	}

	// "bold" rendered at 11-15, source "**bold**" at 13-21
	srcStart, srcEnd = sm.ToSource(11, 15)
	if srcStart != 13 || srcEnd != 21 {
		t.Errorf("bold: ToSource(11, 15) = (%d, %d), want (13, 21)", srcStart, srcEnd)
	}
}

// TestSourceMapCode tests source mapping for inline code (`code`).
func TestSourceMapCode(t *testing.T) {
	input := "`code`"
	_, sm := ParseWithSourceMap(input)

	// Rendered "code" (4 chars) maps to source "`code`" (6 chars)
	srcStart, srcEnd := sm.ToSource(0, 4)
	if srcStart != 0 || srcEnd != 6 {
		t.Errorf("ToSource(0, 4) = (%d, %d), want (0, 6)", srcStart, srcEnd)
	}
}

// TestSourceMapEmpty tests source mapping for empty input.
func TestSourceMapEmpty(t *testing.T) {
	input := ""
	_, sm := ParseWithSourceMap(input)

	// Empty input should return empty range
	srcStart, srcEnd := sm.ToSource(0, 0)
	if srcStart != 0 || srcEnd != 0 {
		t.Errorf("ToSource(0, 0) = (%d, %d), want (0, 0)", srcStart, srcEnd)
	}
}
