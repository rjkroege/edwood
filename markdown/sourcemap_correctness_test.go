package markdown

import (
	"testing"
)

// sourcemap_correctness_test.go — Comprehensive source map correctness tests.
//
// These tests cover categories A-F from the source map correctness design:
//   docs/designs/features/sourcemap-correctness.md
//
// They target the five identified bugs:
//   Bug 1: PrefixLen byte/rune confusion
//   Bug 2: ToSource/ToRendered round-trip asymmetry
//   Bug 3: Point selection normalization masking
//   Bug 4: Entry boundary lookup off-by-one
//   Bug 5: Missing bounds validation
//
// Invariants tested:
//   R1: Rendered→Source→Rendered containment
//   R2: Source→Rendered→Source containment
//   R3: Point selection identity (click → click)
//   R4: Monotonicity (moving right in rendered never moves backward in source)

// ---------- Helpers ----------

// plainText returns the full rendered plain text from parsing input with source map.
func plainText(input string) string {
	content, _, _ := ParseWithSourceMap(input)
	return string(content.Plain())
}

// runeLen returns the rune length of s.
func runeLen(s string) int {
	return len([]rune(s))
}

// assertRoundTripR1 verifies invariant R1: Rendered→Source→Rendered containment.
// After round-tripping through ToSource then ToRendered, the resulting rendered
// range must contain the original rendered range (may expand, must not shrink).
func assertRoundTripR1(t *testing.T, sm *SourceMap, input string, r0, r1 int) {
	t.Helper()

	srcStart, srcEnd := sm.ToSource(r0, r1)
	r0p, r1p := sm.ToRendered(srcStart, srcEnd)

	if r0p == -1 || r1p == -1 {
		t.Errorf("R1: rendered(%d,%d) → source(%d,%d) → rendered(%d,%d): got -1 from ToRendered",
			r0, r1, srcStart, srcEnd, r0p, r1p)
		return
	}
	if r0p > r0 || r1p < r1 {
		t.Errorf("R1: rendered(%d,%d) → source(%d,%d) → rendered(%d,%d): round-trip shrank selection (want r0'<=r0, r1'>=r1)",
			r0, r1, srcStart, srcEnd, r0p, r1p)
	}
}

// assertRoundTripR2 verifies invariant R2: Source→Rendered→Source containment.
// After round-tripping through ToRendered then ToSource, the resulting source
// range must contain the original source range (may expand, must not shrink).
func assertRoundTripR2(t *testing.T, sm *SourceMap, s0, s1 int) {
	t.Helper()

	r0, r1 := sm.ToRendered(s0, s1)
	if r0 == -1 || r1 == -1 {
		// Source range doesn't map to any rendered content — skip.
		return
	}

	s0p, s1p := sm.ToSource(r0, r1)
	if s0p > s0 || s1p < s1 {
		t.Errorf("R2: source(%d,%d) → rendered(%d,%d) → source(%d,%d): round-trip shrank selection (want s0'<=s0, s1'>=s1)",
			s0, s1, r0, r1, s0p, s1p)
	}
}

// assertPointIdentity verifies invariant R3: a point selection maps to a point.
func assertPointIdentity(t *testing.T, sm *SourceMap, pos int) {
	t.Helper()

	srcStart, srcEnd := sm.ToSource(pos, pos)
	if srcStart != srcEnd {
		t.Errorf("R3: ToSource(%d,%d) = (%d,%d): point selection mapped to range",
			pos, pos, srcStart, srcEnd)
	}
}

// assertMonotonicity verifies invariant R4: for a < b, srcA <= srcB.
func assertMonotonicity(t *testing.T, sm *SourceMap, a, b int) {
	t.Helper()

	sa, _ := sm.ToSource(a, a)
	sb, _ := sm.ToSource(b, b)
	if sa > sb {
		t.Errorf("R4: ToSource(%d,%d)=(%d,_) > ToSource(%d,%d)=(%d,_): monotonicity violated",
			a, a, sa, b, b, sb)
	}
}

// ---------- Category A: Round-trip consistency ----------
// Targets: Bug 2 (asymmetry), Bug 3 (normalization masking)

func TestCorrectnessRoundTripPlainText(t *testing.T) {
	// Plain text: 1:1 mapping, trivial round-trip.
	input := "Hello, World!"
	_, sm, _ := ParseWithSourceMap(input)

	// Various sub-ranges
	assertRoundTripR1(t, sm, input, 0, 5)  // "Hello"
	assertRoundTripR1(t, sm, input, 7, 12) // "World"
	assertRoundTripR1(t, sm, input, 0, 13) // entire string
	assertRoundTripR1(t, sm, input, 5, 6)  // ","

	// R2: source→rendered→source
	assertRoundTripR2(t, sm, 0, 5)
	assertRoundTripR2(t, sm, 0, 13)
}

func TestCorrectnessRoundTripBold(t *testing.T) {
	// Bold: rendered "bold" (4 runes) → source "**bold**" (8 runes).
	input := "**bold**"
	_, sm, _ := ParseWithSourceMap(input)

	// Full rendered range
	assertRoundTripR1(t, sm, input, 0, 4)

	// R2: source→rendered→source for full source range
	assertRoundTripR2(t, sm, 0, 8)
	// R2: inner content only
	assertRoundTripR2(t, sm, 2, 6)
}

func TestCorrectnessRoundTripItalic(t *testing.T) {
	input := "*italic*"
	_, sm, _ := ParseWithSourceMap(input)

	assertRoundTripR1(t, sm, input, 0, 6)
	assertRoundTripR2(t, sm, 0, 8)
}

func TestCorrectnessRoundTripHeading(t *testing.T) {
	// Heading: rendered "Title" (5 runes) → source "# Title" (7 runes).
	input := "# Title"
	_, sm, _ := ParseWithSourceMap(input)

	assertRoundTripR1(t, sm, input, 0, 5)
	assertRoundTripR2(t, sm, 0, 7)
	// Content only in source
	assertRoundTripR2(t, sm, 2, 7)
}

func TestCorrectnessRoundTripMixed(t *testing.T) {
	// "# Title\nSome **bold** text\n"
	// Rendered: "Title\nSome bold text" (trailing \n stripped by parser)
	input := "# Title\nSome **bold** text\n"
	content, sm, _ := ParseWithSourceMap(input)

	rendered := string(content.Plain())
	rendLen := runeLen(rendered)

	// Heading portion
	assertRoundTripR1(t, sm, input, 0, 5) // "Title"

	// Bold portion: rendered "bold" at positions 11-15
	assertRoundTripR1(t, sm, input, 11, 15)

	// Plain text portions
	assertRoundTripR1(t, sm, input, 6, 11) // "\nSome "

	// " text" — trailing \n from source is not rendered
	assertRoundTripR1(t, sm, input, 15, rendLen) // " text"
}

