package markdown

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/rjkroege/edwood/rich"
)

// --- Helper functions for tests ---

// contentEqual returns true if two Content slices are equal (same spans
// with same text and style).
func contentEqual(a, b rich.Content) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Text != b[i].Text || a[i].Style != b[i].Style {
			return false
		}
	}
	return true
}

// sourceMapEqual returns true if two SourceMaps have identical entries.
func sourceMapEqual(a, b *SourceMap) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a.entries) != len(b.entries) {
		return false
	}
	for i := range a.entries {
		if a.entries[i] != b.entries[i] {
			return false
		}
	}
	return true
}

// linkMapEqual returns true if two LinkMaps have identical entries.
func linkMapEqual(a, b *LinkMap) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a.entries) != len(b.entries) {
		return false
	}
	for i := range a.entries {
		if a.entries[i] != b.entries[i] {
			return false
		}
	}
	return true
}

// applyEdit applies an EditRecord to a source string, returning the new source.
// For inserts, insertText is the text to insert. For deletes, insertText is "".
func applyEdit(source string, edit EditRecord, insertText string) string {
	runes := []rune(source)
	pos := edit.Pos
	if pos > len(runes) {
		pos = len(runes)
	}
	endPos := pos + edit.OldLen
	if endPos > len(runes) {
		endPos = len(runes)
	}
	result := make([]rune, 0, len(runes)-edit.OldLen+len([]rune(insertText)))
	result = append(result, runes[:pos]...)
	result = append(result, []rune(insertText)...)
	result = append(result, runes[endPos:]...)
	return string(result)
}

// runeCount returns the rune count of a string.
func testRuneCount(s string) int {
	return utf8.RuneCountInString(s)
}

// --- ParseRegion tests ---

// TestParseRegionSingleParagraph verifies that ParseRegion for a paragraph
// produces content matching what ParseWithSourceMap would produce for the
// same text, with source offsets adjusted.
func TestParseRegionSingleParagraph(t *testing.T) {
	// Full document: heading + blank + paragraph
	fullDoc := "# Title\n\nHello world.\n"
	lines := splitLines(fullDoc)

	// Parse the full doc.
	fullContent, fullSM, _ := ParseWithSourceMap(fullDoc)
	_ = fullContent
	_ = fullSM

	// Parse just the paragraph region (lines 2 onward: "Hello world.\n").
	// Line 0: "# Title\n"
	// Line 1: "\n"
	// Line 2: "Hello world.\n"
	paraLines := lines[2:]

	// sourceOffset = rune position of line 2 in the full source.
	sourceOffset := 0
	for _, l := range lines[:2] {
		sourceOffset += testRuneCount(l)
	}

	regionContent, regionSM, regionLM := ParseRegion(paraLines, sourceOffset)

	// The region content should be non-empty.
	if len(regionContent) == 0 {
		t.Fatal("ParseRegion returned empty content for paragraph")
	}

	// The region content text should match the paragraph text.
	regionText := ""
	for _, s := range regionContent {
		regionText += s.Text
	}
	if !strings.Contains(regionText, "Hello world.") {
		t.Errorf("region content %q doesn't contain paragraph text", regionText)
	}

	// Source map byte positions should be shifted to reference the full document.
	// Note: rune positions are NOT shifted by ParseRegion — the caller must
	// call PopulateRunePositions(fullSource) after stitching.
	if regionSM != nil && len(regionSM.entries) > 0 {
		// Compute byte offset for this all-ASCII text.
		byteOffset := len(strings.Join(lines[:2], ""))
		firstEntry := regionSM.entries[0]
		if firstEntry.SourceStart < byteOffset {
			t.Errorf("source map entry SourceStart %d < byteOffset %d",
				firstEntry.SourceStart, byteOffset)
		}
	}

	_ = regionLM
}

