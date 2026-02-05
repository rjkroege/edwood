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