func TestCorrectnessRoundTripCodeBlock(t *testing.T) {
	// Fenced code: rendered "code\n" maps to source content between fences.
	input := "```\ncode\n```"
	_, sm, _ := ParseWithSourceMap(input)

	assertRoundTripR1(t, sm, input, 0, 5) // "code\n"
}

func TestCorrectnessRoundTripListItem(t *testing.T) {
	// List: "- item\n" → rendered "• item\n"
	input := "- item\n"
	_, sm, _ := ParseWithSourceMap(input)

	// Full list item
	assertRoundTripR1(t, sm, input, 0, 7)
	// Content only
	assertRoundTripR1(t, sm, input, 2, 6)
}

func TestCorrectnessRoundTripMultipleParagraphs(t *testing.T) {
	// Two paragraphs with blank line between.
	input := "Para one.\n\nPara two."
	_, sm, _ := ParseWithSourceMap(input)

	rendered := plainText(input)
	rendLen := runeLen(rendered)

	// Full document
	if rendLen > 0 {
		assertRoundTripR1(t, sm, input, 0, rendLen)
	}
}

// ---------- Category B: Cross-boundary selections ----------
// Targets: Bug 4 (boundary lookup off-by-one)

func TestCorrectnessCrossBoundaryBoldToPlain(t *testing.T) {
	// Select from "bol" in bold into " text" in plain.
	// Source: "**bold** text" (13 runes)
	// Rendered: "bold text" (9 runes)
	input := "**bold** text"
	_, sm, _ := ParseWithSourceMap(input)

	// Select rendered 0-7 ("bold te") — crosses from bold entry into plain entry.
	srcStart, srcEnd := sm.ToSource(0, 7)

	// srcStart should include bold markup (at or before source 0)
	if srcStart > 0 {
		t.Errorf("Cross-boundary bold→plain: srcStart=%d, want <=0", srcStart)
	}
	// srcEnd should cover through "te" in " text" which is at source position 10
	if srcEnd < 10 {
		t.Errorf("Cross-boundary bold→plain: srcEnd=%d, want >=10", srcEnd)
	}

	// Verify round-trip
	assertRoundTripR1(t, sm, input, 0, 7)
}

func TestCorrectnessCrossBoundaryPlainToBold(t *testing.T) {
	// Select from "some " into "bol" in bold.
	// Source: "some **bold** text" (18 runes)
	// Rendered: "some bold text" (14 runes)
	input := "some **bold** text"
	_, sm, _ := ParseWithSourceMap(input)

	// Select rendered 0-8 ("some bol") — crosses from plain into bold.
	srcStart, srcEnd := sm.ToSource(0, 8)

	if srcStart != 0 {
		t.Errorf("Cross-boundary plain→bold: srcStart=%d, want 0", srcStart)
	}
	// srcEnd should be within the bold source ("**bold**" starts at 5)
	// "bol" is rendered 5-8, source "bol" is at 7-10 (after **)
	if srcEnd < 10 {
		t.Errorf("Cross-boundary plain→bold: srcEnd=%d, want >=10", srcEnd)
	}

	assertRoundTripR1(t, sm, input, 0, 8)
}

func TestCorrectnessCrossBoundaryHeadingToParagraph(t *testing.T) {
	// Select from heading "Titl" into body "Body".
	// Source: "# Title\nBody" (12 runes)
	// Rendered: "Title\nBody" (10 runes)
	input := "# Title\nBody"
	_, sm, _ := ParseWithSourceMap(input)

	// Select rendered 3-9 ("le\nBod") — crosses from heading into paragraph.
	srcStart, srcEnd := sm.ToSource(3, 9)

	// srcStart should be within the heading content area
	// rendered pos 3 = "l" in "Title", source = "# " (2) + 3 = rune 5
	if srcStart < 0 {
		t.Errorf("Cross-boundary heading→para: srcStart=%d, want >=0", srcStart)
	}
	// srcEnd should be in "Body" area of source
	if srcEnd < 11 {
		t.Errorf("Cross-boundary heading→para: srcEnd=%d, want >=11", srcEnd)
	}

	assertRoundTripR1(t, sm, input, 3, 9)
}

func TestCorrectnessCrossBoundaryListItems(t *testing.T) {
	// Select across two list items.
	// Source: "- one\n- two\n" (12 runes)
	// Rendered: "• one\n• two\n" (12 runes)
	input := "- one\n- two\n"
	_, sm, _ := ParseWithSourceMap(input)

	// Select rendered 3-9 ("e\n• tw") — crosses from first item into second.
	srcStart, srcEnd := sm.ToSource(3, 9)

	// Should cover from "e" in first item through "tw" in second
	if srcStart < 0 || srcEnd < 9 {
		t.Errorf("Cross-boundary list items: ToSource(3,9) = (%d,%d)", srcStart, srcEnd)
	}

	assertRoundTripR1(t, sm, input, 3, 9)
}

func TestCorrectnessCrossBoundaryCodeBlockToParagraph(t *testing.T) {
	// Select from inside code block to text after.
	// Source: "```\ncode\n```\nAfter" (18 bytes / runes)
	// Rendered: "code\nAfter" (10 runes)
	input := "```\ncode\n```\nAfter"
	_, sm, _ := ParseWithSourceMap(input)

	// Select rendered 2-8 ("de\nAft") — from code into paragraph.
	srcStart, _ := sm.ToSource(2, 8)

	// srcStart should be in code content area (source rune 6 = "d" in "code")
	if srcStart < 0 {
		t.Errorf("Cross-boundary code→para: srcStart=%d, want >=0", srcStart)
	}

	assertRoundTripR1(t, sm, input, 2, 8)
}

// ---------- Category C: Exact boundary positions ----------
// Targets: Bug 4 (boundary lookup), Bug 2 (expansion heuristic)

