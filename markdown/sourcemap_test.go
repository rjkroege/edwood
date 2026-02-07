package markdown

import (
	"strings"
	"testing"
)

// TestSourceMapSimple tests source mapping for plain text (1:1 mapping).
func TestSourceMapSimple(t *testing.T) {
	input := "Hello, World!"
	_, sm, _ := ParseWithSourceMap(input)

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
			_, sm, _ := ParseWithSourceMap(tt.input)
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
			_, sm, _ := ParseWithSourceMap(tt.input)
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
	_, sm, _ := ParseWithSourceMap(input)

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
	_, sm, _ := ParseWithSourceMap(input)

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
	_, sm, _ := ParseWithSourceMap(input)

	// Rendered "code" (4 chars) maps to source "`code`" (6 chars)
	srcStart, srcEnd := sm.ToSource(0, 4)
	if srcStart != 0 || srcEnd != 6 {
		t.Errorf("ToSource(0, 4) = (%d, %d), want (0, 6)", srcStart, srcEnd)
	}
}

// TestSourceMapEmpty tests source mapping for empty input.
func TestSourceMapEmpty(t *testing.T) {
	input := ""
	_, sm, _ := ParseWithSourceMap(input)

	// Empty input should return empty range
	srcStart, srcEnd := sm.ToSource(0, 0)
	if srcStart != 0 || srcEnd != 0 {
		t.Errorf("ToSource(0, 0) = (%d, %d), want (0, 0)", srcStart, srcEnd)
	}
}

// TestFencedCodeBlockSourceMap tests source mapping for fenced code blocks.
// The fence lines (``` and ```go etc.) are not rendered, so the source map
// must correctly map the rendered code content to the source position within
// the fences (excluding the fence lines themselves).
func TestFencedCodeBlockSourceMap(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		renderedPos  int
		renderedEnd  int
		wantSrcStart int
		wantSrcEnd   int
	}{
		{
			name: "simple fenced code block",
			// Source: "```\ncode\n```"
			// Positions: 0-3 = "```", 3 = "\n", 4-7 = "code", 8 = "\n", 9-11 = "```"
			// Rendered: "code\n"
			input:        "```\ncode\n```",
			renderedPos:  0,
			renderedEnd:  5, // "code\n"
			wantSrcStart: 4, // Start of "code" in source
			wantSrcEnd:   9, // End of "code\n" in source (before closing ```)
		},
		{
			name: "fenced code block with language",
			// Source: "```go\nfunc main() {}\n```"
			// Positions: 0-4 = "```go", 5 = "\n", 6-19 = "func main() {}", 20 = "\n", 21-23 = "```"
			input:        "```go\nfunc main() {}\n```",
			renderedPos:  0,
			renderedEnd:  15, // "func main() {}\n"
			wantSrcStart: 6,  // Start of "func" in source
			wantSrcEnd:   21, // End of "}\n" in source (before closing ```)
		},
		{
			name: "fenced code block partial selection",
			// Source: "```\nhello world\n```"
			// Rendered: "hello world\n"
			input:        "```\nhello world\n```",
			renderedPos:  0,
			renderedEnd:  5, // "hello" only
			wantSrcStart: 4, // Start of "hello" in source
			wantSrcEnd:   9, // End of "hello" in source
		},
		{
			name: "fenced code block in middle of text",
			// Source: "Before\n```\ncode\n```\nAfter"
			// Positions: 0-5 = "Before", 6 = "\n", 7-9 = "```", 10 = "\n", 11-14 = "code", 15 = "\n", 16-18 = "```", 19 = "\n", 20-24 = "After"
			// Rendered: "Before\ncode\nAfter"
			input:        "Before\n```\ncode\n```\nAfter",
			renderedPos:  7,  // Start of "code" in rendered
			renderedEnd:  12, // "code\n"
			wantSrcStart: 11, // Start of "code" in source
			wantSrcEnd:   16, // End of "code\n" in source
		},
		{
			name: "text before fenced code block",
			// Source: "Before\n```\ncode\n```\nAfter"
			// Rendered: "Before\ncode\nAfter"
			input:        "Before\n```\ncode\n```\nAfter",
			renderedPos:  0,
			renderedEnd:  7, // "Before\n"
			wantSrcStart: 0,
			wantSrcEnd:   7, // "Before\n" maps 1:1
		},
		{
			name: "text after fenced code block",
			// Source: "Before\n```\ncode\n```\nAfter"
			// Rendered: "Before\ncode\nAfter"
			input:        "Before\n```\ncode\n```\nAfter",
			renderedPos:  12, // Start of "After" in rendered
			renderedEnd:  17, // End of "After"
			wantSrcStart: 20, // Start of "After" in source
			wantSrcEnd:   25, // End of "After" in source
		},
		{
			name: "multiline fenced code block",
			// Source: "```\nline1\nline2\n```"
			// Rendered: "line1\nline2\n"
			input:        "```\nline1\nline2\n```",
			renderedPos:  0,
			renderedEnd:  12, // "line1\nline2\n"
			wantSrcStart: 4,  // Start of "line1" in source
			wantSrcEnd:   16, // End of "line2\n" in source
		},
		{
			name: "select second line of code block",
			// Source: "```\nline1\nline2\n```"
			// Rendered: "line1\nline2\n"
			input:        "```\nline1\nline2\n```",
			renderedPos:  6,  // Start of "line2" in rendered
			renderedEnd:  12, // "line2\n"
			wantSrcStart: 10, // Start of "line2" in source
			wantSrcEnd:   16, // End of "line2\n" in source
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm, _ := ParseWithSourceMap(tt.input)
			srcStart, srcEnd := sm.ToSource(tt.renderedPos, tt.renderedEnd)
			if srcStart != tt.wantSrcStart || srcEnd != tt.wantSrcEnd {
				t.Errorf("ToSource(%d, %d) = (%d, %d), want (%d, %d)",
					tt.renderedPos, tt.renderedEnd, srcStart, srcEnd,
					tt.wantSrcStart, tt.wantSrcEnd)
			}
		})
	}
}