// TestParseRegionHeading verifies that ParseRegion for a heading line
// produces correct content and source map with offset.
func TestParseRegionHeading(t *testing.T) {
	fullDoc := "Some text.\n\n## Section\n\nMore text.\n"
	lines := splitLines(fullDoc)

	// Find the heading line.
	headingLineIdx := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "## ") {
			headingLineIdx = i
			break
		}
	}
	if headingLineIdx < 0 {
		t.Fatal("no heading line found")
	}

	// Compute source offset for the heading line.
	sourceOffset := 0
	for _, l := range lines[:headingLineIdx] {
		sourceOffset += testRuneCount(l)
	}

	// Parse just the heading line.
	regionContent, regionSM, _ := ParseRegion(lines[headingLineIdx:headingLineIdx+1], sourceOffset)

	if len(regionContent) == 0 {
		t.Fatal("ParseRegion returned empty content for heading")
	}

	// Should contain "Section" text.
	regionText := ""
	for _, s := range regionContent {
		regionText += s.Text
	}
	if !strings.Contains(regionText, "Section") {
		t.Errorf("heading region %q doesn't contain 'Section'", regionText)
	}

	// Source map byte positions should reference positions in the full document.
	if regionSM != nil && len(regionSM.entries) > 0 {
		byteOffset := len(strings.Join(lines[:headingLineIdx], ""))
		for _, e := range regionSM.entries {
			if e.SourceStart < byteOffset {
				t.Errorf("source map entry SourceStart %d < byteOffset %d",
					e.SourceStart, byteOffset)
			}
		}
	}
}

// TestParseRegionFencedCode verifies that ParseRegion handles a fenced
// code block correctly.
func TestParseRegionFencedCode(t *testing.T) {
	fullDoc := "Text before.\n\n```\nfoo := 1\nbar := 2\n```\n\nText after.\n"
	lines := splitLines(fullDoc)

	// Find the fenced code block lines.
	fenceStart := -1
	fenceEnd := -1
	for i, l := range lines {
		trimmed := strings.TrimRight(l, "\n")
		if strings.HasPrefix(trimmed, "```") {
			if fenceStart < 0 {
				fenceStart = i
			} else {
				fenceEnd = i + 1 // exclusive
				break
			}
		}
	}
	if fenceStart < 0 || fenceEnd < 0 {
		t.Fatal("could not find fenced code block")
	}

	sourceOffset := 0
	for _, l := range lines[:fenceStart] {
		sourceOffset += testRuneCount(l)
	}

	regionContent, regionSM, _ := ParseRegion(lines[fenceStart:fenceEnd], sourceOffset)

	if len(regionContent) == 0 {
		t.Fatal("ParseRegion returned empty content for fenced code")
	}

	// Should contain code text.
	regionText := ""
	for _, s := range regionContent {
		regionText += s.Text
	}
	if !strings.Contains(regionText, "foo := 1") {
		t.Errorf("fenced code region %q doesn't contain code text", regionText)
	}

	_ = regionSM
}

// TestParseRegionWithLinks verifies that ParseRegion correctly tracks
// links with offset positions.
func TestParseRegionWithLinks(t *testing.T) {
	fullDoc := "# Heading\n\nClick [here](http://example.com) for info.\n"
	lines := splitLines(fullDoc)

	// Parse the link paragraph (line 2).
	sourceOffset := 0
	for _, l := range lines[:2] {
		sourceOffset += testRuneCount(l)
	}

	regionContent, _, regionLM := ParseRegion(lines[2:], sourceOffset)

	if len(regionContent) == 0 {
		t.Fatal("ParseRegion returned empty content for paragraph with link")
	}

	// Link map should contain the link.
	if regionLM == nil || len(regionLM.entries) == 0 {
		t.Fatal("ParseRegion returned no link map entries for paragraph with link")
	}

	foundLink := false
	for _, e := range regionLM.entries {
		if e.URL == "http://example.com" {
			foundLink = true
			break
		}
	}
	if !foundLink {
		t.Error("link map missing http://example.com entry")
	}
}

// --- StitchResult and Stitch tests ---

// TestStitchResultStructure verifies the StitchResult type holds all
// necessary fields.
func TestStitchResultStructure(t *testing.T) {
	sr := StitchResult{
		Content:  rich.Content{{Text: "hello", Style: rich.DefaultStyle()}},
		SM:       &SourceMap{},
		LM:       &LinkMap{},
		BlockIdx: &BlockIndex{},
	}

	if len(sr.Content) != 1 {
		t.Errorf("expected 1 content span, got %d", len(sr.Content))
	}
	if sr.SM == nil || sr.LM == nil || sr.BlockIdx == nil {
		t.Error("StitchResult fields should not be nil")
	}
}

// --- Equivalence tests: incremental vs full re-parse ---