func TestCorrectnessExactBoundaryBold(t *testing.T) {
	// renderedEnd == entry.RenderedEnd for bold.
	// Source: "**bold**" (8 runes), rendered: "bold" (4 runes).
	input := "**bold**"
	_, sm, _ := ParseWithSourceMap(input)

	// Select exactly the bold entry: rendered [0, 4)
	srcStart, srcEnd := sm.ToSource(0, 4)
	// Should include full markup
	if srcStart != 0 || srcEnd != 8 {
		t.Errorf("Exact boundary bold: ToSource(0,4) = (%d,%d), want (0,8)", srcStart, srcEnd)
	}
}

func TestCorrectnessExactBoundaryHeading(t *testing.T) {
	// renderedStart == entry.RenderedStart and renderedEnd == entry.RenderedEnd for heading.
	input := "# Title"
	_, sm, _ := ParseWithSourceMap(input)

	// Select exactly the heading entry: rendered [0, 5)
	srcStart, srcEnd := sm.ToSource(0, 5)
	if srcStart != 0 || srcEnd != 7 {
		t.Errorf("Exact boundary heading: ToSource(0,5) = (%d,%d), want (0,7)", srcStart, srcEnd)
	}
}

func TestCorrectnessExactBoundaryItalic(t *testing.T) {
	input := "*italic*"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 6)
	if srcStart != 0 || srcEnd != 8 {
		t.Errorf("Exact boundary italic: ToSource(0,6) = (%d,%d), want (0,8)", srcStart, srcEnd)
	}
}

func TestCorrectnessExactBoundaryInlineCode(t *testing.T) {
	input := "`code`"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 4)
	if srcStart != 0 || srcEnd != 6 {
		t.Errorf("Exact boundary code: ToSource(0,4) = (%d,%d), want (0,6)", srcStart, srcEnd)
	}
}

func TestCorrectnessExactBoundaryTwoAdjacentEntries(t *testing.T) {
	// Two adjacent bold entries: "**a****b**"
	// Source: "**a****b**" — rendered "ab" (2 runes)
	// First bold: rendered [0,1), source [0,5) ("**a**")
	// Second bold: rendered [1,2), source [5,10) ("**b**")
	input := "**a****b**"
	_, sm, _ := ParseWithSourceMap(input)

	// Select spanning both entries exactly: rendered [0, 2)
	srcStart, srcEnd := sm.ToSource(0, 2)
	if srcStart != 0 || srcEnd != 10 {
		t.Errorf("Two adjacent bold: ToSource(0,2) = (%d,%d), want (0,10)", srcStart, srcEnd)
	}

	// Select just first entry: rendered [0, 1)
	srcStart, srcEnd = sm.ToSource(0, 1)
	if srcStart != 0 || srcEnd != 5 {
		t.Errorf("First bold: ToSource(0,1) = (%d,%d), want (0,5)", srcStart, srcEnd)
	}

	// Select just second entry: rendered [1, 2)
	srcStart, srcEnd = sm.ToSource(1, 2)
	if srcStart != 5 || srcEnd != 10 {
		t.Errorf("Second bold: ToSource(1,2) = (%d,%d), want (5,10)", srcStart, srcEnd)
	}
}

func TestCorrectnessExactBoundaryStartOfEntry(t *testing.T) {
	// renderedStart exactly at entry.RenderedStart for bold in middle.
	// Source: "text **bold** end" (17 runes)
	// Rendered: "text bold end" (13 runes)
	input := "text **bold** end"
	_, sm, _ := ParseWithSourceMap(input)

	// Select starting exactly at bold entry start: rendered [5, 9) = "bold"
	srcStart, srcEnd := sm.ToSource(5, 9)
	// Should include bold markup
	if srcStart != 5 || srcEnd != 13 {
		t.Errorf("Start of bold entry: ToSource(5,9) = (%d,%d), want (5,13)", srcStart, srcEnd)
	}
}

func TestCorrectnessExactBoundaryEndOfEntry(t *testing.T) {
	// renderedEnd exactly at entry.RenderedEnd for bold in middle.
	// Source: "text **bold** end" (17 runes)
	// Rendered: "text bold end" (13 runes)
	input := "text **bold** end"
	_, sm, _ := ParseWithSourceMap(input)

	// Select ending exactly at bold entry end: rendered [5, 9) = "bold"
	srcStart, srcEnd := sm.ToSource(5, 9)
	if srcStart != 5 || srcEnd != 13 {
		t.Errorf("End of bold entry: ToSource(5,9) = (%d,%d), want (5,13)", srcStart, srcEnd)
	}

	// Partial selection ending at entry boundary: rendered [6, 9) = "old"
	srcStart, srcEnd = sm.ToSource(6, 9)
	// srcStart should be offset into bold content, srcEnd should include closing **
	// rendered 6 maps to source 8 (after "**b"), but with boundary expansion srcEnd=13
	if srcEnd != 13 {
		t.Errorf("Partial to boundary: ToSource(6,9) srcEnd=%d, want 13", srcEnd)
	}
}

// ---------- Category D: Point selections ----------
// Targets: Bug 3 (normalization), Bug 1 (PrefixLen), Invariant R3

func TestCorrectnessPointSelectionHeadingStart(t *testing.T) {
	// Click at start of heading: rendered position 0 in "# Hello".
	input := "# Hello"
	_, sm, _ := ParseWithSourceMap(input)

	assertPointIdentity(t, sm, 0)
}

func TestCorrectnessPointSelectionHeadingMiddle(t *testing.T) {
	// Click in middle of heading content.
	input := "# Hello"
	_, sm, _ := ParseWithSourceMap(input)

	assertPointIdentity(t, sm, 3) // "l" in "Hello"

	// Verify the source position is in content area (past "# " prefix)
	srcStart, _ := sm.ToSource(3, 3)
	// Rendered pos 3 in "Hello" = source pos 5 ("# " + 3)
	if srcStart != 5 {
		t.Errorf("Point in heading middle: ToSource(3,3)=(%d,_), want (5,_)", srcStart)
	}
}

func TestCorrectnessPointSelectionBoldStart(t *testing.T) {
	// Click at start of bold: "**bold** text" at rendered position 0.
	input := "**bold** text"
	_, sm, _ := ParseWithSourceMap(input)

	assertPointIdentity(t, sm, 0)
}

func TestCorrectnessPointSelectionPlainBetweenFormatted(t *testing.T) {
	// Click in plain text between two formatted elements.
	// Source: "**a** x **b**" (13 runes)
	// Rendered: "a x b" (5 runes)
	input := "**a** x **b**"
	_, sm, _ := ParseWithSourceMap(input)

	// Click at position 2 ("x")
	assertPointIdentity(t, sm, 2)
}