// TestHorizontalRuleSourceMap tests source mapping for horizontal rules (---, ***, ___).
// A horizontal rule like "---" (3 chars) renders as HRuleRune + "\n" (2 runes).
// The source map should map the rendered HRuleRune position back to the full source line.
func TestHorizontalRuleSourceMap(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		renderedPos  int
		renderedEnd  int
		wantSrcStart int
		wantSrcEnd   int
	}{
		{
			name: "simple hrule with hyphens",
			// Source: "---\n" (4 bytes)
			// Rendered: HRuleRune + "\n" (2 runes, positions 0-1)
			input:        "---\n",
			renderedPos:  0,
			renderedEnd:  2, // HRuleRune + newline
			wantSrcStart: 0,
			wantSrcEnd:   4, // "---\n"
		},
		{
			name: "hrule without trailing newline",
			// Source: "---" (3 bytes)
			// Rendered: HRuleRune (1 rune, position 0)
			input:        "---",
			renderedPos:  0,
			renderedEnd:  1, // just HRuleRune
			wantSrcStart: 0,
			wantSrcEnd:   3, // "---"
		},
		{
			name: "hrule with asterisks",
			// Source: "***\n" (4 bytes)
			// Rendered: HRuleRune + "\n" (2 runes)
			input:        "***\n",
			renderedPos:  0,
			renderedEnd:  2,
			wantSrcStart: 0,
			wantSrcEnd:   4,
		},
		{
			name: "hrule with underscores",
			// Source: "___\n" (4 bytes)
			// Rendered: HRuleRune + "\n" (2 runes)
			input:        "___\n",
			renderedPos:  0,
			renderedEnd:  2,
			wantSrcStart: 0,
			wantSrcEnd:   4,
		},
		{
			name: "longer hrule",
			// Source: "----------\n" (11 bytes)
			// Rendered: HRuleRune + "\n" (2 runes)
			input:        "----------\n",
			renderedPos:  0,
			renderedEnd:  2,
			wantSrcStart: 0,
			wantSrcEnd:   11,
		},
		{
			name: "hrule between text",
			// Source: "Above\n---\nBelow" (15 bytes)
			// Positions: 0-5 = "Above\n", 6-9 = "---\n", 10-14 = "Below"
			// Rendered: "Above\n" + HRuleRune + "\n" + "Below" (14 runes)
			// Positions: 0-5 = "Above\n", 6 = HRuleRune, 7 = "\n", 8-12 = "Below"
			input:        "Above\n---\nBelow",
			renderedPos:  6,
			renderedEnd:  8, // HRuleRune + "\n"
			wantSrcStart: 6,
			wantSrcEnd:   10, // "---\n"
		},
		{
			name: "text before hrule",
			// Source: "Above\n---\nBelow"
			// Rendered: "Above\n" + HRuleRune + "\n" + "Below"
			input:        "Above\n---\nBelow",
			renderedPos:  0,
			renderedEnd:  6, // "Above\n"
			wantSrcStart: 0,
			wantSrcEnd:   6, // "Above\n" maps 1:1
		},
		{
			name: "text after hrule",
			// Source: "Above\n---\nBelow"
			// Rendered: "Above\n" + HRuleRune + "\n" + "Below"
			input:        "Above\n---\nBelow",
			renderedPos:  8,  // Start of "Below" in rendered
			renderedEnd:  13, // End of "Below"
			wantSrcStart: 10, // Start of "Below" in source
			wantSrcEnd:   15, // End of "Below" in source
		},
		{
			name: "select just hrule marker",
			// Source: "---\n"
			// Rendered: HRuleRune + "\n"
			// Select just the HRuleRune (position 0)
			input:        "---\n",
			renderedPos:  0,
			renderedEnd:  1, // just HRuleRune
			wantSrcStart: 0,
			wantSrcEnd:   3, // "---" (without newline since we didn't select it)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm, _ := ParseWithSourceMap(tt.input)
			srcStart, srcEnd := sm.ToSource(tt.renderedPos, tt.renderedEnd)
			if srcStart != tt.wantSrcStart || srcEnd != tt.wantSrcEnd {
				t.Errorf("ToSource(%d, %d) = (%d, %d), want (%d, %d)",
					tt.renderedPos, tt.renderedEnd, srcStart, srcEnd,
					tt.wantSrcStart, tt.wantSrcEnd)
			}
		})
	}
}

// TestParseWithSourceMapLinks tests that ParseWithSourceMap returns a LinkMap
// that correctly tracks link positions in the rendered content.
func TestParseWithSourceMapLinks(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLinks []struct {
			pos     int    // Position within the link to test
			wantURL string // Expected URL at that position
		}
		noLinks []int // Positions that should NOT be in a link
	}{
		{
			name:  "simple link",
			input: "[click here](https://example.com)",
			// Rendered: "click here" (10 chars, positions 0-9)
			wantLinks: []struct {
				pos     int
				wantURL string
			}{
				{pos: 0, wantURL: "https://example.com"},
				{pos: 5, wantURL: "https://example.com"},
				{pos: 9, wantURL: "https://example.com"},
			},
			noLinks: []int{10, 15}, // Outside link
		},
		{
			name:  "link in middle of text",
			input: "See [this link](https://example.com) for details.",
			// Rendered: "See this link for details." (27 chars)
			// "See " = 0-3, "this link" = 4-12, " for details." = 13-26
			wantLinks: []struct {
				pos     int
				wantURL string
			}{
				{pos: 4, wantURL: "https://example.com"},
				{pos: 8, wantURL: "https://example.com"},
				{pos: 12, wantURL: "https://example.com"},
			},
			noLinks: []int{0, 3, 13, 20},
		},
		{
			name:  "multiple links",
			input: "[one](https://one.com) and [two](https://two.com)",
			// Rendered: "one and two" (11 chars)
			// "one" = 0-2, " and " = 3-7, "two" = 8-10
			wantLinks: []struct {
				pos     int
				wantURL string
			}{
				{pos: 0, wantURL: "https://one.com"},
				{pos: 2, wantURL: "https://one.com"},
				{pos: 8, wantURL: "https://two.com"},
				{pos: 10, wantURL: "https://two.com"},
			},
			noLinks: []int{3, 5, 7, 11},
		},
		{
			name:  "no links",
			input: "Just plain text",
			wantLinks: []struct {
				pos     int
				wantURL string
			}{},
			noLinks: []int{0, 5, 10},
		},
		{
			name:  "link with bold text",
			input: "[**bold link**](https://example.com)",
			// Rendered: "bold link" (9 chars, positions 0-8)
			wantLinks: []struct {
				pos     int
				wantURL string
			}{
				{pos: 0, wantURL: "https://example.com"},
				{pos: 4, wantURL: "https://example.com"},
				{pos: 8, wantURL: "https://example.com"},
			},
			noLinks: []int{9, 15},
		},
		{
			name:  "adjacent links",
			input: "[a](url1)[b](url2)",
			// Rendered: "ab" (2 chars)
			// "a" = 0, "b" = 1
			wantLinks: []struct {
				pos     int
				wantURL string
			}{
				{pos: 0, wantURL: "url1"},
				{pos: 1, wantURL: "url2"},
			},
			noLinks: []int{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, lm := ParseWithSourceMap(tt.input)
			if lm == nil {
				t.Fatal("ParseWithSourceMap returned nil LinkMap")
			}

			// Check positions that should be in links
			for _, want := range tt.wantLinks {
				gotURL := lm.URLAt(want.pos)
				if gotURL != want.wantURL {
					t.Errorf("URLAt(%d) = %q, want %q", want.pos, gotURL, want.wantURL)
				}
			}

			// Check positions that should NOT be in links
			for _, pos := range tt.noLinks {
				gotURL := lm.URLAt(pos)
				if gotURL != "" {
					t.Errorf("URLAt(%d) = %q, want empty string (not in link)", pos, gotURL)
				}
			}
		})
	}
}

// TestFencedCodeBlockHasBlockFlag verifies that fenced code blocks from
// ParseWithSourceMap have the Block flag set to true.
func TestFencedCodeBlockHasBlockFlag(t *testing.T) {
	content, _, _ := ParseWithSourceMap("```\ncode\n```")

	if len(content) != 1 {
		t.Fatalf("got %d spans, want 1", len(content))
	}

	span := content[0]
	if !span.Style.Code {
		t.Error("span.Style.Code = false, want true")
	}
	if !span.Style.Block {
		t.Error("span.Style.Block = false, want true for fenced code blocks")
	}
	if span.Style.Bg == nil {
		t.Error("span.Style.Bg is nil, want background color")
	}
}