// incrementalUpdate performs an incremental update and returns the stitched
// result. It applies the edit to the old source, computes the affected
// range, parses the affected region, and stitches it back.
// Returns nil if AffectedRange signals full re-parse (-1, -1).
func incrementalUpdate(t *testing.T, oldSource, newSource string, edit EditRecord) *StitchResult {
	t.Helper()

	// Parse the old document to get the block index.
	oldContent, oldSM, oldLM, oldBI := ParseWithSourceMapAndIndex(oldSource)
	if oldBI == nil {
		t.Fatal("nil BlockIndex from old source")
	}

	// Find affected blocks.
	startBlock, endBlock := oldBI.AffectedRange([]EditRecord{edit})
	if startBlock == -1 || endBlock == -1 {
		return nil // full re-parse required
	}

	// Get old lines and new lines.
	newLines := splitLines(newSource)

	// Compute the source rune range of affected blocks in old source.
	oldBlockStart := oldBI.Blocks[startBlock].SourceRuneStart
	var oldBlockEnd int
	if endBlock <= len(oldBI.Blocks) {
		lastBlock := endBlock - 1
		if lastBlock < len(oldBI.Blocks) {
			oldBlockEnd = oldBI.Blocks[lastBlock].SourceRuneEnd
		}
	}

	// Compute source deltas (rune and byte).
	sourceDelta := edit.NewLen - edit.OldLen
	sourceBytesDelta := len(newSource) - len(oldSource)

	// Find new block boundaries: the affected region in the new source
	// spans from oldBlockStart to oldBlockEnd + sourceDelta.
	newBlockEnd := oldBlockEnd + sourceDelta
	if newBlockEnd < oldBlockStart {
		newBlockEnd = oldBlockStart
	}

	// Find line range in the new source that covers [oldBlockStart, newBlockEnd).
	newLineStart := 0
	newLineEnd := len(newLines)
	runePos := 0
	for i, l := range newLines {
		lineEnd := runePos + testRuneCount(l)
		if runePos <= oldBlockStart && lineEnd > oldBlockStart {
			newLineStart = i
		}
		if lineEnd >= newBlockEnd && newLineEnd == len(newLines) {
			newLineEnd = i + 1
			break
		}
		runePos = lineEnd
	}

	// Compute source rune offset and byte offset for the new region.
	sourceRuneOffset := 0
	sourceByteOffset := 0
	for _, l := range newLines[:newLineStart] {
		sourceRuneOffset += testRuneCount(l)
		sourceByteOffset += len(l)
	}

	// Parse the affected region.
	regionContent, regionSM, regionLM := ParseRegion(newLines[newLineStart:newLineEnd], sourceRuneOffset, sourceByteOffset)

	// Build old and new StitchResults.
	oldResult := StitchResult{
		Content:  oldContent,
		SM:       oldSM,
		LM:       oldLM,
		BlockIdx: oldBI,
	}
	newRegion := StitchResult{
		Content:  regionContent,
		SM:       regionSM,
		LM:       regionLM,
		BlockIdx: buildBlockIndex(strings.Join(newLines[newLineStart:newLineEnd], "")),
	}

	result := Stitch(oldResult, newRegion, startBlock, endBlock, sourceDelta, sourceBytesDelta)

	// Re-populate rune positions from globally-correct byte positions
	// using the full new source text. This ensures byte-to-rune conversion
	// matches what ParseWithSourceMap produces for the full document.
	result.SM.PopulateRunePositions(newSource)

	return &result
}

