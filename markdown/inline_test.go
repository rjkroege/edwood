package markdown

import (
	"image/color"
	"testing"

	"github.com/rjkroege/edwood/rich"
)

// ---- Category A: Basic Formatting (spans only, no source map) ----
// Tests that parseInline produces correct spans with default InlineOpts.

func TestInlinePlainText(t *testing.T) {
	spans := parseInline("hello world", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Text != "hello world" {
		t.Errorf("expected text %q, got %q", "hello world", spans[0].Text)
	}
	if spans[0].Style != rich.DefaultStyle() {
		t.Errorf("expected default style, got %+v", spans[0].Style)
	}
}

func TestInlineBold(t *testing.T) {
	spans := parseInline("**bold**", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d: %+v", len(spans), spans)
	}
	if spans[0].Text != "bold" {
		t.Errorf("expected text %q, got %q", "bold", spans[0].Text)
	}
	if !spans[0].Style.Bold {
		t.Error("expected Bold=true")
	}
	if spans[0].Style.Italic {
		t.Error("expected Italic=false")
	}
}

func TestInlineItalic(t *testing.T) {
	spans := parseInline("*italic*", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d: %+v", len(spans), spans)
	}
	if spans[0].Text != "italic" {
		t.Errorf("expected text %q, got %q", "italic", spans[0].Text)
	}
	if !spans[0].Style.Italic {
		t.Error("expected Italic=true")
	}
	if spans[0].Style.Bold {
		t.Error("expected Bold=false")
	}
}

func TestInlineBoldItalic(t *testing.T) {
	spans := parseInline("***both***", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d: %+v", len(spans), spans)
	}
	if spans[0].Text != "both" {
		t.Errorf("expected text %q, got %q", "both", spans[0].Text)
	}
	if !spans[0].Style.Bold {
		t.Error("expected Bold=true")
	}
	if !spans[0].Style.Italic {
		t.Error("expected Italic=true")
	}
}

func TestInlineCode(t *testing.T) {
	spans := parseInline("`code`", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d: %+v", len(spans), spans)
	}
	if spans[0].Text != "code" {
		t.Errorf("expected text %q, got %q", "code", spans[0].Text)
	}
	if !spans[0].Style.Code {
		t.Error("expected Code=true")
	}
	if spans[0].Style.Bg != rich.InlineCodeBg {
		t.Errorf("expected Bg=InlineCodeBg, got %v", spans[0].Style.Bg)
	}
}

func TestInlineMixed(t *testing.T) {
	// "a **b** c" should produce 3 spans: plain "a ", bold "b", plain " c"
	spans := parseInline("a **b** c", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 3 {
		t.Fatalf("expected 3 spans, got %d: %+v", len(spans), spans)
	}
	if spans[0].Text != "a " {
		t.Errorf("span[0] text: expected %q, got %q", "a ", spans[0].Text)
	}
	if spans[0].Style.Bold {
		t.Error("span[0] should not be bold")
	}
	if spans[1].Text != "b" {
		t.Errorf("span[1] text: expected %q, got %q", "b", spans[1].Text)
	}
	if !spans[1].Style.Bold {
		t.Error("span[1] should be bold")
	}
	if spans[2].Text != " c" {
		t.Errorf("span[2] text: expected %q, got %q", " c", spans[2].Text)
	}
	if spans[2].Style.Bold {
		t.Error("span[2] should not be bold")
	}
}

func TestInlineUnclosedBold(t *testing.T) {
	// Unclosed ** should be treated as literal text
	spans := parseInline("**oops", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d: %+v", len(spans), spans)
	}
	if spans[0].Text != "**oops" {
		t.Errorf("expected text %q, got %q", "**oops", spans[0].Text)
	}
	if spans[0].Style.Bold {
		t.Error("unclosed bold should not set Bold=true")
	}
}

func TestInlineUnclosedCode(t *testing.T) {
	// Unclosed backtick should be treated as literal text
	spans := parseInline("`oops", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d: %+v", len(spans), spans)
	}
	if spans[0].Text != "`oops" {
		t.Errorf("expected text %q, got %q", "`oops", spans[0].Text)
	}
}

func TestInlineBaseStylePreserved(t *testing.T) {
	// Bold span should preserve Fg, Bg, Scale from baseStyle
	base := rich.Style{
		Fg:    color.RGBA{R: 100, G: 0, B: 0, A: 255},
		Bg:    color.RGBA{R: 0, G: 100, B: 0, A: 255},
		Scale: 2.0,
	}
	spans := parseInline("**bold**", base, InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	s := spans[0].Style
	if s.Fg != base.Fg {
		t.Errorf("Fg not preserved: expected %v, got %v", base.Fg, s.Fg)
	}
	if s.Bg != base.Bg {
		t.Errorf("Bg not preserved: expected %v, got %v", base.Bg, s.Bg)
	}
	if s.Scale != base.Scale {
		t.Errorf("Scale not preserved: expected %v, got %v", base.Scale, s.Scale)
	}
	if !s.Bold {
		t.Error("expected Bold=true")
	}
}

func TestInlineItalicPreservesBold(t *testing.T) {
	// If baseStyle has Bold=true, italic should preserve it
	base := rich.Style{Bold: true, Scale: 1.0}
	spans := parseInline("*italic*", base, InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if !spans[0].Style.Bold {
		t.Error("italic should preserve Bold from baseStyle")
	}
	if !spans[0].Style.Italic {
		t.Error("expected Italic=true")
	}
}

func TestInlineBoldPreservesItalic(t *testing.T) {
	// If baseStyle has Italic=true, bold should preserve it
	base := rich.Style{Italic: true, Scale: 1.0}
	spans := parseInline("**bold**", base, InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if !spans[0].Style.Italic {
		t.Error("bold should preserve Italic from baseStyle")
	}
	if !spans[0].Style.Bold {
		t.Error("expected Bold=true")
	}
}

// ---- Category B: Links and Images ----
// Tests that link and image parsing works, and NoLinks mode disables them.

func TestInlineLink(t *testing.T) {
	spans := parseInline("[text](http://example.com)", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d: %+v", len(spans), spans)
	}
	if spans[0].Text != "text" {
		t.Errorf("expected text %q, got %q", "text", spans[0].Text)
	}
	if !spans[0].Style.Link {
		t.Error("expected Link=true")
	}
	if spans[0].Style.Fg != rich.LinkBlue {
		t.Errorf("expected Fg=LinkBlue, got %v", spans[0].Style.Fg)
	}
}

func TestInlineLinkWithBold(t *testing.T) {
	// Bold text inside a link: [**bold**](url)
	spans := parseInline("[**bold**](http://example.com)", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d: %+v", len(spans), spans)
	}
	if spans[0].Text != "bold" {
		t.Errorf("expected text %q, got %q", "bold", spans[0].Text)
	}
	if !spans[0].Style.Link {
		t.Error("expected Link=true")
	}
	if !spans[0].Style.Bold {
		t.Error("expected Bold=true")
	}
}

func TestInlineLinkNoLinksMode(t *testing.T) {
	// With NoLinks=true, [text](url) should be treated as plain text
	spans := parseInline("[text](url)", rich.DefaultStyle(), InlineOpts{NoLinks: true})
	// Should contain the literal characters, not parsed as a link
	totalText := ""
	for _, s := range spans {
		totalText += s.Text
	}
	if totalText != "[text](url)" {
		t.Errorf("expected literal %q, got %q", "[text](url)", totalText)
	}
	for _, s := range spans {
		if s.Style.Link {
			t.Error("NoLinks mode should not produce Link=true spans")
		}
	}
}

func TestInlineImage(t *testing.T) {
	spans := parseInline("![alt text](image.png)", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d: %+v", len(spans), spans)
	}
	if !spans[0].Style.Image {
		t.Error("expected Image=true")
	}
	if spans[0].Style.ImageAlt != "alt text" {
		t.Errorf("expected ImageAlt %q, got %q", "alt text", spans[0].Style.ImageAlt)
	}
	// Placeholder text should contain alt text
	if spans[0].Text != "[Image: alt text]" {
		t.Errorf("expected placeholder %q, got %q", "[Image: alt text]", spans[0].Text)
	}
}

func TestInlineImageNoAlt(t *testing.T) {
	spans := parseInline("![](image.png)", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d: %+v", len(spans), spans)
	}
	if spans[0].Text != "[Image]" {
		t.Errorf("expected placeholder %q, got %q", "[Image]", spans[0].Text)
	}
}

func TestInlineImageNoLinksMode(t *testing.T) {
	// With NoLinks=true, images should not be parsed
	spans := parseInline("![alt](url)", rich.DefaultStyle(), InlineOpts{NoLinks: true})
	totalText := ""
	for _, s := range spans {
		totalText += s.Text
	}
	if totalText != "![alt](url)" {
		t.Errorf("expected literal %q, got %q", "![alt](url)", totalText)
	}
	for _, s := range spans {
		if s.Style.Image {
			t.Error("NoLinks mode should not produce Image=true spans")
		}
	}
}

func TestInlineEmptyLink(t *testing.T) {
	// [](url) should produce an empty span with Link=true
	spans := parseInline("[](http://example.com)", rich.DefaultStyle(), InlineOpts{})
	found := false
	for _, s := range spans {
		if s.Style.Link {
			found = true
			if s.Text != "" {
				t.Errorf("expected empty link text, got %q", s.Text)
			}
		}
	}
	if !found {
		t.Error("expected a span with Link=true for empty link")
	}
}

func TestInlineTextAroundLink(t *testing.T) {
	// "before [link](url) after"
	spans := parseInline("before [link](http://example.com) after", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 3 {
		t.Fatalf("expected 3 spans, got %d: %+v", len(spans), spans)
	}
	if spans[0].Text != "before " {
		t.Errorf("span[0]: expected %q, got %q", "before ", spans[0].Text)
	}
	if spans[1].Text != "link" {
		t.Errorf("span[1]: expected %q, got %q", "link", spans[1].Text)
	}
	if !spans[1].Style.Link {
		t.Error("span[1] should be a link")
	}
	if spans[2].Text != " after" {
		t.Errorf("span[2]: expected %q, got %q", " after", spans[2].Text)
	}
}

// ---- Category C: List Style Preservation ----
// Tests that baseStyle list fields are preserved in all formatted spans.
// This is the key improvement: the unified parser uses copy-all-then-override,
// so list fields from baseStyle propagate to bold/italic/code/link/image spans.

func TestInlineBoldPreservesListStyle(t *testing.T) {
	base := rich.Style{
		ListItem:   true,
		ListIndent: 1,
		Scale:      1.0,
	}
	spans := parseInline("**bold**", base, InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	s := spans[0].Style
	if !s.Bold {
		t.Error("expected Bold=true")
	}
	if !s.ListItem {
		t.Error("expected ListItem=true preserved from baseStyle")
	}
	if s.ListIndent != 1 {
		t.Errorf("expected ListIndent=1, got %d", s.ListIndent)
	}
}

func TestInlineCodePreservesListStyle(t *testing.T) {
	base := rich.Style{
		ListItem:    true,
		ListOrdered: true,
		ListNumber:  3,
		Scale:       1.0,
	}
	spans := parseInline("`code`", base, InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	s := spans[0].Style
	if !s.Code {
		t.Error("expected Code=true")
	}
	if !s.ListItem {
		t.Error("expected ListItem=true preserved from baseStyle")
	}
	if !s.ListOrdered {
		t.Error("expected ListOrdered=true preserved from baseStyle")
	}
	if s.ListNumber != 3 {
		t.Errorf("expected ListNumber=3, got %d", s.ListNumber)
	}
}

func TestInlineLinkPreservesListStyle(t *testing.T) {
	base := rich.Style{
		ListItem: true,
		Scale:    1.0,
	}
	spans := parseInline("[text](url)", base, InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	s := spans[0].Style
	if !s.Link {
		t.Error("expected Link=true")
	}
	if !s.ListItem {
		t.Error("expected ListItem=true preserved from baseStyle")
	}
}

func TestInlineItalicPreservesListStyle(t *testing.T) {
	base := rich.Style{
		ListItem:   true,
		ListIndent: 2,
		Scale:      1.0,
	}
	spans := parseInline("*italic*", base, InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	s := spans[0].Style
	if !s.Italic {
		t.Error("expected Italic=true")
	}
	if !s.ListItem {
		t.Error("expected ListItem=true preserved from baseStyle")
	}
	if s.ListIndent != 2 {
		t.Errorf("expected ListIndent=2, got %d", s.ListIndent)
	}
}

func TestInlineBoldItalicPreservesListStyle(t *testing.T) {
	base := rich.Style{
		ListItem:    true,
		ListOrdered: true,
		ListNumber:  5,
		ListIndent:  1,
		Scale:       1.0,
	}
	spans := parseInline("***both***", base, InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	s := spans[0].Style
	if !s.Bold || !s.Italic {
		t.Error("expected Bold=true, Italic=true")
	}
	if !s.ListItem {
		t.Error("expected ListItem=true preserved from baseStyle")
	}
	if !s.ListOrdered {
		t.Error("expected ListOrdered=true preserved from baseStyle")
	}
	if s.ListNumber != 5 {
		t.Errorf("expected ListNumber=5, got %d", s.ListNumber)
	}
}

func TestInlineImagePreservesListStyle(t *testing.T) {
	base := rich.Style{
		ListItem:   true,
		ListIndent: 1,
		Scale:      1.0,
	}
	spans := parseInline("![alt](img.png)", base, InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	s := spans[0].Style
	if !s.Image {
		t.Error("expected Image=true")
	}
	if !s.ListItem {
		t.Error("expected ListItem=true preserved from baseStyle")
	}
	if s.ListIndent != 1 {
		t.Errorf("expected ListIndent=1, got %d", s.ListIndent)
	}
}

// ---- Category D: Source Map Generation ----
// Tests that source map entries are correctly generated when SourceMap is non-nil.

func TestInlineSourceMapPlainText(t *testing.T) {
	var entries []SourceMapEntry
	spans := parseInline("abc", rich.DefaultStyle(), InlineOpts{
		SourceMap:      &entries,
		SourceOffset:   0,
		RenderedOffset: 0,
	})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	// Plain text: each character gets a 1:1 source map entry
	if len(entries) != 3 {
		t.Fatalf("expected 3 source map entries, got %d", len(entries))
	}
	for i, e := range entries {
		if e.RenderedStart != i {
			t.Errorf("entry[%d]: RenderedStart=%d, expected %d", i, e.RenderedStart, i)
		}
		if e.RenderedEnd != i+1 {
			t.Errorf("entry[%d]: RenderedEnd=%d, expected %d", i, e.RenderedEnd, i+1)
		}
		if e.SourceStart != i {
			t.Errorf("entry[%d]: SourceStart=%d, expected %d", i, e.SourceStart, i)
		}
		if e.SourceEnd != i+1 {
			t.Errorf("entry[%d]: SourceEnd=%d, expected %d", i, e.SourceEnd, i+1)
		}
	}
}

func TestInlineSourceMapBold(t *testing.T) {
	// "**b**" -> rendered "b" at rune [0,1), source bytes [0,5)
	var entries []SourceMapEntry
	spans := parseInline("**b**", rich.DefaultStyle(), InlineOpts{
		SourceMap:      &entries,
		SourceOffset:   0,
		RenderedOffset: 0,
	})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 source map entry, got %d", len(entries))
	}
	e := entries[0]
	if e.RenderedStart != 0 || e.RenderedEnd != 1 {
		t.Errorf("rendered range: expected [0,1), got [%d,%d)", e.RenderedStart, e.RenderedEnd)
	}
	if e.SourceStart != 0 || e.SourceEnd != 5 {
		t.Errorf("source range: expected [0,5), got [%d,%d)", e.SourceStart, e.SourceEnd)
	}
}

func TestInlineSourceMapCode(t *testing.T) {
	// "`x`" -> rendered "x" at rune [0,1), source bytes [0,3)
	var entries []SourceMapEntry
	parseInline("`x`", rich.DefaultStyle(), InlineOpts{
		SourceMap:      &entries,
		SourceOffset:   0,
		RenderedOffset: 0,
	})
	if len(entries) != 1 {
		t.Fatalf("expected 1 source map entry, got %d", len(entries))
	}
	e := entries[0]
	if e.RenderedStart != 0 || e.RenderedEnd != 1 {
		t.Errorf("rendered range: expected [0,1), got [%d,%d)", e.RenderedStart, e.RenderedEnd)
	}
	if e.SourceStart != 0 || e.SourceEnd != 3 {
		t.Errorf("source range: expected [0,3), got [%d,%d)", e.SourceStart, e.SourceEnd)
	}
}

func TestInlineSourceMapMixed(t *testing.T) {
	// "a **b** c" -> spans: "a ", "b", " c"
	// source map entries: 'a', ' ', 'b' (bold), ' ', 'c'
	var entries []SourceMapEntry
	parseInline("a **b** c", rich.DefaultStyle(), InlineOpts{
		SourceMap:      &entries,
		SourceOffset:   0,
		RenderedOffset: 0,
	})
	// "a " = 2 char-by-char entries, "b" = 1 bold entry, " c" = 2 char-by-char entries
	if len(entries) != 5 {
		t.Fatalf("expected 5 source map entries, got %d: %+v", len(entries), entries)
	}
	// Entry for 'a': rendered [0,1) source [0,1)
	if entries[0].RenderedStart != 0 || entries[0].RenderedEnd != 1 {
		t.Errorf("entry[0] rendered: expected [0,1), got [%d,%d)", entries[0].RenderedStart, entries[0].RenderedEnd)
	}
	if entries[0].SourceStart != 0 || entries[0].SourceEnd != 1 {
		t.Errorf("entry[0] source: expected [0,1), got [%d,%d)", entries[0].SourceStart, entries[0].SourceEnd)
	}
	// Entry for ' ': rendered [1,2) source [1,2)
	if entries[1].RenderedStart != 1 || entries[1].RenderedEnd != 2 {
		t.Errorf("entry[1] rendered: expected [1,2), got [%d,%d)", entries[1].RenderedStart, entries[1].RenderedEnd)
	}
	// Entry for bold 'b': rendered [2,3) source [2,7) i.e. "**b**"
	if entries[2].RenderedStart != 2 || entries[2].RenderedEnd != 3 {
		t.Errorf("entry[2] rendered: expected [2,3), got [%d,%d)", entries[2].RenderedStart, entries[2].RenderedEnd)
	}
	if entries[2].SourceStart != 2 || entries[2].SourceEnd != 7 {
		t.Errorf("entry[2] source: expected [2,7), got [%d,%d)", entries[2].SourceStart, entries[2].SourceEnd)
	}
	// Entry for ' ': rendered [3,4) source [7,8)
	if entries[3].RenderedStart != 3 || entries[3].RenderedEnd != 4 {
		t.Errorf("entry[3] rendered: expected [3,4), got [%d,%d)", entries[3].RenderedStart, entries[3].RenderedEnd)
	}
	if entries[3].SourceStart != 7 || entries[3].SourceEnd != 8 {
		t.Errorf("entry[3] source: expected [7,8), got [%d,%d)", entries[3].SourceStart, entries[3].SourceEnd)
	}
	// Entry for 'c': rendered [4,5) source [8,9)
	if entries[4].RenderedStart != 4 || entries[4].RenderedEnd != 5 {
		t.Errorf("entry[4] rendered: expected [4,5), got [%d,%d)", entries[4].RenderedStart, entries[4].RenderedEnd)
	}
}

func TestInlineSourceMapLink(t *testing.T) {
	// "[t](u)" -> rendered "t" at rune [0,1)
	// Source map entry should cover the link text portion within brackets
	var entries []SourceMapEntry
	var links []LinkEntry
	parseInline("[t](u)", rich.DefaultStyle(), InlineOpts{
		SourceMap:      &entries,
		LinkMap:        &links,
		SourceOffset:   0,
		RenderedOffset: 0,
	})
	// The link text "t" should have source map entries from the recursive parse
	// The entry for "t" should have SourceStart=1 (past the [), SourceEnd=2
	if len(entries) < 1 {
		t.Fatalf("expected at least 1 source map entry, got %d", len(entries))
	}
	// Check link entry
	if len(links) != 1 {
		t.Fatalf("expected 1 link entry, got %d", len(links))
	}
	if links[0].URL != "u" {
		t.Errorf("expected URL %q, got %q", "u", links[0].URL)
	}
	if links[0].Start != 0 || links[0].End != 1 {
		t.Errorf("link range: expected [0,1), got [%d,%d)", links[0].Start, links[0].End)
	}
}

func TestInlineSourceMapImage(t *testing.T) {
	// "![a](img.png)" -> rendered "[Image: a]"
	var entries []SourceMapEntry
	parseInline("![a](img.png)", rich.DefaultStyle(), InlineOpts{
		SourceMap:      &entries,
		SourceOffset:   0,
		RenderedOffset: 0,
	})
	if len(entries) != 1 {
		t.Fatalf("expected 1 source map entry, got %d", len(entries))
	}
	e := entries[0]
	// Rendered: "[Image: a]" = 10 runes
	if e.RenderedStart != 0 || e.RenderedEnd != 10 {
		t.Errorf("rendered range: expected [0,10), got [%d,%d)", e.RenderedStart, e.RenderedEnd)
	}
	// Source: "![a](img.png)" = 13 bytes
	if e.SourceStart != 0 || e.SourceEnd != 13 {
		t.Errorf("source range: expected [0,13), got [%d,%d)", e.SourceStart, e.SourceEnd)
	}
}

func TestInlineSourceMapWithOffset(t *testing.T) {
	// Verify that SourceOffset and RenderedOffset are correctly applied
	var entries []SourceMapEntry
	parseInline("**b**", rich.DefaultStyle(), InlineOpts{
		SourceMap:      &entries,
		SourceOffset:   10,
		RenderedOffset: 5,
	})
	if len(entries) != 1 {
		t.Fatalf("expected 1 source map entry, got %d", len(entries))
	}
	e := entries[0]
	if e.RenderedStart != 5 || e.RenderedEnd != 6 {
		t.Errorf("rendered range: expected [5,6), got [%d,%d)", e.RenderedStart, e.RenderedEnd)
	}
	if e.SourceStart != 10 || e.SourceEnd != 15 {
		t.Errorf("source range: expected [10,15), got [%d,%d)", e.SourceStart, e.SourceEnd)
	}
}

func TestInlineSourceMapNilNoEntries(t *testing.T) {
	// When SourceMap is nil, no entries should be generated
	spans := parseInline("**bold**", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	// No way to check entries weren't created, but ensure no panic
}

func TestInlineSourceMapItalic(t *testing.T) {
	// "*i*" -> rendered "i" at [0,1), source [0,3)
	var entries []SourceMapEntry
	parseInline("*i*", rich.DefaultStyle(), InlineOpts{
		SourceMap:      &entries,
		SourceOffset:   0,
		RenderedOffset: 0,
	})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.RenderedStart != 0 || e.RenderedEnd != 1 {
		t.Errorf("rendered: expected [0,1), got [%d,%d)", e.RenderedStart, e.RenderedEnd)
	}
	if e.SourceStart != 0 || e.SourceEnd != 3 {
		t.Errorf("source: expected [0,3), got [%d,%d)", e.SourceStart, e.SourceEnd)
	}
}

func TestInlineSourceMapBoldItalic(t *testing.T) {
	// "***bi***" -> rendered "bi" at [0,2), source [0,8)
	var entries []SourceMapEntry
	parseInline("***bi***", rich.DefaultStyle(), InlineOpts{
		SourceMap:      &entries,
		SourceOffset:   0,
		RenderedOffset: 0,
	})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.RenderedStart != 0 || e.RenderedEnd != 2 {
		t.Errorf("rendered: expected [0,2), got [%d,%d)", e.RenderedStart, e.RenderedEnd)
	}
	if e.SourceStart != 0 || e.SourceEnd != 8 {
		t.Errorf("source: expected [0,8), got [%d,%d)", e.SourceStart, e.SourceEnd)
	}
}

// ---- Category F: Edge Cases ----

func TestInlineEmptyString(t *testing.T) {
	spans := parseInline("", rich.DefaultStyle(), InlineOpts{})
	// Empty input: existing behavior returns a single span with empty text
	if len(spans) != 1 {
		t.Fatalf("expected 1 span for empty input, got %d", len(spans))
	}
	if spans[0].Text != "" {
		t.Errorf("expected empty text, got %q", spans[0].Text)
	}
}

func TestInlineEmptyStringWithSourceMap(t *testing.T) {
	var entries []SourceMapEntry
	spans := parseInline("", rich.DefaultStyle(), InlineOpts{
		SourceMap:      &entries,
		SourceOffset:   0,
		RenderedOffset: 0,
	})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span for empty input, got %d", len(spans))
	}
	// Empty string produces no source map entries (zero-length mapping is meaningless)
	if len(entries) != 0 {
		t.Fatalf("expected 0 source map entries for empty input, got %d", len(entries))
	}
}

func TestInlineOnlyMarkers(t *testing.T) {
	// "****" is treated as empty bold: ** matches ** → empty text between
	spans := parseInline("****", rich.DefaultStyle(), InlineOpts{})
	// The existing parser finds closing ** immediately at position 0,
	// producing a bold span with empty text
	if len(spans) < 1 {
		t.Fatalf("expected at least 1 span, got %d", len(spans))
	}
}

func TestInlineNestedMarkers(t *testing.T) {
	// "**a *b* c**" — the existing parser doesn't do true nesting.
	// Verify parseInline produces consistent output.
	spans := parseInline("**a *b* c**", rich.DefaultStyle(), InlineOpts{})
	if len(spans) < 1 {
		t.Fatalf("expected at least 1 span, got %d", len(spans))
	}
}

func TestInlineMultiByteRunes(t *testing.T) {
	// "**über**" — multi-byte rune within bold
	spans := parseInline("**über**", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d: %+v", len(spans), spans)
	}
	if spans[0].Text != "über" {
		t.Errorf("expected text %q, got %q", "über", spans[0].Text)
	}
	if !spans[0].Style.Bold {
		t.Error("expected Bold=true")
	}
}

func TestInlineMultiByteRunesSourceMap(t *testing.T) {
	// "**über**" — verify source map handles multi-byte correctly
	var entries []SourceMapEntry
	parseInline("**über**", rich.DefaultStyle(), InlineOpts{
		SourceMap:      &entries,
		SourceOffset:   0,
		RenderedOffset: 0,
	})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	// Rendered: "über" = 4 runes
	if e.RenderedStart != 0 || e.RenderedEnd != 4 {
		t.Errorf("rendered: expected [0,4), got [%d,%d)", e.RenderedStart, e.RenderedEnd)
	}
	// Source: "**über**" = 2 + 5 (ü=2 bytes + ber=3 bytes) + 2 = 9 bytes
	// Actually: ** = 2, über = 5 bytes (ü = 2 bytes), ** = 2 → total 9
	if e.SourceStart != 0 || e.SourceEnd != 9 {
		t.Errorf("source: expected [0,9), got [%d,%d)", e.SourceStart, e.SourceEnd)
	}
}

func TestInlineCodeWithAsterisks(t *testing.T) {
	// "`**not bold**`" — asterisks inside code are literal
	spans := parseInline("`**not bold**`", rich.DefaultStyle(), InlineOpts{})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Text != "**not bold**" {
		t.Errorf("expected text %q, got %q", "**not bold**", spans[0].Text)
	}
	if !spans[0].Style.Code {
		t.Error("expected Code=true")
	}
	if spans[0].Style.Bold {
		t.Error("asterisks in code should not trigger bold")
	}
}

func TestInlineMultipleLinks(t *testing.T) {
	// "[a](u1) and [b](u2)"
	var links []LinkEntry
	parseInline("[a](u1) and [b](u2)", rich.DefaultStyle(), InlineOpts{
		LinkMap: &links,
	})
	if len(links) != 2 {
		t.Fatalf("expected 2 link entries, got %d", len(links))
	}
	if links[0].URL != "u1" {
		t.Errorf("link[0] URL: expected %q, got %q", "u1", links[0].URL)
	}
	if links[1].URL != "u2" {
		t.Errorf("link[1] URL: expected %q, got %q", "u2", links[1].URL)
	}
}

func TestInlineLinkMapWithoutSourceMap(t *testing.T) {
	// LinkMap can be non-nil without SourceMap
	var links []LinkEntry
	spans := parseInline("[text](url)", rich.DefaultStyle(), InlineOpts{
		LinkMap: &links,
	})
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link entry, got %d", len(links))
	}
	if links[0].URL != "url" {
		t.Errorf("expected URL %q, got %q", "url", links[0].URL)
	}
}

func TestInlineInvalidLink(t *testing.T) {
	// "[text" — unclosed bracket, treated as literal
	spans := parseInline("[text", rich.DefaultStyle(), InlineOpts{})
	totalText := ""
	for _, s := range spans {
		totalText += s.Text
	}
	if totalText != "[text" {
		t.Errorf("expected %q, got %q", "[text", totalText)
	}
}

func TestInlineInvalidImage(t *testing.T) {
	// "![text" — unclosed bracket, treated as literal
	spans := parseInline("![text", rich.DefaultStyle(), InlineOpts{})
	totalText := ""
	for _, s := range spans {
		totalText += s.Text
	}
	if totalText != "![text" {
		t.Errorf("expected %q, got %q", "![text", totalText)
	}
}

func TestInlineBracketNotLink(t *testing.T) {
	// "[text] not a link" — ] not followed by ( is plain text
	spans := parseInline("[text] not a link", rich.DefaultStyle(), InlineOpts{})
	totalText := ""
	for _, s := range spans {
		totalText += s.Text
	}
	if totalText != "[text] not a link" {
		t.Errorf("expected %q, got %q", "[text] not a link", totalText)
	}
}