// TestListSourceMap tests source mapping for list items (- item, 1. item).
// List items render as "• content\n" or "1. content\n" where the bullet/number
// replaces the original marker, but the content position should map correctly.
func TestListSourceMap(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		renderedPos  int
		renderedEnd  int
		wantSrcStart int
		wantSrcEnd   int
	}{
		{
			name: "simple unordered list item",
			// Source: "- item\n" (7 bytes)
			// Rendered: "• item\n" (7 runes: bullet + space + "item" + newline)
			// The bullet "•" (pos 0) maps to "-" in source (pos 0-1)
			// The space (pos 1) maps to space (pos 1-2)
			// "item" (pos 2-5) maps to "item" (pos 2-6)
			// newline (pos 6) maps to newline (pos 6-7)
			input:        "- item\n",
			renderedPos:  0,
			renderedEnd:  7, // "• item\n"
			wantSrcStart: 0,
			wantSrcEnd:   7, // "- item\n"
		},
		{
			name: "unordered list bullet only",
			// Select just the bullet "•"
			input:        "- item\n",
			renderedPos:  0,
			renderedEnd:  1, // just "•"
			wantSrcStart: 0,
			wantSrcEnd:   1, // just "-"
		},
		{
			name: "unordered list content only",
			// Select just "item" (without bullet/space/newline)
			input:        "- item\n",
			renderedPos:  2,
			renderedEnd:  6, // "item"
			wantSrcStart: 2,
			wantSrcEnd:   6, // "item"
		},
		{
			name: "simple ordered list item",
			// Source: "1. item\n" (8 bytes)
			// Rendered: "1. item\n" (8 runes: "1." + space + "item" + newline)
			input:        "1. item\n",
			renderedPos:  0,
			renderedEnd:  8, // "1. item\n"
			wantSrcStart: 0,
			wantSrcEnd:   8, // "1. item\n"
		},
		{
			name: "ordered list number only",
			// Select just "1."
			input:        "1. item\n",
			renderedPos:  0,
			renderedEnd:  2, // "1."
			wantSrcStart: 0,
			wantSrcEnd:   2, // "1."
		},
		{
			name: "ordered list content only",
			// Select just "item" (after "1. ")
			input:        "1. item\n",
			renderedPos:  3,
			renderedEnd:  7, // "item"
			wantSrcStart: 3,
			wantSrcEnd:   7, // "item"
		},
		{
			name: "nested unordered list item",
			// Source: "  - nested\n" (11 bytes, 2-space indent)
			// Rendered: "• nested\n" (9 runes) - indent handled in layout, not rendered text
			input:        "  - nested\n",
			renderedPos:  0,
			renderedEnd:  9, // "• nested\n"
			wantSrcStart: 0,
			wantSrcEnd:   11, // "  - nested\n"
		},
		{
			name: "list item with bold",
			// Source: "- **bold** text\n" (16 bytes)
			// Rendered: "• bold text\n" (12 runes)
			input:        "- **bold** text\n",
			renderedPos:  2,
			renderedEnd:  6, // "bold"
			wantSrcStart: 2,
			wantSrcEnd:   10, // "**bold**"
		},
		{
			name: "multiple list items - first item",
			// Source: "- one\n- two\n" (12 bytes)
			// Rendered: "• one\n• two\n" (12 runes)
			input:        "- one\n- two\n",
			renderedPos:  0,
			renderedEnd:  6, // "• one\n"
			wantSrcStart: 0,
			wantSrcEnd:   6, // "- one\n"
		},
		{
			name: "multiple list items - second item",
			// Source: "- one\n- two\n" (12 bytes)
			// Rendered: "• one\n• two\n" (12 runes)
			input:        "- one\n- two\n",
			renderedPos:  6,
			renderedEnd:  12, // "• two\n"
			wantSrcStart: 6,
			wantSrcEnd:   12, // "- two\n"
		},
		{
			name: "text before list",
			// Source: "Intro\n- item\n" (13 bytes)
			// Rendered: "Intro\n• item\n" (13 runes)
			input:        "Intro\n- item\n",
			renderedPos:  0,
			renderedEnd:  6, // "Intro\n"
			wantSrcStart: 0,
			wantSrcEnd:   6, // "Intro\n"
		},
		{
			name: "list between text",
			// Source: "Before\n- item\nAfter" (19 bytes)
			// Rendered: "Before\n• item\nAfter" (19 runes)
			input:        "Before\n- item\nAfter",
			renderedPos:  7,
			renderedEnd:  14, // "• item\n"
			wantSrcStart: 7,
			wantSrcEnd:   14, // "- item\n"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm, _ := ParseWithSourceMap(tt.input)
			srcStart, srcEnd := sm.ToSource(tt.renderedPos, tt.renderedEnd)
			if srcStart != tt.wantSrcStart || srcEnd != tt.wantSrcEnd {
				t.Errorf("ToSource(%d, %d) = (%d, %d), want (%d, %d)",
					tt.renderedPos, tt.renderedEnd, srcStart, srcEnd,
					tt.wantSrcStart, tt.wantSrcEnd)
			}
		})
	}
}

// TestSourceMapToRendered tests the reverse mapping: given source rune positions,
// find the corresponding rendered rune positions. This is the inverse of ToSource().
func TestSourceMapToRendered(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		srcRuneStart  int
		srcRuneEnd    int
		wantRendStart int
		wantRendEnd   int
	}{
		{
			name:          "plain text 1:1 mapping",
			input:         "Hello, World!",
			srcRuneStart:  0,
			srcRuneEnd:    5, // "Hello"
			wantRendStart: 0,
			wantRendEnd:   5,
		},
		{
			name:          "plain text middle",
			input:         "Hello, World!",
			srcRuneStart:  7,
			srcRuneEnd:    12, // "World"
			wantRendStart: 7,
			wantRendEnd:   12,
		},
		{
			name:          "bold text - full source range",
			input:         "**bold**",
			// Source: "**bold**" (8 runes), rendered: "bold" (4 runes)
			srcRuneStart:  0,
			srcRuneEnd:    8,
			wantRendStart: 0,
			wantRendEnd:   4,
		},
		{
			name:          "bold text - inner content only",
			input:         "**bold**",
			// Source rune positions 2-6 = "bold" inside the markers
			srcRuneStart:  2,
			srcRuneEnd:    6,
			wantRendStart: 0,
			wantRendEnd:   4,
		},
		{
			name:          "bold in middle of text - select bold source",
			input:         "some **bold** text",
			// Source: "some **bold** text" (18 runes)
			// Rendered: "some bold text" (14 runes)
			// "**bold**" is at source positions 5-13
			srcRuneStart:  5,
			srcRuneEnd:    13,
			wantRendStart: 5,
			wantRendEnd:   9, // "bold" in rendered
		},
		{
			name:          "text after bold",
			input:         "some **bold** text",
			// " text" is at source positions 13-18
			// In rendered, " text" is at positions 9-14
			srcRuneStart:  13,
			srcRuneEnd:    18,
			wantRendStart: 9,
			wantRendEnd:   14,
		},
		{
			name:          "heading - full source",
			input:         "# Title",
			// Source: "# Title" (7 runes), rendered: "Title" (5 runes)
			srcRuneStart:  0,
			srcRuneEnd:    7,
			wantRendStart: 0,
			wantRendEnd:   5,
		},
		{
			name:          "heading - content only",
			input:         "# Title",
			// "Title" starts at source rune 2 (after "# ")
			srcRuneStart:  2,
			srcRuneEnd:    7,
			wantRendStart: 0,
			wantRendEnd:   5,
		},
		{
			name:          "italic text",
			input:         "*italic*",
			// Source: "*italic*" (8 runes), rendered: "italic" (6 runes)
			srcRuneStart:  0,
			srcRuneEnd:    8,
			wantRendStart: 0,
			wantRendEnd:   6,
		},
		{
			name:          "inline code",
			input:         "`code`",
			// Source: "`code`" (6 runes), rendered: "code" (4 runes)
			srcRuneStart:  0,
			srcRuneEnd:    6,
			wantRendStart: 0,
			wantRendEnd:   4,
		},
		{
			name:          "fenced code block - code content",
			input:         "```\ncode\n```",
			// Source: "```\ncode\n```" - "code\n" starts at byte 4, rune 4
			// Rendered: "code\n" (5 runes)
			srcRuneStart:  4,
			srcRuneEnd:    9, // "code\n"
			wantRendStart: 0,
			wantRendEnd:   5,
		},
		{
			name:          "text before and after code block",
			input:         "Before\n```\ncode\n```\nAfter",
			// Rendered: "Before\ncode\nAfter" (17 runes)
			// "After" in source starts at rune 20
			srcRuneStart:  20,
			srcRuneEnd:    25, // "After"
			wantRendStart: 12,
			wantRendEnd:   17,
		},
		{
			name:          "no mapping found returns -1,-1",
			input:         "",
			srcRuneStart:  5,
			srcRuneEnd:    10,
			wantRendStart: -1,
			wantRendEnd:   -1,
		},
		{
			name:          "bold+italic",
			input:         "***bolditalic***",
			// Source: 16 runes, rendered: "bolditalic" (10 runes)
			srcRuneStart:  0,
			srcRuneEnd:    16,
			wantRendStart: 0,
			wantRendEnd:   10,
		},
		{
			name:          "mixed content - select bold in second line",
			input:         "# Title\nSome **bold** text\n",
			// Rendered: "Title\nSome bold text\n" (21 runes)
			// "**bold**" in source at runes 13-21
			srcRuneStart:  13,
			srcRuneEnd:    21,
			wantRendStart: 11,
			wantRendEnd:   15, // "bold" in rendered
		},
		{
			name:          "unordered list - content",
			input:         "- item\n",
			// Source: "- item\n" (7 runes)
			// Rendered: "• item\n" (7 runes)
			// "item" at source runes 2-6
			srcRuneStart:  2,
			srcRuneEnd:    6,
			wantRendStart: 2,
			wantRendEnd:   6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm, _ := ParseWithSourceMap(tt.input)
			rendStart, rendEnd := sm.ToRendered(tt.srcRuneStart, tt.srcRuneEnd)
			if rendStart != tt.wantRendStart || rendEnd != tt.wantRendEnd {
				t.Errorf("ToRendered(%d, %d) = (%d, %d), want (%d, %d)",
					tt.srcRuneStart, tt.srcRuneEnd, rendStart, rendEnd,
					tt.wantRendStart, tt.wantRendEnd)
			}
		})
	}
}