func TestCorrectnessPointSelectionDocumentStart(t *testing.T) {
	// Click at position 0 of a plain document.
	input := "Hello"
	_, sm, _ := ParseWithSourceMap(input)

	assertPointIdentity(t, sm, 0)
}

func TestCorrectnessPointSelectionDocumentEnd(t *testing.T) {
	// Click at position at end of document.
	input := "Hello"
	_, sm, _ := ParseWithSourceMap(input)

	// Position 5 is one past end — this is a boundary case.
	// Some implementations may not have an entry for this position.
	// Just verify it doesn't panic and is still a point.
	srcStart, srcEnd := sm.ToSource(5, 5)
	if srcStart != srcEnd {
		t.Errorf("Point at document end: ToSource(5,5)=(%d,%d), want point", srcStart, srcEnd)
	}
}

func TestCorrectnessPointSelectionAllHeadingLevels(t *testing.T) {
	// Invariant R3 for all heading levels at position 0.
	headings := []string{
		"# H1",
		"## H2",
		"### H3",
		"#### H4",
		"##### H5",
		"###### H6",
	}

	for _, h := range headings {
		t.Run(h, func(t *testing.T) {
			_, sm, _ := ParseWithSourceMap(h)
			assertPointIdentity(t, sm, 0)
		})
	}
}

func TestCorrectnessPointSelectionMonotonicity(t *testing.T) {
	// Verify R4 (monotonicity) across a heading.
	// Rendered: "Hello" from "# Hello"
	input := "# Hello"
	_, sm, _ := ParseWithSourceMap(input)

	for i := 0; i < 4; i++ {
		assertMonotonicity(t, sm, i, i+1)
	}
}

func TestCorrectnessPointSelectionMonotonicityMixed(t *testing.T) {
	// Verify R4 across mixed content.
	input := "some **bold** text"
	_, sm, _ := ParseWithSourceMap(input)

	rendered := plainText(input)
	for i := 0; i < runeLen(rendered)-1; i++ {
		assertMonotonicity(t, sm, i, i+1)
	}
}

// ---------- Category E: Non-ASCII content ----------
// Targets: Bug 1 (PrefixLen byte/rune confusion)

func TestCorrectnessNonASCIIHeading(t *testing.T) {
	// "# Über" — heading with 2-byte rune ü (U+00FC).
	// Source: "# Über" = 7 runes (8 bytes due to ü).
	// Rendered: "Über" = 4 runes.
	input := "# Über"
	_, sm, _ := ParseWithSourceMap(input)

	// Full heading selection
	srcStart, srcEnd := sm.ToSource(0, 4) // "Über"
	if srcStart != 0 || srcEnd != 6 {
		t.Errorf("Non-ASCII heading full: ToSource(0,4) = (%d,%d), want (0,6)", srcStart, srcEnd)
	}

	// Point selection at start
	assertPointIdentity(t, sm, 0)

	// Point at position 1 ("b" in "Über")
	assertPointIdentity(t, sm, 1)

	// Round-trip
	assertRoundTripR1(t, sm, input, 0, 4)

	// Monotonicity
	for i := 0; i < 3; i++ {
		assertMonotonicity(t, sm, i, i+1)
	}
}

func TestCorrectnessNonASCIIBold(t *testing.T) {
	// "**café**" — bold with multi-byte é (U+00E9).
	// Source: "**café**" = 8 runes (9 bytes: **=2, café=5bytes/4runes, **=2).
	// Rendered: "café" = 4 runes.
	input := "**café**"
	content, sm, _ := ParseWithSourceMap(input)

	rendered := string(content.Plain())
	renderedRunes := runeLen(rendered)
	srcRunes := runeLen(input) // 8

	// Full selection
	srcStart, srcEnd := sm.ToSource(0, renderedRunes)
	if srcStart != 0 || srcEnd != srcRunes {
		t.Errorf("Non-ASCII bold full: ToSource(0,%d) = (%d,%d), want (0,%d)", renderedRunes, srcStart, srcEnd, srcRunes)
	}

	assertRoundTripR1(t, sm, input, 0, renderedRunes)
}

func TestCorrectnessNonASCIIHeadingCJK(t *testing.T) {
	// "# 日本語" — heading with 3-byte CJK runes.
	// Source: "# 日本語" = 5 runes (11 bytes).
	// Rendered: "日本語" = 3 runes.
	input := "# 日本語"
	_, sm, _ := ParseWithSourceMap(input)

	// Full heading selection
	srcStart, srcEnd := sm.ToSource(0, 3)
	if srcStart != 0 || srcEnd != 5 {
		t.Errorf("CJK heading full: ToSource(0,3) = (%d,%d), want (0,5)", srcStart, srcEnd)
	}

	// Point selections
	assertPointIdentity(t, sm, 0)
	assertPointIdentity(t, sm, 1)
	assertPointIdentity(t, sm, 2)

	// Monotonicity
	assertMonotonicity(t, sm, 0, 1)
	assertMonotonicity(t, sm, 1, 2)

	assertRoundTripR1(t, sm, input, 0, 3)
}

func TestCorrectnessNonASCIIBoldInMixed(t *testing.T) {
	// "Some **日本語** text" — bold multi-byte in mixed content.
	// Source runes: S,o,m,e, ,*,*,日,本,語,*,*, ,t,e,x,t = 17 runes.
	// Rendered runes: S,o,m,e, ,日,本,語, ,t,e,x,t = 13 runes.
	// Bold entry: source runes [5,12) = "**日本語**" (7 runes), rendered [5,8) = "日本語" (3 runes).
	input := "Some **日本語** text"
	_, sm, _ := ParseWithSourceMap(input)

	// Bold portion: rendered [5, 8) = "日本語"
	srcStart, srcEnd := sm.ToSource(5, 8)
	if srcStart != 5 || srcEnd != 12 {
		t.Errorf("CJK bold in mixed: ToSource(5,8) = (%d,%d), want (5,12)", srcStart, srcEnd)
	}

	assertRoundTripR1(t, sm, input, 5, 8)

	// Cross-boundary: from plain into CJK bold
	assertRoundTripR1(t, sm, input, 3, 7)
}