// TestIncrementalEquivalenceEditInParagraph verifies that editing within
// a paragraph produces the same result incrementally as a full re-parse.
func TestIncrementalEquivalenceEditInParagraph(t *testing.T) {
	oldSource := "# Title\n\nSome text here.\n\nAnother paragraph.\n"
	// Insert " extra" after "Some" (position of 'S' + 4 = position after "Some").
	insertText := " extra"
	editPos := testRuneCount("# Title\n\n") + 4 // after "Some"
	edit := EditRecord{Pos: editPos, OldLen: 0, NewLen: testRuneCount(insertText)}
	newSource := applyEdit(oldSource, edit, insertText)

	// Full re-parse of new source.
	fullContent, fullSM, fullLM := ParseWithSourceMap(newSource)

	// Incremental path.
	result := incrementalUpdate(t, oldSource, newSource, edit)
	if result == nil {
		t.Log("AffectedRange returned full re-parse (conservative, acceptable)")
		return
	}

	// Compare content.
	if !contentEqual(fullContent, result.Content) {
		t.Errorf("content mismatch:\n  full: %v\n  incr: %v", fullContent, result.Content)
	}

	// Compare source maps.
	if !sourceMapEqual(fullSM, result.SM) {
		t.Errorf("source map mismatch:\n  full entries: %d\n  incr entries: %d",
			len(fullSM.entries), len(result.SM.entries))
		for i := 0; i < len(fullSM.entries) && i < len(result.SM.entries); i++ {
			if fullSM.entries[i] != result.SM.entries[i] {
				t.Errorf("  entry %d: full=%+v incr=%+v", i, fullSM.entries[i], result.SM.entries[i])
			}
		}
	}

	// Compare link maps.
	if !linkMapEqual(fullLM, result.LM) {
		t.Errorf("link map mismatch: full %d entries, incr %d entries",
			len(fullLM.entries), len(result.LM.entries))
	}
}

// TestIncrementalEquivalenceDeleteInParagraph verifies that deleting text
// within a paragraph produces the same result incrementally as full re-parse.
func TestIncrementalEquivalenceDeleteInParagraph(t *testing.T) {
	oldSource := "# Title\n\nSome extra text here.\n\nAnother paragraph.\n"
	// Delete " extra" (6 chars) after "Some".
	editPos := testRuneCount("# Title\n\n") + 4
	edit := EditRecord{Pos: editPos, OldLen: 6, NewLen: 0}
	newSource := applyEdit(oldSource, edit, "")

	fullContent, fullSM, fullLM := ParseWithSourceMap(newSource)

	result := incrementalUpdate(t, oldSource, newSource, edit)
	if result == nil {
		t.Log("full re-parse fallback (acceptable)")
		return
	}

	if !contentEqual(fullContent, result.Content) {
		t.Errorf("content mismatch after delete")
	}
	if !sourceMapEqual(fullSM, result.SM) {
		t.Errorf("source map mismatch after delete")
	}
	if !linkMapEqual(fullLM, result.LM) {
		t.Errorf("link map mismatch after delete")
	}
}

// TestIncrementalEquivalenceEditInCodeBlock verifies that editing inside
// a fenced code block produces the same result incrementally.
func TestIncrementalEquivalenceEditInCodeBlock(t *testing.T) {
	oldSource := "# Title\n\n```\nold code\n```\n\nAfter.\n"
	// Replace "old" with "new" inside the code block.
	editPos := testRuneCount("# Title\n\n```\n")
	edit := EditRecord{Pos: editPos, OldLen: 3, NewLen: 3}
	newSource := applyEdit(oldSource, edit, "new")

	fullContent, fullSM, fullLM := ParseWithSourceMap(newSource)

	result := incrementalUpdate(t, oldSource, newSource, edit)
	if result == nil {
		t.Log("full re-parse fallback for code block edit (acceptable)")
		return
	}

	if !contentEqual(fullContent, result.Content) {
		t.Errorf("content mismatch for code block edit")
	}
	if !sourceMapEqual(fullSM, result.SM) {
		t.Errorf("source map mismatch for code block edit")
	}
	if !linkMapEqual(fullLM, result.LM) {
		t.Errorf("link map mismatch for code block edit")
	}
}

// TestIncrementalEquivalenceEditInListItem verifies that editing a list
// item produces the same result incrementally.
func TestIncrementalEquivalenceEditInListItem(t *testing.T) {
	oldSource := "- item one\n- item two\n- item three\n"
	// Insert " modified" after "item two".
	insertText := " modified"
	editPos := testRuneCount("- item one\n- item two")
	edit := EditRecord{Pos: editPos, OldLen: 0, NewLen: testRuneCount(insertText)}
	newSource := applyEdit(oldSource, edit, insertText)

	fullContent, fullSM, fullLM := ParseWithSourceMap(newSource)

	result := incrementalUpdate(t, oldSource, newSource, edit)
	if result == nil {
		t.Log("full re-parse fallback (acceptable)")
		return
	}

	if !contentEqual(fullContent, result.Content) {
		t.Errorf("content mismatch for list item edit")
	}
	if !sourceMapEqual(fullSM, result.SM) {
		t.Errorf("source map mismatch for list item edit")
	}
	if !linkMapEqual(fullLM, result.LM) {
		t.Errorf("link map mismatch for list item edit")
	}
}