// TestSourceMapToRenderedRoundTrip verifies that mapping rendered→source→rendered
// produces the original rendered positions (or expanded ones for formatted elements).
// This tests the consistency of ToSource() and ToRendered() as inverse operations.
func TestSourceMapToRenderedRoundTrip(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		renderedStart int
		renderedEnd   int
	}{
		{
			name:          "plain text round trip",
			input:         "Hello, World!",
			renderedStart: 0,
			renderedEnd:   5,
		},
		{
			name:          "bold text round trip",
			input:         "**bold**",
			renderedStart: 0,
			renderedEnd:   4, // "bold"
		},
		{
			name:          "italic text round trip",
			input:         "*italic*",
			renderedStart: 0,
			renderedEnd:   6, // "italic"
		},
		{
			name:          "heading round trip",
			input:         "# Title",
			renderedStart: 0,
			renderedEnd:   5, // "Title"
		},
		{
			name:          "inline code round trip",
			input:         "some `code` here",
			renderedStart: 5,
			renderedEnd:   9, // "code"
		},
		{
			name:          "bold in middle round trip",
			input:         "some **bold** text",
			renderedStart: 5,
			renderedEnd:   9, // "bold"
		},
		{
			name:          "plain text in mixed content",
			input:         "some **bold** text",
			renderedStart: 0,
			renderedEnd:   5, // "some "
		},
		{
			name:          "fenced code block content",
			input:         "```\nhello\n```",
			renderedStart: 0,
			renderedEnd:   6, // "hello\n"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm, _ := ParseWithSourceMap(tt.input)

			// Step 1: rendered → source (byte positions)
			srcStart, srcEnd := sm.ToSource(tt.renderedStart, tt.renderedEnd)

			// Step 2: source → rendered (round trip)
			// ToSource returns byte positions but ToRendered accepts rune positions.
			// Convert byte→rune using the source text.
			srcRuneStart := byteToRunePos(tt.input, srcStart)
			srcRuneEnd := byteToRunePos(tt.input, srcEnd)

			rendStart, rendEnd := sm.ToRendered(srcRuneStart, srcRuneEnd)

			// The round trip should produce positions that contain the original selection.
			// For formatted elements, the result may be equal or expanded (since
			// ToSource expands to include markers, and ToRendered maps back to
			// the full rendered element).
			if rendStart > tt.renderedStart || rendEnd < tt.renderedEnd {
				t.Errorf("round trip failed: rendered(%d,%d) → source(%d,%d) [runes: %d,%d] → rendered(%d,%d); want rendered to contain [%d,%d]",
					tt.renderedStart, tt.renderedEnd,
					srcStart, srcEnd,
					srcRuneStart, srcRuneEnd,
					rendStart, rendEnd,
					tt.renderedStart, tt.renderedEnd)
			}
		})
	}
}

// TestSourceMapPointSelectionHeading verifies that a point selection (q0==q1)
// in rendered content maps to a point selection in source, not a range.
// Regression test for the bug where clicking in a heading in preview mode
// produced a 3-character selection (spanning "## ") when exiting Markdeep.
func TestSourceMapPointSelectionHeading(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		renderedPos int
	}{
		{"h1 start", "# Hello", 0},
		{"h1 middle", "# Hello", 3},
		{"h2 start", "## Hello", 0},
		{"h2 middle", "## Hello", 2},
		{"h3 start", "### Hello", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm, _ := ParseWithSourceMap(tt.input)
			srcStart, srcEnd := sm.ToSource(tt.renderedPos, tt.renderedPos)
			if srcStart != srcEnd {
				t.Errorf("ToSource(%d, %d) = (%d, %d), want point selection (srcStart == srcEnd)",
					tt.renderedPos, tt.renderedPos, srcStart, srcEnd)
			}
		})
	}
}

