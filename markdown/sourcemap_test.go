package markdown

import (
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