// TestIncrementalEquivalenceDeleteHeading verifies that deleting a heading
// produces the same result incrementally.
func TestIncrementalEquivalenceDeleteHeading(t *testing.T) {
	oldSource := "# Title\n\nFirst paragraph.\n\n## Section\n\nSecond paragraph.\n"
	// Delete "## Section\n" (12 runes).
	headingText := "## Section\n"
	editPos := testRuneCount("# Title\n\nFirst paragraph.\n\n")
	edit := EditRecord{Pos: editPos, OldLen: testRuneCount(headingText), NewLen: 0}
	newSource := applyEdit(oldSource, edit, "")

	fullContent, fullSM, fullLM := ParseWithSourceMap(newSource)

	result := incrementalUpdate(t, oldSource, newSource, edit)
	if result == nil {
		t.Log("full re-parse fallback (acceptable)")
		return
	}

	if !contentEqual(fullContent, result.Content) {
		t.Errorf("content mismatch after heading deletion")
	}
	if !sourceMapEqual(fullSM, result.SM) {
		t.Errorf("source map mismatch after heading deletion")
	}
	if !linkMapEqual(fullLM, result.LM) {
		t.Errorf("link map mismatch after heading deletion")
	}
}

// TestIncrementalEquivalenceAddHeading verifies that inserting a heading
// produces the same result incrementally.
func TestIncrementalEquivalenceAddHeading(t *testing.T) {
	oldSource := "First paragraph.\n\nSecond paragraph.\n"
	// Insert "## New Section\n\n" before second paragraph.
	insertText := "## New Section\n\n"
	editPos := testRuneCount("First paragraph.\n\n")
	edit := EditRecord{Pos: editPos, OldLen: 0, NewLen: testRuneCount(insertText)}
	newSource := applyEdit(oldSource, edit, insertText)

	fullContent, fullSM, fullLM := ParseWithSourceMap(newSource)

	result := incrementalUpdate(t, oldSource, newSource, edit)
	if result == nil {
		t.Log("full re-parse fallback (acceptable)")
		return
	}

	if !contentEqual(fullContent, result.Content) {
		t.Errorf("content mismatch after heading insertion")
	}
	if !sourceMapEqual(fullSM, result.SM) {
		t.Errorf("source map mismatch after heading insertion")
	}
	if !linkMapEqual(fullLM, result.LM) {
		t.Errorf("link map mismatch after heading insertion")
	}
}

// TestIncrementalEquivalenceEditInTable verifies that editing a table cell
// produces the same result incrementally.
func TestIncrementalEquivalenceEditInTable(t *testing.T) {
	oldSource := "Text.\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\nMore text.\n"
	// Replace "1" with "10" in the table cell.
	editPos := testRuneCount("Text.\n\n| A | B |\n|---|---|\n| ")
	edit := EditRecord{Pos: editPos, OldLen: 1, NewLen: 2}
	newSource := applyEdit(oldSource, edit, "10")

	fullContent, fullSM, fullLM := ParseWithSourceMap(newSource)

	result := incrementalUpdate(t, oldSource, newSource, edit)
	if result == nil {
		t.Log("full re-parse fallback for table edit (acceptable)")
		return
	}

	if !contentEqual(fullContent, result.Content) {
		t.Errorf("content mismatch for table edit")
	}
	if !sourceMapEqual(fullSM, result.SM) {
		t.Errorf("source map mismatch for table edit")
	}
	if !linkMapEqual(fullLM, result.LM) {
		t.Errorf("link map mismatch for table edit")
	}
}

// TestIncrementalEquivalenceEditWithBoldFormatting verifies that editing
// text containing bold formatting produces the same result incrementally.
func TestIncrementalEquivalenceEditWithBoldFormatting(t *testing.T) {
	oldSource := "# Title\n\nSome **bold** text here.\n\nAnother paragraph.\n"
	// Insert " very" before "bold" — after "Some **".
	insertText := " very"
	editPos := testRuneCount("# Title\n\nSome **")
	edit := EditRecord{Pos: editPos, OldLen: 0, NewLen: testRuneCount(insertText)}
	newSource := applyEdit(oldSource, edit, insertText)

	fullContent, fullSM, fullLM := ParseWithSourceMap(newSource)

	result := incrementalUpdate(t, oldSource, newSource, edit)
	if result == nil {
		t.Log("full re-parse fallback (acceptable)")
		return
	}

	if !contentEqual(fullContent, result.Content) {
		t.Errorf("content mismatch for bold formatting edit")
	}
	if !sourceMapEqual(fullSM, result.SM) {
		t.Errorf("source map mismatch for bold formatting edit")
	}
	if !linkMapEqual(fullLM, result.LM) {
		t.Errorf("link map mismatch for bold formatting edit")
	}
}