// TestBlockquoteSourceMapToSource tests that ToSource correctly maps rendered
// positions back to source positions for blockquote content. The `> ` prefix
// is stripped from rendered output, so source map entries must account for it.
// This tests the "heading model" approach where PrefixLen records the prefix.
func TestBlockquoteSourceMapToSource(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		renderedPos  int
		renderedEnd  int
		wantSrcStart int
		wantSrcEnd   int
	}{
		{
			name: "simple blockquote - select all content",
			// Source: "> hello" (7 bytes/runes)
			// Prefix: "> " (2 runes), content: "hello" (5 runes)
			// Rendered: "hello" (5 runes, positions 0-4)
			// Selecting all rendered content should expand to include "> " prefix
			input:        "> hello",
			renderedPos:  0,
			renderedEnd:  5, // "hello"
			wantSrcStart: 0,
			wantSrcEnd:   7, // "> hello"
		},
		{
			name: "simple blockquote - partial selection",
			// Source: "> hello" (7 runes)
			// Rendered: "hello" (5 runes)
			// Selecting "hel" (positions 0-3) — partial, so no prefix expansion
			// Maps to source runes 2-5 ("hel" after "> " prefix)
			input:        "> hello",
			renderedPos:  0,
			renderedEnd:  3, // "hel"
			wantSrcStart: 0,
			wantSrcEnd:   5, // "> hel" - boundary expansion includes prefix at start
		},
		{
			name: "simple blockquote - middle selection",
			// Source: "> hello" (7 runes)
			// Rendered: "hello" (5 runes)
			// Selecting "ell" (rendered positions 1-4)
			input:        "> hello",
			renderedPos:  1,
			renderedEnd:  4, // "ell"
			wantSrcStart: 3,
			wantSrcEnd:   6, // "ell" in source (past "> h")
		},
		{
			name: "simple blockquote - point selection at start",
			// Source: "> hello" (7 runes)
			// Rendered: "hello"
			// Point click at rendered position 0 → should map to point in source
			input:        "> hello",
			renderedPos:  0,
			renderedEnd:  0, // point selection
			wantSrcStart: 2,
			wantSrcEnd:   2, // point at content start (after "> ")
		},
		{
			name: "nested blockquote depth 2 - select all",
			// Source: "> > inner" (9 runes)
			// Prefix: "> > " (4 runes), content: "inner" (5 runes)
			// Rendered: "inner" (5 runes)
			input:        "> > inner",
			renderedPos:  0,
			renderedEnd:  5, // "inner"
			wantSrcStart: 0,
			wantSrcEnd:   9, // "> > inner"
		},
		{
			name: "nested blockquote depth 3 - select all",
			// Source: "> > > deep" (10 runes)
			// Prefix: "> > > " (6 runes), content: "deep" (4 runes)
			// Rendered: "deep" (4 runes)
			input:        "> > > deep",
			renderedPos:  0,
			renderedEnd:  4, // "deep"
			wantSrcStart: 0,
			wantSrcEnd:   10, // "> > > deep"
		},
		{
			name: "blockquote with trailing newline",
			// Source: "> hello\n" (8 bytes)
			// Rendered: "hello\n" (6 runes)
			input:        "> hello\n",
			renderedPos:  0,
			renderedEnd:  6, // "hello\n"
			wantSrcStart: 0,
			wantSrcEnd:   8, // "> hello\n"
		},
		{
			name: "blockquote followed by paragraph",
			// Source: "> quote\n\nparagraph" (18 bytes)
			// Rendered: "quote\n" + "\n" (parabreak) + "paragraph"
			//          = runes 0-5 "quote\n", 6 "\n", 7-15 "paragraph"
			// Select the blockquote content "quote\n"
			input:        "> quote\n\nparagraph",
			renderedPos:  0,
			renderedEnd:  6, // "quote\n"
			wantSrcStart: 0,
			wantSrcEnd:   8, // "> quote\n"
		},
		{
			name: "blockquote followed by paragraph - select paragraph",
			// Source: "> quote\n\nparagraph"
			// Rendered: "quote\n" + "\n" + "paragraph"
			// Select "paragraph" (rendered positions 7-16)
			input:        "> quote\n\nparagraph",
			renderedPos:  7,
			renderedEnd:  16, // "paragraph"
			wantSrcStart: 9,
			wantSrcEnd:   18, // "paragraph"
		},
		{
			name: "blockquote with bold - select bold text",
			// Source: "> **bold** text" (15 runes)
			// Prefix: "> " (2 runes), content: "**bold** text" (13 runes)
			// Rendered: "bold text" (9 runes)
			// "bold" is at rendered positions 0-3, " text" at 4-8
			input:        "> **bold** text",
			renderedPos:  0,
			renderedEnd:  4, // "bold"
			wantSrcStart: 0,
			wantSrcEnd:   10, // "> **bold**" - boundary expands to include prefix + markers
		},
		{
			name: "blockquote with bold - select plain text after bold",
			// Source: "> **bold** text" (15 runes)
			// Rendered: "bold text" (9 runes)
			// " text" at rendered positions 4-8
			input:        "> **bold** text",
			renderedPos:  4,
			renderedEnd:  9, // " text"
			wantSrcStart: 10,
			wantSrcEnd:   15, // " text" in source
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm, _ := ParseWithSourceMap(tt.input)
			srcStart, srcEnd := sm.ToSource(tt.renderedPos, tt.renderedEnd)
			if srcStart != tt.wantSrcStart || srcEnd != tt.wantSrcEnd {
				t.Errorf("ToSource(%d, %d) = (%d, %d), want (%d, %d)",
					tt.renderedPos, tt.renderedEnd, srcStart, srcEnd,
					tt.wantSrcStart, tt.wantSrcEnd)
			}
		})
	}
}

// TestBlockquoteSourceMapToRendered tests that ToRendered correctly maps source
// rune positions to rendered positions for blockquote content.
func TestBlockquoteSourceMapToRendered(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		srcRuneStart  int
		srcRuneEnd    int
		wantRendStart int
		wantRendEnd   int
	}{
		{
			name: "blockquote - full source range",
			// Source: "> hello" (7 runes), rendered: "hello" (5 runes)
			input:         "> hello",
			srcRuneStart:  0,
			srcRuneEnd:    7,
			wantRendStart: 0,
			wantRendEnd:   5,
		},
		{
			name: "blockquote - content only (after prefix)",
			// Source rune 2 is 'h' (first char after "> "), rune 7 is end
			input:         "> hello",
			srcRuneStart:  2,
			srcRuneEnd:    7,
			wantRendStart: 0,
			wantRendEnd:   5,
		},
		{
			name: "blockquote - within prefix maps to rendered start",
			// Source rune 0 is '>', rune 1 is ' ' — both within prefix
			// Should map to rendered start
			input:         "> hello",
			srcRuneStart:  0,
			srcRuneEnd:    2, // Just the "> " prefix
			wantRendStart: 0,
			wantRendEnd:   0, // Prefix maps to rendered start (no rendered content)
		},
		{
			name: "nested blockquote depth 2 - full source",
			// Source: "> > inner" (9 runes), rendered: "inner" (5 runes)
			input:         "> > inner",
			srcRuneStart:  0,
			srcRuneEnd:    9,
			wantRendStart: 0,
			wantRendEnd:   5,
		},
		{
			name: "nested blockquote depth 2 - content only",
			// "> > " is 4 runes, content "inner" starts at rune 4
			input:         "> > inner",
			srcRuneStart:  4,
			srcRuneEnd:    9,
			wantRendStart: 0,
			wantRendEnd:   5,
		},
		{
			name: "blockquote with bold - source bold markers",
			// Source: "> **bold** text" (15 runes)
			// Content starts at rune 2: "**bold** text"
			// "**bold**" = source runes 2-10
			// Rendered: "bold text" (9 runes), "bold" at 0-3
			input:         "> **bold** text",
			srcRuneStart:  2,
			srcRuneEnd:    10, // "**bold**"
			wantRendStart: 0,
			wantRendEnd:   4, // "bold"
		},
		{
			name: "blockquote with bold - source text after bold",
			// Source: "> **bold** text" (15 runes)
			// " text" = source runes 10-15
			// Rendered: "bold text", " text" at 4-8
			input:         "> **bold** text",
			srcRuneStart:  10,
			srcRuneEnd:    15,
			wantRendStart: 4,
			wantRendEnd:   9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm, _ := ParseWithSourceMap(tt.input)
			rendStart, rendEnd := sm.ToRendered(tt.srcRuneStart, tt.srcRuneEnd)
			if rendStart != tt.wantRendStart || rendEnd != tt.wantRendEnd {
				t.Errorf("ToRendered(%d, %d) = (%d, %d), want (%d, %d)",
					tt.srcRuneStart, tt.srcRuneEnd, rendStart, rendEnd,
					tt.wantRendStart, tt.wantRendEnd)
			}
		})
	}
}