func TestCorrectnessNonASCIIHeadingFollowedByParagraph(t *testing.T) {
	// "# Ém\nSome text" — heading with multi-byte followed by paragraph.
	// Source: "# Ém\nSome text" = 15 runes (16 bytes due to É).
	// Note: É = U+00C9 = 2 bytes.
	// Rendered: heading "Ém\n" + paragraph "Some text"
	input := "# Ém\nSome text"
	_, sm, _ := ParseWithSourceMap(input)

	// Heading portion: rendered [0, 2) = "Ém"
	srcStart, _ := sm.ToSource(0, 2)
	if srcStart != 0 {
		t.Errorf("Non-ASCII heading+para: srcStart=%d, want 0", srcStart)
	}

	// Point selection in heading
	assertPointIdentity(t, sm, 0)
	assertPointIdentity(t, sm, 1)

	// Cross-boundary heading → paragraph
	assertRoundTripR1(t, sm, input, 1, 6)
}

// ---------- Category F: Edge positions ----------
// Targets: Bug 5 (bounds validation), robustness

func TestCorrectnessEdgeEmptyDocument(t *testing.T) {
	// Empty document: no entries.
	input := ""
	_, sm, _ := ParseWithSourceMap(input)

	// ToSource(0,0) on empty document
	srcStart, srcEnd := sm.ToSource(0, 0)
	if srcStart != 0 || srcEnd != 0 {
		t.Errorf("Empty doc ToSource(0,0) = (%d,%d), want (0,0)", srcStart, srcEnd)
	}

	// ToRendered(0,0) on empty document
	rendStart, rendEnd := sm.ToRendered(0, 0)
	// Empty source map returns -1,-1 which is the defined behavior.
	if rendStart != -1 || rendEnd != -1 {
		t.Errorf("Empty doc ToRendered(0,0) = (%d,%d), want (-1,-1)", rendStart, rendEnd)
	}
}

func TestCorrectnessEdgePosition0NonEmpty(t *testing.T) {
	// Position 0 in non-empty document.
	input := "Hello"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 0)
	if srcStart != 0 || srcEnd != 0 {
		t.Errorf("Position 0 point: ToSource(0,0) = (%d,%d), want (0,0)", srcStart, srcEnd)
	}
}

func TestCorrectnessEdgeDocumentEnd(t *testing.T) {
	// Position at exact document end.
	input := "Hello"
	_, sm, _ := ParseWithSourceMap(input)

	// renderedEnd = total rendered runes (5)
	srcStart, srcEnd := sm.ToSource(0, 5)
	if srcStart != 0 || srcEnd != 5 {
		t.Errorf("To document end: ToSource(0,5) = (%d,%d), want (0,5)", srcStart, srcEnd)
	}
}

func TestCorrectnessEdgeBeyondDocumentEnd(t *testing.T) {
	// Position beyond document end should not crash.
	input := "Hello"
	_, sm, _ := ParseWithSourceMap(input)

	// This tests robustness — should not panic.
	srcStart, srcEnd := sm.ToSource(10, 20)
	_ = srcStart
	_ = srcEnd
	// No assertions on values — just verify no panic.
}

func TestCorrectnessEdgeNegativePosition(t *testing.T) {
	// Negative position should not crash.
	input := "Hello"
	_, sm, _ := ParseWithSourceMap(input)

	// Should not panic.
	srcStart, srcEnd := sm.ToSource(-1, -1)
	_ = srcStart
	_ = srcEnd
}

func TestCorrectnessEdgeSingleCharDocument(t *testing.T) {
	// Single-character document.
	input := "X"
	_, sm, _ := ParseWithSourceMap(input)

	// Full selection
	srcStart, srcEnd := sm.ToSource(0, 1)
	if srcStart != 0 || srcEnd != 1 {
		t.Errorf("Single char: ToSource(0,1) = (%d,%d), want (0,1)", srcStart, srcEnd)
	}

	// Point at 0
	assertPointIdentity(t, sm, 0)

	// Point at 1 (end)
	srcStart, srcEnd = sm.ToSource(1, 1)
	if srcStart != srcEnd {
		t.Errorf("Single char end point: ToSource(1,1) = (%d,%d), want point", srcStart, srcEnd)
	}
}

func TestCorrectnessEdgeHeadingOnlyHash(t *testing.T) {
	// Edge case: heading with no content after prefix.
	// "# " — just prefix, no content. This may not parse as heading depending on parser.
	// But "# \n" might. Test for robustness.
	input := "# \n"
	_, sm, _ := ParseWithSourceMap(input)

	// Should not panic regardless of how it parses.
	srcStart, srcEnd := sm.ToSource(0, 0)
	_ = srcStart
	_ = srcEnd
}

// ---------- Composite invariant tests ----------

func TestCorrectnessInvariantR1Comprehensive(t *testing.T) {
	// Test R1 (rendered→source→rendered containment) across many document types.
	tests := []struct {
		name  string
		input string
	}{
		{"plain", "Hello, World!"},
		{"bold", "**bold**"},
		{"italic", "*italic*"},
		{"bold+italic", "***bolditalic***"},
		{"heading h1", "# Title"},
		{"heading h2", "## Title"},
		{"inline code", "`code`"},
		{"code block", "```\ncode\n```"},
		{"list item", "- item\n"},
		{"ordered list", "1. item\n"},
		{"mixed", "# Title\nSome **bold** and *italic* text\n"},
		{"link", "[click](http://example.com)"},
		{"image", "![alt](image.png)"},
		{"non-ascii heading", "# Über"},
		{"non-ascii bold", "**café**"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, sm, _ := ParseWithSourceMap(tt.input)
			rendered := string(content.Plain())
			rendLen := runeLen(rendered)

			if rendLen == 0 {
				return
			}

			// Test R1 for full rendered range
			assertRoundTripR1(t, sm, tt.input, 0, rendLen)

			// Test R1 for first character
			if rendLen >= 1 {
				assertRoundTripR1(t, sm, tt.input, 0, 1)
			}

			// Test R1 for last character
			if rendLen >= 1 {
				assertRoundTripR1(t, sm, tt.input, rendLen-1, rendLen)
			}
		})
	}
}

func TestCorrectnessInvariantR3Comprehensive(t *testing.T) {
	// Test R3 (point selection identity) at every position in several documents.
	tests := []struct {
		name  string
		input string
	}{
		{"plain", "Hello"},
		{"bold", "**bold**"},
		{"heading", "# Title"},
		{"mixed", "# Title\nSome **bold** text"},
		{"non-ascii heading", "# Über"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, sm, _ := ParseWithSourceMap(tt.input)
			rendered := string(content.Plain())
			rendLen := runeLen(rendered)

			for pos := 0; pos < rendLen; pos++ {
				assertPointIdentity(t, sm, pos)
			}
		})
	}
}