// TestIncrementalEquivalenceEditWithLink verifies that editing a
// paragraph with a link produces the same result incrementally.
func TestIncrementalEquivalenceEditWithLink(t *testing.T) {
	oldSource := "# Title\n\nClick [here](http://example.com) please.\n\nEnd.\n"
	// Insert " now" after "please".
	insertText := " now"
	editPos := testRuneCount("# Title\n\nClick [here](http://example.com) please")
	edit := EditRecord{Pos: editPos, OldLen: 0, NewLen: testRuneCount(insertText)}
	newSource := applyEdit(oldSource, edit, insertText)

	fullContent, fullSM, fullLM := ParseWithSourceMap(newSource)

	result := incrementalUpdate(t, oldSource, newSource, edit)
	if result == nil {
		t.Log("full re-parse fallback (acceptable)")
		return
	}

	if !contentEqual(fullContent, result.Content) {
		t.Errorf("content mismatch for link edit")
	}
	if !sourceMapEqual(fullSM, result.SM) {
		t.Errorf("source map mismatch for link edit")
	}
	if !linkMapEqual(fullLM, result.LM) {
		t.Errorf("link map mismatch for link edit")
	}
}

// TestIncrementalEquivalenceMergeParagraphs verifies that deleting the
// blank line between two paragraphs (merging them) produces the same
// result incrementally as full re-parse.
func TestIncrementalEquivalenceMergeParagraphs(t *testing.T) {
	oldSource := "First paragraph.\n\nSecond paragraph.\n"
	// Delete the blank line "\n" between paragraphs.
	editPos := testRuneCount("First paragraph.\n")
	edit := EditRecord{Pos: editPos, OldLen: 1, NewLen: 0}
	newSource := applyEdit(oldSource, edit, "")

	fullContent, fullSM, fullLM := ParseWithSourceMap(newSource)

	result := incrementalUpdate(t, oldSource, newSource, edit)
	if result == nil {
		t.Log("full re-parse fallback for paragraph merge (acceptable)")
		return
	}

	if !contentEqual(fullContent, result.Content) {
		t.Errorf("content mismatch after paragraph merge")
	}
	if !sourceMapEqual(fullSM, result.SM) {
		t.Errorf("source map mismatch after paragraph merge")
	}
	if !linkMapEqual(fullLM, result.LM) {
		t.Errorf("link map mismatch after paragraph merge")
	}
}

// TestIncrementalEquivalenceSplitParagraph verifies that inserting a blank
// line to split a paragraph produces the same result incrementally.
func TestIncrementalEquivalenceSplitParagraph(t *testing.T) {
	oldSource := "First line.\nSecond line.\n"
	// Insert "\n" after first line to split into two paragraphs.
	insertText := "\n"
	editPos := testRuneCount("First line.\n")
	edit := EditRecord{Pos: editPos, OldLen: 0, NewLen: testRuneCount(insertText)}
	newSource := applyEdit(oldSource, edit, insertText)

	fullContent, fullSM, fullLM := ParseWithSourceMap(newSource)

	result := incrementalUpdate(t, oldSource, newSource, edit)
	if result == nil {
		t.Log("full re-parse fallback for paragraph split (acceptable)")
		return
	}

	if !contentEqual(fullContent, result.Content) {
		t.Errorf("content mismatch after paragraph split")
	}
	if !sourceMapEqual(fullSM, result.SM) {
		t.Errorf("source map mismatch after paragraph split")
	}
	if !linkMapEqual(fullLM, result.LM) {
		t.Errorf("link map mismatch after paragraph split")
	}
}