// TestBlockquoteSourceMapRoundTrip tests that rendered→source→rendered produces
// positions that contain the original selection for blockquote content.
func TestBlockquoteSourceMapRoundTrip(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		renderedStart int
		renderedEnd   int
	}{
		{
			name:          "simple blockquote round trip",
			input:         "> hello",
			renderedStart: 0,
			renderedEnd:   5, // "hello"
		},
		{
			name:          "nested blockquote round trip",
			input:         "> > inner",
			renderedStart: 0,
			renderedEnd:   5, // "inner"
		},
		{
			name:          "blockquote with bold round trip",
			input:         "> **bold** text",
			renderedStart: 0,
			renderedEnd:   4, // "bold"
		},
		{
			name:          "blockquote plain text after bold round trip",
			input:         "> **bold** text",
			renderedStart: 4,
			renderedEnd:   9, // " text"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm, _ := ParseWithSourceMap(tt.input)

			// Step 1: rendered → source
			srcStart, srcEnd := sm.ToSource(tt.renderedStart, tt.renderedEnd)

			// Step 2: convert byte positions to rune positions for ToRendered
			srcRuneStart := byteToRunePos(tt.input, srcStart)
			srcRuneEnd := byteToRunePos(tt.input, srcEnd)

			// Step 3: source → rendered (round trip)
			rendStart, rendEnd := sm.ToRendered(srcRuneStart, srcRuneEnd)

			// The round trip should produce positions that contain the original selection
			if rendStart > tt.renderedStart || rendEnd < tt.renderedEnd {
				t.Errorf("round trip failed: rendered(%d,%d) → source(%d,%d) [runes: %d,%d] → rendered(%d,%d); want rendered to contain [%d,%d]",
					tt.renderedStart, tt.renderedEnd,
					srcStart, srcEnd,
					srcRuneStart, srcRuneEnd,
					rendStart, rendEnd,
					tt.renderedStart, tt.renderedEnd)
			}
		})
	}
}

// TestBlockquoteSourceMapPointSelection verifies that point selections (clicks)
// in blockquote rendered content produce point selections in source.
func TestBlockquoteSourceMapPointSelection(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		renderedPos int
	}{
		{"blockquote start", "> hello", 0},
		{"blockquote middle", "> hello", 2},
		{"blockquote end", "> hello", 4},
		{"nested blockquote start", "> > inner", 0},
		{"nested blockquote middle", "> > inner", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm, _ := ParseWithSourceMap(tt.input)
			srcStart, srcEnd := sm.ToSource(tt.renderedPos, tt.renderedPos)
			if srcStart != srcEnd {
				t.Errorf("ToSource(%d, %d) = (%d, %d), want point selection (srcStart == srcEnd)",
					tt.renderedPos, tt.renderedPos, srcStart, srcEnd)
			}
		})
	}
}

// TestBlockquoteSourceMapContentProduction verifies that ParseWithSourceMap
// produces the correct rendered content (spans) for blockquotes, with
// blockquote styling applied and the `> ` prefix stripped.
func TestBlockquoteSourceMapContentProduction(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTexts []string // expected span texts
	}{
		{
			name:      "simple blockquote",
			input:     "> hello",
			wantTexts: []string{"hello"},
		},
		{
			name:      "nested blockquote",
			input:     "> > inner",
			wantTexts: []string{"inner"},
		},
		{
			name:      "blockquote with bold",
			input:     "> **bold** text",
			wantTexts: []string{"bold", " text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, _, _ := ParseWithSourceMap(tt.input)
			if len(content) != len(tt.wantTexts) {
				var gotTexts []string
				for _, s := range content {
					gotTexts = append(gotTexts, s.Text)
				}
				t.Fatalf("got %d spans %v, want %d spans %v",
					len(content), gotTexts, len(tt.wantTexts), tt.wantTexts)
			}
			for i, want := range tt.wantTexts {
				if content[i].Text != want {
					t.Errorf("span[%d].Text = %q, want %q", i, content[i].Text, want)
				}
				if !content[i].Style.Blockquote {
					t.Errorf("span[%d].Style.Blockquote = false, want true", i)
				}
			}
		})
	}
}

// TestTableSourceMapPerCell tests that per-cell source map entries correctly
// map rendered positions to source positions for table cells. This replaces
// the old per-row mapping that caused incorrect highlights due to column
// width normalization.
func TestTableSourceMapPerCell(t *testing.T) {
	// Source:
	//   | Flag | Purpose |\n     (line 1, 19 bytes, runes 0-18)
	//   | --- | --- |\n          (line 2, 14 bytes, runes 19-32)
	//   | -v | Verbose |\n       (line 3, 19 bytes, runes 33-51)
	//
	// Rendered (box-drawn):
	//   ┌──────┬─────────┐\n    (span 0, rendPos 0-18)
	//   │ Flag │ Purpose │\n    (span 1, rendPos 19-37)
	//   ├──────┼─────────┤\n    (span 2, rendPos 38-56)
	//   │ -v   │ Verbose │\n    (span 3, rendPos 57-75)
	//   └──────┴─────────┘\n    (span 4, rendPos 76-94)
	//
	// Cell content rendered positions:
	//   Header: "Flag" at rend 21-25, "Purpose" at rend 28-35
	//   Data:   "-v" at rend 59-61,   "Verbose" at rend 66-73
	//
	// Cell content source rune positions:
	//   Header: "Flag" at src 2-6, "Purpose" at src 9-16
	//   Data:   "-v" at src 35-37, "Verbose" at src 40-47

	input := "| Flag | Purpose |\n| --- | --- |\n| -v | Verbose |\n"

	t.Run("ToSource maps rendered cell positions to correct source positions", func(t *testing.T) {
		tests := []struct {
			name         string
			renderedPos  int
			renderedEnd  int
			wantSrcStart int
			wantSrcEnd   int
		}{
			{"Flag full cell", 21, 25, 2, 6},
			{"Purpose full cell", 28, 35, 9, 16},
			{"-v full cell", 59, 61, 35, 37},
			{"Verbose full cell", 66, 73, 40, 47},
			{"Flag partial (Fl)", 21, 23, 2, 4},
			{"Purpose partial (Pur)", 28, 31, 9, 12},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, sm, _ := ParseWithSourceMap(input)
				srcStart, srcEnd := sm.ToSource(tt.renderedPos, tt.renderedEnd)
				if srcStart != tt.wantSrcStart || srcEnd != tt.wantSrcEnd {
					t.Errorf("ToSource(%d, %d) = (%d, %d), want (%d, %d)",
						tt.renderedPos, tt.renderedEnd, srcStart, srcEnd,
						tt.wantSrcStart, tt.wantSrcEnd)
				}
			})
		}
	})

	t.Run("ToRendered maps source cell positions to correct rendered positions", func(t *testing.T) {
		tests := []struct {
			name          string
			srcRuneStart  int
			srcRuneEnd    int
			wantRendStart int
			wantRendEnd   int
		}{
			{"Flag", 2, 6, 21, 25},
			{"Purpose", 9, 16, 28, 35},
			{"-v", 35, 37, 59, 61},
			{"Verbose", 40, 47, 66, 73},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, sm, _ := ParseWithSourceMap(input)
				rendStart, rendEnd := sm.ToRendered(tt.srcRuneStart, tt.srcRuneEnd)
				if rendStart != tt.wantRendStart || rendEnd != tt.wantRendEnd {
					t.Errorf("ToRendered(%d, %d) = (%d, %d), want (%d, %d)",
						tt.srcRuneStart, tt.srcRuneEnd, rendStart, rendEnd,
						tt.wantRendStart, tt.wantRendEnd)
				}
			})
		}
	})

	t.Run("point selections map correctly", func(t *testing.T) {
		_, sm, _ := ParseWithSourceMap(input)

		// Point selection in rendered table cell should produce point in source
		for _, rendPos := range []int{21, 23, 25, 28, 32, 59, 66, 70} {
			srcStart, srcEnd := sm.ToSource(rendPos, rendPos)
			if srcStart != srcEnd {
				t.Errorf("ToSource(%d, %d) = (%d, %d), want point selection",
					rendPos, rendPos, srcStart, srcEnd)
			}
		}
	})

	t.Run("round trip rendered→source→rendered", func(t *testing.T) {
		_, sm, _ := ParseWithSourceMap(input)

		cases := []struct {
			name string
			rS   int
			rE   int
		}{
			{"Flag", 21, 25},
			{"Purpose", 28, 35},
			{"-v", 59, 61},
			{"Verbose", 66, 73},
		}

		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				srcStart, srcEnd := sm.ToSource(tt.rS, tt.rE)
				srcRuneStart := byteToRunePos(input, srcStart)
				srcRuneEnd := byteToRunePos(input, srcEnd)
				rendStart, rendEnd := sm.ToRendered(srcRuneStart, srcRuneEnd)

				if rendStart > tt.rS || rendEnd < tt.rE {
					t.Errorf("round trip: rendered(%d,%d) → source(%d,%d) → rendered(%d,%d); want to contain [%d,%d]",
						tt.rS, tt.rE, srcStart, srcEnd, rendStart, rendEnd, tt.rS, tt.rE)
				}
			})
		}
	})
}