// ---------- Category H: Markup-boundary selection heuristics ----------
// Phase 1.4: When a selection boundary aligns with an entry boundary,
// expand the source position to include the corresponding markup delimiter.
//
// Design doc rule: "If q0 is at the start of a markup operation (ie, first
// text after bold, italic, code block, image, table, etc) the source q0
// should include the markup; likewise if on the last character, the trailing
// markup should be included."
//
// This means boundary expansion is INDEPENDENT at each end:
// - renderedStart == entry.RenderedStart → srcStart includes opening markup
// - renderedEnd == entry.RenderedEnd → srcEnd includes closing markup
// Both, either, or neither can apply in a given selection.

func TestMarkupBoundaryBoldFullSelection(t *testing.T) {
	// Selecting all of "bold" in "**bold**" should include both ** delimiters.
	// Source: "**bold**" = 8 runes. Rendered: "bold" = 4 runes.
	input := "**bold**"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 4) // full "bold"
	if srcStart != 0 || srcEnd != 8 {
		t.Errorf("Bold full selection: ToSource(0,4) = (%d,%d), want (0,8)", srcStart, srcEnd)
	}
}

func TestMarkupBoundaryBoldPartialFromStart(t *testing.T) {
	// Selecting "bol" from start of "**bold**": rendered [0,3).
	// Start aligns with entry → include opening **. End does NOT align → no closing **.
	// Source: opening ** included → srcStart=0. "bol" content → srcEnd=5.
	input := "**bold**"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 3) // "bol" from start
	if srcStart != 0 {
		t.Errorf("Bold partial 'bol': srcStart=%d, want 0 (include opening **)", srcStart)
	}
	if srcEnd != 5 {
		t.Errorf("Bold partial 'bol': srcEnd=%d, want 5 (no closing **)", srcEnd)
	}
}

func TestMarkupBoundaryBoldPartialFromEnd(t *testing.T) {
	// Selecting "old" at end of "**bold**": rendered [1,4).
	// Start does NOT align → no opening **. End aligns → include closing **.
	// srcStart: offset 1 within content → source 2 + 1 = 3.
	// srcEnd: end aligns with entry end → include closing ** → source 8.
	input := "**bold**"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(1, 4) // "old"
	if srcStart != 3 {
		t.Errorf("Bold partial 'old': srcStart=%d, want 3 (no opening **)", srcStart)
	}
	if srcEnd != 8 {
		t.Errorf("Bold partial 'old': srcEnd=%d, want 8 (include closing **)", srcEnd)
	}
}

func TestMarkupBoundaryBoldInteriorPartial(t *testing.T) {
	// Selecting "ol" from middle of "**bold**": rendered [1,3).
	// Neither boundary aligns → no delimiters included.
	// srcStart: offset 1 → source 3. srcEnd: offset 3 → source 5.
	input := "**bold**"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(1, 3) // "ol"
	if srcStart != 3 {
		t.Errorf("Bold interior 'ol': srcStart=%d, want 3", srcStart)
	}
	if srcEnd != 5 {
		t.Errorf("Bold interior 'ol': srcEnd=%d, want 5", srcEnd)
	}
}

func TestMarkupBoundaryItalicFullSelection(t *testing.T) {
	// Selecting all of "italic" in "*italic*" should include * delimiters.
	// Source: "*italic*" = 8 runes. Rendered: "italic" = 6 runes.
	input := "*italic*"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 6) // full "italic"
	if srcStart != 0 || srcEnd != 8 {
		t.Errorf("Italic full selection: ToSource(0,6) = (%d,%d), want (0,8)", srcStart, srcEnd)
	}
}

func TestMarkupBoundaryItalicPartialFromStart(t *testing.T) {
	// Selecting "ita" from start of "*italic*": rendered [0,3).
	// Start aligns → include opening *. End doesn't align → no closing *.
	// srcStart=0 (include *), srcEnd=4 (content only).
	input := "*italic*"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 3) // "ita" from start
	if srcStart != 0 {
		t.Errorf("Italic partial 'ita': srcStart=%d, want 0 (include opening *)", srcStart)
	}
	if srcEnd != 4 {
		t.Errorf("Italic partial 'ita': srcEnd=%d, want 4 (no closing *)", srcEnd)
	}
}

func TestMarkupBoundaryItalicPartialFromEnd(t *testing.T) {
	// Selecting "lic" at end of "*italic*": rendered [3,6).
	// Start doesn't align → no opening *. End aligns → include closing *.
	// srcStart = 1 + 3 = 4. srcEnd = 8 (include closing *).
	input := "*italic*"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(3, 6) // "lic"
	if srcStart != 4 {
		t.Errorf("Italic partial 'lic': srcStart=%d, want 4 (no opening *)", srcStart)
	}
	if srcEnd != 8 {
		t.Errorf("Italic partial 'lic': srcEnd=%d, want 8 (include closing *)", srcEnd)
	}
}

func TestMarkupBoundaryCodeFullSelection(t *testing.T) {
	// Selecting all of "code" in "`code`" should include ` delimiters.
	// Source: "`code`" = 6 runes. Rendered: "code" = 4 runes.
	input := "`code`"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 4) // full "code"
	if srcStart != 0 || srcEnd != 6 {
		t.Errorf("Code full selection: ToSource(0,4) = (%d,%d), want (0,6)", srcStart, srcEnd)
	}
}

func TestMarkupBoundaryCodePartialFromStart(t *testing.T) {
	// Selecting "cod" from start of "`code`": rendered [0,3).
	// Start aligns → include opening `. End doesn't → no closing `.
	// srcStart=0, srcEnd=4.
	input := "`code`"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 3) // "cod" from start
	if srcStart != 0 {
		t.Errorf("Code partial 'cod': srcStart=%d, want 0 (include opening `)", srcStart)
	}
	if srcEnd != 4 {
		t.Errorf("Code partial 'cod': srcEnd=%d, want 4 (no closing `)", srcEnd)
	}
}

func TestMarkupBoundaryHeadingFullSelection(t *testing.T) {
	// Selecting all of "heading" in "# heading" should include # prefix.
	// Source: "# heading" = 9 runes. Rendered: "heading" = 7 runes.
	input := "# heading"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 7) // full "heading"
	if srcStart != 0 || srcEnd != 9 {
		t.Errorf("Heading full selection: ToSource(0,7) = (%d,%d), want (0,9)", srcStart, srcEnd)
	}
}