// TestIncrementalEquivalenceNonASCII verifies incremental correctness
// when the document contains non-ASCII characters.
func TestIncrementalEquivalenceNonASCII(t *testing.T) {
	oldSource := "# Über\n\nText with émojis 🎉 here.\n\nAnother paragraph.\n"
	// Insert " more" after "émojis".
	insertText := " more"
	editPos := testRuneCount("# Über\n\nText with émojis")
	edit := EditRecord{Pos: editPos, OldLen: 0, NewLen: testRuneCount(insertText)}
	newSource := applyEdit(oldSource, edit, insertText)

	fullContent, fullSM, fullLM := ParseWithSourceMap(newSource)

	result := incrementalUpdate(t, oldSource, newSource, edit)
	if result == nil {
		t.Log("full re-parse fallback (acceptable)")
		return
	}

	if !contentEqual(fullContent, result.Content) {
		t.Errorf("content mismatch for non-ASCII edit")
	}
	if !sourceMapEqual(fullSM, result.SM) {
		t.Errorf("source map mismatch for non-ASCII edit")
	}
	if !linkMapEqual(fullLM, result.LM) {
		t.Errorf("link map mismatch for non-ASCII edit")
	}
}

// TestIncrementalEquivalenceLastBlock verifies incremental correctness
// when editing the last block of the document.
func TestIncrementalEquivalenceLastBlock(t *testing.T) {
	oldSource := "# Title\n\nOnly paragraph.\n"
	// Append " extra" to the paragraph.
	insertText := " extra"
	editPos := testRuneCount("# Title\n\nOnly paragraph")
	edit := EditRecord{Pos: editPos, OldLen: 0, NewLen: testRuneCount(insertText)}
	newSource := applyEdit(oldSource, edit, insertText)

	fullContent, fullSM, fullLM := ParseWithSourceMap(newSource)

	result := incrementalUpdate(t, oldSource, newSource, edit)
	if result == nil {
		t.Log("full re-parse fallback (acceptable)")
		return
	}

	if !contentEqual(fullContent, result.Content) {
		t.Errorf("content mismatch for last block edit")
	}
	if !sourceMapEqual(fullSM, result.SM) {
		t.Errorf("source map mismatch for last block edit")
	}
	if !linkMapEqual(fullLM, result.LM) {
		t.Errorf("link map mismatch for last block edit")
	}
}

// TestIncrementalEquivalenceFirstBlock verifies incremental correctness
// when editing the first block (heading) of the document.
func TestIncrementalEquivalenceFirstBlock(t *testing.T) {
	oldSource := "# Title\n\nParagraph.\n"
	// Replace "Title" with "New Title" in the heading.
	editPos := testRuneCount("# ")
	edit := EditRecord{Pos: editPos, OldLen: 5, NewLen: 9}
	newSource := applyEdit(oldSource, edit, "New Title")

	fullContent, fullSM, fullLM := ParseWithSourceMap(newSource)

	result := incrementalUpdate(t, oldSource, newSource, edit)
	if result == nil {
		t.Log("full re-parse fallback (acceptable)")
		return
	}

	if !contentEqual(fullContent, result.Content) {
		t.Errorf("content mismatch for first block edit")
	}
	if !sourceMapEqual(fullSM, result.SM) {
		t.Errorf("source map mismatch for first block edit")
	}
	if !linkMapEqual(fullLM, result.LM) {
		t.Errorf("link map mismatch for first block edit")
	}
}

// --- Stitch unit tests ---

// TestStitchPrefixOnly verifies that stitching with no actual change
// (replacing blocks with identical content) produces the same result.
func TestStitchPrefixOnly(t *testing.T) {
	source := "# Title\n\nParagraph.\n"
	content, sm, lm, bi := ParseWithSourceMapAndIndex(source)

	full := StitchResult{
		Content:  content,
		SM:       sm,
		LM:       lm,
		BlockIdx: bi,
	}

	// "Re-parse" the entire document (same as original).
	result := Stitch(full, full, 0, len(bi.Blocks), 0, 0)

	if !contentEqual(content, result.Content) {
		t.Errorf("content should be unchanged after identity stitch")
	}
	if !sourceMapEqual(sm, result.SM) {
		t.Errorf("source map should be unchanged after identity stitch")
	}
}