// TestTableSourceMapAlignments tests per-cell source map entries with
// left, center, and right aligned columns.
func TestTableSourceMapAlignments(t *testing.T) {
	input := "| Left | Center | Right |\n| :--- | :---: | ---: |\n| a | b | c |\n"
	_, sm, _ := ParseWithSourceMap(input)

	// Source line 3: "| a | b | c |\n" (14 bytes)
	// starts at byte/rune 50 (26 + 24 = 50)
	// "a" at source rune 52, "b" at 56, "c" at 60

	// Test ToRendered for each alignment
	rsA, reA := sm.ToRendered(52, 53)
	rsB, reB := sm.ToRendered(56, 57)
	rsC, reC := sm.ToRendered(60, 61)

	// Each should map to exactly 1 rune in rendered
	if reA-rsA != 1 {
		t.Errorf("left-aligned 'a': ToRendered(52,53) = (%d,%d), want 1-rune span", rsA, reA)
	}
	if reB-rsB != 1 {
		t.Errorf("center-aligned 'b': ToRendered(56,57) = (%d,%d), want 1-rune span", rsB, reB)
	}
	if reC-rsC != 1 {
		t.Errorf("right-aligned 'c': ToRendered(60,61) = (%d,%d), want 1-rune span", rsC, reC)
	}

	// Verify round trip for each
	for _, tt := range []struct {
		name string
		rS   int
		rE   int
	}{
		{"a left-aligned", rsA, reA},
		{"b center-aligned", rsB, reB},
		{"c right-aligned", rsC, reC},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srcStart, srcEnd := sm.ToSource(tt.rS, tt.rE)
			srcRuneStart := byteToRunePos(input, srcStart)
			srcRuneEnd := byteToRunePos(input, srcEnd)
			rendStart, rendEnd := sm.ToRendered(srcRuneStart, srcRuneEnd)

			if rendStart > tt.rS || rendEnd < tt.rE {
				t.Errorf("round trip: rendered(%d,%d) → source(%d,%d) → rendered(%d,%d); want to contain [%d,%d]",
					tt.rS, tt.rE, srcStart, srcEnd, rendStart, rendEnd, tt.rS, tt.rE)
			}
		})
	}
}

// TestTableSourceMapEmptyCell tests that empty cells get zero-length
// point entries in the source map.
func TestTableSourceMapEmptyCell(t *testing.T) {
	input := "| A | |\n| --- | --- |\n| x | |\n"
	_, sm, _ := ParseWithSourceMap(input)

	// "A" in header should map correctly
	rsA, reA := sm.ToRendered(2, 3) // "A" at source rune 2-3
	if reA-rsA != 1 {
		t.Errorf("'A': ToRendered(2,3) = (%d,%d), want 1-rune span", rsA, reA)
	}

	// "x" in data row should map correctly
	// Source line 1: "| A | |\n" = 8 runes (0-7)
	// Source line 2: "| --- | --- |\n" = 14 runes (8-21)
	// Source line 3: "| x | |\n" starts at rune 22
	// "x" at source rune 24
	rsX, reX := sm.ToRendered(24, 25)
	if reX-rsX != 1 {
		t.Errorf("'x': ToRendered(24,25) = (%d,%d), want 1-rune span", rsX, reX)
	}
}

// TestTableGapSnapping tests that clicking on gap positions (padding spaces,
// pipe delimiters) within a rendered table row snaps the cursor to the end of
// the preceding cell's content, not into the delimiter area.
func TestTableGapSnapping(t *testing.T) {
	// Source:
	//   | Flag | Purpose |\n
	//   | --- | --- |\n
	//   | -v | Verbose |\n
	//
	// Rendered (box-drawn):
	//   ┌──────┬─────────┐\n    (rend 0-18)
	//   │ Flag │ Purpose │\n    (rend 19-37)
	//   ├──────┼─────────┤\n    (rend 38-56)
	//   │ -v   │ Verbose │\n    (rend 57-75)
	//   └──────┴─────────┘\n    (rend 76-94)
	//
	// Per-cell entries for header row:
	//   "Flag"    at rend [21,25), source rune [2,6)
	//   "Purpose" at rend [28,35), source rune [9,16)
	// Gap positions 25-27 (space, │, space) are between entries.
	//
	// Per-cell entries for data row:
	//   "-v"      at rend [59,61), source rune [35,37)
	//   "Verbose" at rend [66,73), source rune [40,47)
	// Gap positions 61-65 are between entries.

	input := "| Flag | Purpose |\n| --- | --- |\n| -v | Verbose |\n"

	t.Run("point selection in header row gap before border snaps to cell end", func(t *testing.T) {
		_, sm, _ := ParseWithSourceMap(input)

		// Gap positions 25 (space), 26 (│) are before or on the border.
		// They should snap to source rune 6 (end of "Flag" content).
		for _, rendPos := range []int{25, 26} {
			srcStart, srcEnd := sm.ToSource(rendPos, rendPos)
			if srcStart != srcEnd {
				t.Errorf("ToSource(%d, %d) = (%d, %d), want point selection", rendPos, rendPos, srcStart, srcEnd)
			}
			if srcStart != 6 {
				t.Errorf("ToSource(%d, %d) = (%d, %d), want (6, 6) — snap to end of 'Flag'", rendPos, rendPos, srcStart, srcEnd)
			}
		}
	})

	t.Run("point selection in header row gap after border snaps to following cell", func(t *testing.T) {
		_, sm, _ := ParseWithSourceMap(input)

		// Gap position 27 (space after │) is in the following cell's area.
		// It should snap to source rune 9 (start of "Purpose" content).
		srcStart, srcEnd := sm.ToSource(27, 27)
		if srcStart != srcEnd {
			t.Errorf("ToSource(27, 27) = (%d, %d), want point selection", srcStart, srcEnd)
		}
		if srcStart != 9 {
			t.Errorf("ToSource(27, 27) = (%d, %d), want (9, 9) — snap to start of 'Purpose'", srcStart, srcEnd)
		}
	})

	t.Run("point selection in data row gap before border snaps to cell end", func(t *testing.T) {
		_, sm, _ := ParseWithSourceMap(input)

		// Gap positions 61-64 are before or on the border (│ at 64).
		// They should snap to source rune 37 (end of "-v" content).
		for _, rendPos := range []int{61, 62, 63, 64} {
			srcStart, srcEnd := sm.ToSource(rendPos, rendPos)
			if srcStart != srcEnd {
				t.Errorf("ToSource(%d, %d) = (%d, %d), want point selection", rendPos, rendPos, srcStart, srcEnd)
			}
			if srcStart != 37 {
				t.Errorf("ToSource(%d, %d) = (%d, %d), want (37, 37) — snap to end of '-v'", rendPos, rendPos, srcStart, srcEnd)
			}
		}
	})

	t.Run("point selection in data row gap after border snaps to following cell", func(t *testing.T) {
		_, sm, _ := ParseWithSourceMap(input)

		// Gap position 65 (space after │) is in the following cell's area.
		// It should snap to source rune 40 (start of "Verbose" content).
		srcStart, srcEnd := sm.ToSource(65, 65)
		if srcStart != srcEnd {
			t.Errorf("ToSource(65, 65) = (%d, %d), want point selection", srcStart, srcEnd)
		}
		if srcStart != 40 {
			t.Errorf("ToSource(65, 65) = (%d, %d), want (40, 40) — snap to start of 'Verbose'", srcStart, srcEnd)
		}
	})
}