func TestMarkupBoundaryHeadingPartialFromStart(t *testing.T) {
	// Selecting "hea" from start of "# heading": rendered [0,3).
	// Start aligns → include # prefix in source. End doesn't → content only.
	// srcStart=0 (include "# "), srcEnd=5.
	input := "# heading"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 3) // "hea" from start
	if srcStart != 0 {
		t.Errorf("Heading partial 'hea': srcStart=%d, want 0 (include '# ' prefix)", srcStart)
	}
	if srcEnd != 5 {
		t.Errorf("Heading partial 'hea': srcEnd=%d, want 5", srcEnd)
	}
}

func TestMarkupBoundaryHeadingPartialFromEnd(t *testing.T) {
	// Selecting "ing" at end of "# heading": rendered [4,7).
	// Start doesn't align → no prefix. End aligns → include trailing content.
	// For headings, there is no closing markup, so srcEnd = entry.SourceRuneEnd = 9.
	// srcStart: offset 4 in rendered → source 2 + 4 = 6.
	input := "# heading"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(4, 7) // "ing" at end
	if srcStart != 6 {
		t.Errorf("Heading partial 'ing': srcStart=%d, want 6 (no prefix)", srcStart)
	}
	if srcEnd != 9 {
		t.Errorf("Heading partial 'ing': srcEnd=%d, want 9 (entry end)", srcEnd)
	}
}

func TestMarkupBoundaryLinkFullSelection(t *testing.T) {
	// Selecting all of "link" in "[link](url)" should include full markup.
	// Source: "[link](url)" = 11 runes. Rendered: "link" = 4 runes.
	// The link creates per-character entries for the text ("l","i","n","k"),
	// each mapping 1:1 to source positions within "[link]" (source 1-5).
	// With boundary expansion:
	// - Start: renderedStart=0 == first entry.RenderedStart → srcStart = entry.SourceRuneStart = 1
	//   (source position of "l", not of "["). Links need special handling since
	//   the "[" and "](url)" are not part of any entry.
	// - End: renderedEnd=4 == last entry.RenderedEnd → srcEnd = entry.SourceRuneEnd = 5
	//   (source position past "k", not past ")").
	// The per-character entries don't know about the surrounding link syntax.
	// This test documents the current limitation: link delimiters are NOT included
	// because the entries only cover the link text, not the full markup.
	input := "[link](url)"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 4) // full "link"
	if srcStart > 1 {
		t.Errorf("Link full selection: srcStart=%d, want <=1", srcStart)
	}
	if srcEnd < 5 {
		t.Errorf("Link full selection: srcEnd=%d, want >=5 (at least covering 'link')", srcEnd)
	}
}

func TestMarkupBoundaryBoldInMiddleFullEntry(t *testing.T) {
	// Bold in middle of text: "text **bold** end"
	// Source: 17 runes. Rendered: "text bold end" = 13 runes.
	// Bold "bold" at rendered [5,9), source entry for "**bold**" at source [5,13).
	input := "text **bold** end"
	_, sm, _ := ParseWithSourceMap(input)

	// Full bold: rendered [5,9) — both boundaries align with entry.
	// Should include ** delimiters → source [5,13).
	srcStart, srcEnd := sm.ToSource(5, 9)
	if srcStart != 5 || srcEnd != 13 {
		t.Errorf("Bold in middle full: ToSource(5,9) = (%d,%d), want (5,13)", srcStart, srcEnd)
	}
}

func TestMarkupBoundaryBoldInMiddlePartialFromStart(t *testing.T) {
	// "text **bold** end" — select "bol" from start of bold entry: rendered [5,8).
	// Start aligns → include opening **. End doesn't → no closing **.
	// srcStart = entry.SourceRuneStart = 5 (include **). srcEnd = 5 + 2 + 3 = 10.
	input := "text **bold** end"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(5, 8) // "bol" from start of bold
	if srcStart != 5 {
		t.Errorf("Bold middle partial 'bol': srcStart=%d, want 5 (include opening **)", srcStart)
	}
	if srcEnd != 10 {
		t.Errorf("Bold middle partial 'bol': srcEnd=%d, want 10 (no closing **)", srcEnd)
	}
}

func TestMarkupBoundaryBoldInMiddlePartialFromEnd(t *testing.T) {
	// "text **bold** end" — select "old" at end of bold entry: rendered [6,9).
	// Start doesn't align → no opening **. End aligns → include closing **.
	// srcStart: offset 1 in content → source 7 + 1 = 8. srcEnd: entry.SourceRuneEnd = 13.
	input := "text **bold** end"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(6, 9) // "old" ending at bold boundary
	if srcStart != 8 {
		t.Errorf("Bold middle partial 'old': srcStart=%d, want 8 (no opening **)", srcStart)
	}
	if srcEnd != 13 {
		t.Errorf("Bold middle partial 'old': srcEnd=%d, want 13 (include closing **)", srcEnd)
	}
}

func TestMarkupBoundaryBoldItalicFullSelection(t *testing.T) {
	// "***bi***" — bold+italic with 3-char delimiters.
	// Source: "***bi***" = 8 runes. Rendered: "bi" = 2 runes.
	input := "***bi***"
	_, sm, _ := ParseWithSourceMap(input)

	// Full selection: rendered [0,2) → source [0,8)
	srcStart, srcEnd := sm.ToSource(0, 2)
	if srcStart != 0 || srcEnd != 8 {
		t.Errorf("BoldItalic full: ToSource(0,2) = (%d,%d), want (0,8)", srcStart, srcEnd)
	}
}

func TestMarkupBoundaryBoldItalicPartialFromStart(t *testing.T) {
	// "***bi***" — select just "b": rendered [0,1).
	// Start aligns → include opening ***. End doesn't → no closing ***.
	// srcStart=0 (include ***), srcEnd=4.
	input := "***bi***"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 1) // just "b"
	if srcStart != 0 {
		t.Errorf("BoldItalic partial 'b': srcStart=%d, want 0 (include opening ***)", srcStart)
	}
	if srcEnd != 4 {
		t.Errorf("BoldItalic partial 'b': srcEnd=%d, want 4 (no closing ***)", srcEnd)
	}
}

func TestMarkupBoundaryH2FullSelection(t *testing.T) {
	// "## Title" — heading level 2.
	// Source: "## Title" = 8 runes. Prefix "## " = 3 runes. Rendered: "Title" = 5 runes.
	input := "## Title"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 5) // full "Title"
	if srcStart != 0 || srcEnd != 8 {
		t.Errorf("H2 full selection: ToSource(0,5) = (%d,%d), want (0,8)", srcStart, srcEnd)
	}
}