// TestStitchShiftsSuffix verifies that Stitch correctly shifts source
// map positions in the suffix blocks when the middle region changes size.
func TestStitchShiftsSuffix(t *testing.T) {
	source := "# Title\n\nMiddle paragraph.\n\nEnd paragraph.\n"
	content, sm, lm, bi := ParseWithSourceMapAndIndex(source)

	if len(bi.Blocks) < 3 {
		t.Skipf("expected at least 3 content blocks, got %d", len(bi.Blocks))
	}

	full := StitchResult{
		Content:  content,
		SM:       sm,
		LM:       lm,
		BlockIdx: bi,
	}

	// Simulate replacing middle block with a shorter version.
	// sourceDelta = -5 means the middle block shrunk by 5 runes.
	// We construct a new region by parsing "Short.\n" in place of "Middle paragraph.\n"
	newMiddle := "Short.\n"
	newMiddleLines := splitLines(newMiddle)
	middleOffset := bi.Blocks[2].SourceRuneStart // second content block (after heading, blank)

	regionContent, regionSM, regionLM := ParseRegion(newMiddleLines, middleOffset)
	newRegion := StitchResult{
		Content:  regionContent,
		SM:       regionSM,
		LM:       regionLM,
		BlockIdx: buildBlockIndex(newMiddle),
	}

	// The middle paragraph block — find it.
	paraBlocks := []int{}
	for i, b := range bi.Blocks {
		if b.Type == BlockParagraph {
			paraBlocks = append(paraBlocks, i)
		}
	}
	if len(paraBlocks) < 1 {
		t.Fatal("no paragraph blocks found")
	}

	firstPara := paraBlocks[0]
	oldMiddleLen := testRuneCount("Middle paragraph.\n")
	newMiddleLen := testRuneCount(newMiddle)
	sourceDelta := newMiddleLen - oldMiddleLen
	sourceBytesDelta := len(newMiddle) - len("Middle paragraph.\n")

	result := Stitch(full, newRegion, firstPara, firstPara+1, sourceDelta, sourceBytesDelta)

	// The result should have content from all three regions.
	if len(result.Content) == 0 {
		t.Error("Stitch produced empty content")
	}

	// Source map byte positions in suffix should be shifted by sourceBytesDelta.
	if result.SM != nil && len(result.SM.entries) > 0 {
		lastOrigEntry := sm.entries[len(sm.entries)-1]
		lastNewEntry := result.SM.entries[len(result.SM.entries)-1]

		expectedShift := sourceBytesDelta
		actualShift := lastNewEntry.SourceStart - lastOrigEntry.SourceStart
		if actualShift != expectedShift {
			t.Errorf("suffix source map byte shift: expected %d, got %d", expectedShift, actualShift)
		}
	}
}

// TestStitchBlockIndexUpdated verifies that Stitch produces a valid
// BlockIndex in the result.
func TestStitchBlockIndexUpdated(t *testing.T) {
	source := "# Title\n\nParagraph.\n"
	content, sm, lm, bi := ParseWithSourceMapAndIndex(source)

	full := StitchResult{
		Content:  content,
		SM:       sm,
		LM:       lm,
		BlockIdx: bi,
	}

	result := Stitch(full, full, 0, len(bi.Blocks), 0, 0)

	if result.BlockIdx == nil {
		t.Fatal("Stitch returned nil BlockIndex")
	}

	if len(result.BlockIdx.Blocks) == 0 {
		t.Error("Stitch returned empty BlockIndex")
	}
}

// TestParseRegionEmptyLines verifies ParseRegion with empty input.
func TestParseRegionEmptyLines(t *testing.T) {
	content, sm, lm := ParseRegion(nil, 0)
	// Should return empty content without panicking.
	_ = content
	_ = sm
	_ = lm
}

// TestParseRegionOffsetZero verifies that ParseRegion with offset 0
// produces the same output as ParseWithSourceMap for the same text.
func TestParseRegionOffsetZero(t *testing.T) {
	source := "Hello **bold** world.\n"
	lines := splitLines(source)

	fullContent, fullSM, fullLM := ParseWithSourceMap(source)
	regionContent, regionSM, regionLM := ParseRegion(lines, 0)

	if !contentEqual(fullContent, regionContent) {
		t.Errorf("ParseRegion(offset=0) content differs from ParseWithSourceMap:\n  full: %v\n  region: %v",
			fullContent, regionContent)
	}

	if !sourceMapEqual(fullSM, regionSM) {
		t.Errorf("ParseRegion(offset=0) source map differs from ParseWithSourceMap")
	}

	if !linkMapEqual(fullLM, regionLM) {
		t.Errorf("ParseRegion(offset=0) link map differs from ParseWithSourceMap")
	}
}