// TestTableGapTyping simulates the full type-in-gap scenario: click a gap
// position, map to source, verify the source position is at cell content end
// (not in the pipe delimiter), so inserting a character appends to the cell.
func TestTableGapTyping(t *testing.T) {
	input := "| Flag | Purpose |\n| --- | --- |\n| -v | Verbose |\n"
	_, sm, _ := ParseWithSourceMap(input)

	// Click at rendered position 26 (the │ between "Flag" and "Purpose")
	srcStart, srcEnd := sm.ToSource(26, 26)

	// Should snap to source rune 6 (right after "g" in "Flag")
	if srcStart != 6 || srcEnd != 6 {
		t.Fatalf("ToSource(26, 26) = (%d, %d), want (6, 6)", srcStart, srcEnd)
	}

	// Verify that source rune 6 is the position right after "Flag" content
	// In source "| Flag | Purpose |\n", rune 6 is the space after "Flag"
	// which is the pipe position — but since we're snapping to SourceRuneEnd
	// of the "Flag" cell entry (which is the byte-end converted to rune),
	// inserting here would append to cell content.
	sourceRunes := []rune(input)
	if srcStart > len(sourceRunes) {
		t.Fatalf("source position %d out of range (source len %d runes)", srcStart, len(sourceRunes))
	}
}

// TestTableGapDoesNotAffectNonTable verifies that non-table gap handling
// (paragraph breaks, code block boundaries) still uses 1:1 offset and
// is not affected by the table cell gap snapping logic.
func TestTableGapDoesNotAffectNonTable(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		renderedPos  int
		renderedEnd  int
		wantSrcStart int
		wantSrcEnd   int
	}{
		{
			name: "paragraph break gap uses offset",
			// Source: "Para1\n\nPara2" (12 bytes)
			// Rendered: "Para1\n" + "\n" (parabreak) + "Para2"
			// The "\n" parabreak at rendered pos 6 is a gap between entries.
			// The end gap handler finds "Para1\n" entry and adds 1:1 offset,
			// mapping to source rune 7 (start of "Para2").
			input:        "Para1\n\nPara2",
			renderedPos:  6,
			renderedEnd:  6,
			wantSrcStart: 7, // 1:1 offset through gap (non-table)
			wantSrcEnd:   7,
		},
		{
			name: "code block boundary gap",
			// Source: "Before\n```\ncode\n```\nAfter" (25 bytes)
			// Rendered: "Before\ncode\nAfter" (17 runes)
			// Clicking at the start of "After" (rendered 12) should map correctly.
			input:        "Before\n```\ncode\n```\nAfter",
			renderedPos:  12,
			renderedEnd:  17,
			wantSrcStart: 20,
			wantSrcEnd:   25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, sm, _ := ParseWithSourceMap(tt.input)
			srcStart, srcEnd := sm.ToSource(tt.renderedPos, tt.renderedEnd)
			if srcStart != tt.wantSrcStart || srcEnd != tt.wantSrcEnd {
				t.Errorf("ToSource(%d, %d) = (%d, %d), want (%d, %d)",
					tt.renderedPos, tt.renderedEnd, srcStart, srcEnd,
					tt.wantSrcStart, tt.wantSrcEnd)
			}
		})
	}
}

// TestRunePositionsValidGuard verifies that ToSource and ToRendered panic
// when called on a SourceMap whose rune positions have not been populated
// (or have been invalidated). This catches the temporal coupling bug where
// SourceRuneStart/End are zero-valued but treated as meaningful.
func TestRunePositionsValidGuard(t *testing.T) {
	t.Run("ToSource panics on invalid rune positions", func(t *testing.T) {
		sm := &SourceMap{
			entries: []SourceMapEntry{
				{RenderedStart: 0, RenderedEnd: 5, SourceStart: 0, SourceEnd: 5},
			},
		}
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic from ToSource on invalid rune positions")
			}
			msg, ok := r.(string)
			if !ok || !strings.Contains(msg, "ToSource") {
				t.Errorf("unexpected panic message: %v", r)
			}
		}()
		sm.ToSource(0, 1)
	})

	t.Run("ToRendered panics on invalid rune positions", func(t *testing.T) {
		sm := &SourceMap{
			entries: []SourceMapEntry{
				{RenderedStart: 0, RenderedEnd: 5, SourceStart: 0, SourceEnd: 5},
			},
		}
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic from ToRendered on invalid rune positions")
			}
			msg, ok := r.(string)
			if !ok || !strings.Contains(msg, "ToRendered") {
				t.Errorf("unexpected panic message: %v", r)
			}
		}()
		sm.ToRendered(0, 1)
	})

	t.Run("PopulateRunePositions enables ToSource", func(t *testing.T) {
		// After PopulateRunePositions, the guard should pass.
		source := "Hello"
		sm := &SourceMap{
			entries: []SourceMapEntry{
				{RenderedStart: 0, RenderedEnd: 5, SourceStart: 0, SourceEnd: 5},
			},
		}
		sm.PopulateRunePositions(source)
		// Should not panic.
		sm.ToSource(0, 1)
	})

	t.Run("InvalidateRunePositions re-enables guard", func(t *testing.T) {
		source := "Hello"
		sm := &SourceMap{
			entries: []SourceMapEntry{
				{RenderedStart: 0, RenderedEnd: 5, SourceStart: 0, SourceEnd: 5},
			},
		}
		sm.PopulateRunePositions(source)
		sm.InvalidateRunePositions()
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic after InvalidateRunePositions")
			}
		}()
		sm.ToSource(0, 1)
	})

	t.Run("empty SourceMap does not panic", func(t *testing.T) {
		sm := &SourceMap{}
		// Empty entries should return early before the guard check.
		srcStart, srcEnd := sm.ToSource(0, 0)
		if srcStart != 0 || srcEnd != 0 {
			t.Errorf("empty ToSource = (%d,%d), want (0,0)", srcStart, srcEnd)
		}
		rendStart, rendEnd := sm.ToRendered(0, 0)
		if rendStart != -1 || rendEnd != -1 {
			t.Errorf("empty ToRendered = (%d,%d), want (-1,-1)", rendStart, rendEnd)
		}
	})

	t.Run("ParseWithSourceMap returns valid rune positions", func(t *testing.T) {
		_, sm, _ := ParseWithSourceMap("Hello **bold** world")
		// Should not panic — ParseWithSourceMap calls populateRunePositions.
		sm.ToSource(0, 5)
		sm.ToRendered(0, 5)
	})
}

// byteToRunePos converts a byte position in a string to a rune position.
func byteToRunePos(s string, bytePos int) int {
	if bytePos <= 0 {
		return 0
	}
	if bytePos >= len(s) {
		return len([]rune(s))
	}
	return len([]rune(s[:bytePos]))
}