func TestMarkupBoundaryH2PartialFromStart(t *testing.T) {
	// "## Title" — partial "Tit": rendered [0,3).
	// Start aligns → include "## " prefix. End doesn't → content only.
	// srcStart=0, srcEnd=6.
	input := "## Title"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 3) // "Tit" from start
	if srcStart != 0 {
		t.Errorf("H2 partial 'Tit': srcStart=%d, want 0 (include '## ' prefix)", srcStart)
	}
	if srcEnd != 6 {
		t.Errorf("H2 partial 'Tit': srcEnd=%d, want 6", srcEnd)
	}
}

func TestMarkupBoundaryCrossBoundaryNoExpansion(t *testing.T) {
	// When a selection spans from inside one entry into another, neither
	// entry is fully selected at the selection boundary, so no markup
	// expansion should occur for either.
	// "**aa** **bb**": rendered "aa bb" (5 runes).
	// First bold: rendered [0,2), source [0,6) "**aa**"
	// Space: rendered [2,3), source [6,7) " "
	// Second bold: rendered [3,5), source [7,13) "**bb**"
	input := "**aa** **bb**"
	_, sm, _ := ParseWithSourceMap(input)

	// Select from middle of first bold to middle of second: rendered [1,4) = "a b"
	srcStart, srcEnd := sm.ToSource(1, 4)
	// srcStart in first bold content: offset 1 → source 2 + 1 = 3 (no opening **)
	if srcStart != 3 {
		t.Errorf("Cross-boundary no expand: srcStart=%d, want 3", srcStart)
	}
	// srcEnd in second bold content: offset 1 → source 7 + 2 + 1 = 10 (no closing **)
	if srcEnd != 10 {
		t.Errorf("Cross-boundary no expand: srcEnd=%d, want 10", srcEnd)
	}
}

func TestMarkupBoundaryCrossBoundaryStartExpands(t *testing.T) {
	// Selection starts at entry start of first bold but ends mid-second bold.
	// "**aa** **bb**": rendered "aa bb" (5 runes).
	// Select rendered [0,4) = "aa b" — start aligns with first bold entry.
	input := "**aa** **bb**"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 4)
	// Start aligns with first bold entry → include opening ** → srcStart=0
	if srcStart != 0 {
		t.Errorf("Cross-boundary start expands: srcStart=%d, want 0 (include opening **)", srcStart)
	}
	// End doesn't align with second bold entry end → no closing **
	if srcEnd != 10 {
		t.Errorf("Cross-boundary start expands: srcEnd=%d, want 10 (no closing **)", srcEnd)
	}
}

func TestMarkupBoundaryCrossBoundaryEndExpands(t *testing.T) {
	// Selection starts mid-first bold but ends at entry end of second bold.
	// "**aa** **bb**": rendered "aa bb" (5 runes).
	// Select rendered [1,5) = "a bb" — end aligns with second bold entry.
	input := "**aa** **bb**"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(1, 5)
	// Start doesn't align with first bold entry start → no opening **
	if srcStart != 3 {
		t.Errorf("Cross-boundary end expands: srcStart=%d, want 3 (no opening **)", srcStart)
	}
	// End aligns with second bold entry end → include closing ** → srcEnd=13
	if srcEnd != 13 {
		t.Errorf("Cross-boundary end expands: srcEnd=%d, want 13 (include closing **)", srcEnd)
	}
}

func TestMarkupBoundaryFullEntryAmongPlainText(t *testing.T) {
	// "before **bold** after" — select exactly "bold" (full entry).
	// Source: "before **bold** after" = 21 runes.
	// Rendered: "before bold after" = 17 runes.
	// Bold "bold" at rendered [7,11), source entry for "**bold**" at [7,15).
	input := "before **bold** after"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(7, 11) // exactly "bold"
	if srcStart != 7 || srcEnd != 15 {
		t.Errorf("Full bold among text: ToSource(7,11) = (%d,%d), want (7,15)", srcStart, srcEnd)
	}
}

func TestMarkupBoundaryImageFullSelection(t *testing.T) {
	// "![alt](image.png)" — image with alt text.
	// Rendered: "[Image: alt]" = 12 runes. Source: "![alt](image.png)" = 17 runes.
	input := "![alt](image.png)"
	_, sm, _ := ParseWithSourceMap(input)

	rendered := plainText(input)
	rendLen := runeLen(rendered)

	// Full selection of the image placeholder — both boundaries align.
	srcStart, srcEnd := sm.ToSource(0, rendLen)
	if srcStart != 0 || srcEnd != 17 {
		t.Errorf("Image full selection: ToSource(0,%d) = (%d,%d), want (0,17)", rendLen, srcStart, srcEnd)
	}
}

func TestMarkupBoundaryImagePartialSelection(t *testing.T) {
	// "![alt](image.png)" — partial selection of image placeholder text.
	// Rendered: "[Image: alt]" = 12 runes.
	// Partial: rendered [0,3) = "[Im" — start aligns, end doesn't.
	// Since this is a single entry, start expansion includes opening markup
	// but end does NOT include closing markup.
	input := "![alt](image.png)"
	_, sm, _ := ParseWithSourceMap(input)

	srcStart, srcEnd := sm.ToSource(0, 3)
	// Start aligns → include opening markup. srcStart = entry.SourceRuneStart = 0.
	if srcStart != 0 {
		t.Errorf("Image partial: srcStart=%d, want 0 (include opening markup)", srcStart)
	}
	// End doesn't align → no full closing markup.
	// Source 17 would mean full markup; partial should be less.
	if srcEnd >= 17 {
		t.Errorf("Image partial: srcEnd=%d, want <17 (no closing markup)", srcEnd)
	}
}

func TestCorrectnessInvariantR4Comprehensive(t *testing.T) {
	// Test R4 (monotonicity) across every consecutive pair of positions.
	tests := []struct {
		name  string
		input string
	}{
		{"plain", "Hello, World!"},
		{"bold in text", "some **bold** text"},
		{"heading + body", "# Title\nBody text"},
		{"non-ascii", "# Über café"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, sm, _ := ParseWithSourceMap(tt.input)
			rendered := string(content.Plain())
			rendLen := runeLen(rendered)

			for i := 0; i < rendLen-1; i++ {
				assertMonotonicity(t, sm, i, i+1)
			}
		})
	}
}
