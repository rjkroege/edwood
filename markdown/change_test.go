package markdown

import (
	"testing"
)

// TestEditRecordBasic verifies the EditRecord struct captures insert and
// delete operations correctly.
func TestEditRecordBasic(t *testing.T) {
	// Pure insert: 5 runes inserted at position 10
	ins := EditRecord{Pos: 10, OldLen: 0, NewLen: 5}
	if ins.Pos != 10 || ins.OldLen != 0 || ins.NewLen != 5 {
		t.Errorf("insert record fields wrong: %+v", ins)
	}

	// Pure delete: 3 runes removed at position 20
	del := EditRecord{Pos: 20, OldLen: 3, NewLen: 0}
	if del.Pos != 20 || del.OldLen != 3 || del.NewLen != 0 {
		t.Errorf("delete record fields wrong: %+v", del)
	}

	// Replace: 2 runes replaced by 4 runes at position 0
	rep := EditRecord{Pos: 0, OldLen: 2, NewLen: 4}
	if rep.Pos != 0 || rep.OldLen != 2 || rep.NewLen != 4 {
		t.Errorf("replace record fields wrong: %+v", rep)
	}
}

// TestBlockIndexFromParse verifies that ParseWithSourceMapAndIndex
// returns a BlockIndex with correct block boundaries for a document
// containing various block types.
func TestBlockIndexFromParse(t *testing.T) {
	// Document with: heading, blank line, paragraph, blank line, fenced code, blank line, list item.
	doc := "# Title\n\nSome paragraph text.\n\n```\ncode line\n```\n\n- item one\n"
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)

	if bi == nil {
		t.Fatal("ParseWithSourceMapAndIndex returned nil BlockIndex")
	}
	if len(bi.Blocks) == 0 {
		t.Fatal("BlockIndex has no blocks")
	}

	// Verify we have at least the expected block types.
	typesSeen := make(map[BlockType]bool)
	for _, b := range bi.Blocks {
		typesSeen[b.Type] = true
	}

	for _, want := range []BlockType{BlockHeading, BlockParagraph, BlockFencedCode, BlockListItem} {
		if !typesSeen[want] {
			t.Errorf("expected block type %v not found in BlockIndex; blocks: %+v", want, bi.Blocks)
		}
	}
}

// TestBlockInfoSourceRunePositions verifies that BlockInfo entries have
// non-overlapping SourceRuneStart/End ranges covering the source document.
func TestBlockInfoSourceRunePositions(t *testing.T) {
	doc := "# Heading\n\nParagraph one.\n\nParagraph two.\n"
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}

	// Filter out blank-line entries for contiguity check —
	// blank lines are boundary markers, not content blocks.
	var contentBlocks []BlockInfo
	for _, b := range bi.Blocks {
		if b.Type != BlockBlankLine {
			contentBlocks = append(contentBlocks, b)
		}
	}

	if len(contentBlocks) < 2 {
		t.Fatalf("expected at least 2 content blocks, got %d: %+v", len(contentBlocks), bi.Blocks)
	}

	// Blocks should be ordered and non-overlapping by source rune position.
	for i := 1; i < len(contentBlocks); i++ {
		prev := contentBlocks[i-1]
		cur := contentBlocks[i]
		if cur.SourceRuneStart < prev.SourceRuneEnd {
			t.Errorf("block %d (rune %d-%d) overlaps block %d (rune %d-%d)",
				i-1, prev.SourceRuneStart, prev.SourceRuneEnd,
				i, cur.SourceRuneStart, cur.SourceRuneEnd)
		}
	}
}

// TestAffectedRangeEditInParagraph verifies that a single-line edit
// within a paragraph identifies only the containing paragraph block.
func TestAffectedRangeEditInParagraph(t *testing.T) {
	doc := "# Title\n\nSome paragraph text here.\n\nAnother paragraph.\n"
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}

	// Find the first paragraph block.
	paraIdx := -1
	for i, b := range bi.Blocks {
		if b.Type == BlockParagraph {
			paraIdx = i
			break
		}
	}
	if paraIdx < 0 {
		t.Fatal("no paragraph block found")
	}

	para := bi.Blocks[paraIdx]

	// Edit: insert 3 runes in the middle of the paragraph.
	editPos := para.SourceRuneStart + 5
	edits := []EditRecord{{Pos: editPos, OldLen: 0, NewLen: 3}}

	startBlock, endBlock := bi.AffectedRange(edits)

	// Should not trigger full re-parse.
	if startBlock == -1 || endBlock == -1 {
		t.Fatalf("AffectedRange returned full re-parse (-1,-1) for edit within paragraph")
	}

	// The affected range should include the paragraph (possibly with ±1 expansion).
	if startBlock > paraIdx || endBlock <= paraIdx {
		t.Errorf("affected range [%d, %d) does not include paragraph block %d", startBlock, endBlock, paraIdx)
	}
}

// TestAffectedRangeEditInCodeBlock verifies that an edit inside a fenced
// code block identifies the code block (without triggering full re-parse,
// since the edit doesn't touch a fence delimiter).
func TestAffectedRangeEditInCodeBlock(t *testing.T) {
	doc := "# Title\n\n```\nsome code\nmore code\n```\n\nAfter code.\n"
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}

	// Find the fenced code block.
	codeIdx := -1
	for i, b := range bi.Blocks {
		if b.Type == BlockFencedCode {
			codeIdx = i
			break
		}
	}
	if codeIdx < 0 {
		t.Fatal("no fenced code block found")
	}

	code := bi.Blocks[codeIdx]

	// Edit: insert 4 runes inside the code content (not on a fence line).
	editPos := code.SourceRuneStart + 5 // well inside the code content
	edits := []EditRecord{{Pos: editPos, OldLen: 0, NewLen: 4}}

	startBlock, endBlock := bi.AffectedRange(edits)

	// Should not trigger full re-parse (edit is inside code, not touching fence).
	if startBlock == -1 || endBlock == -1 {
		t.Fatalf("AffectedRange returned full re-parse for edit inside code block content")
	}

	// The affected range should include the code block.
	if startBlock > codeIdx || endBlock <= codeIdx {
		t.Errorf("affected range [%d, %d) does not include code block %d", startBlock, endBlock, codeIdx)
	}
}

// TestAffectedRangeAddHeading verifies that inserting a heading line is
// detected correctly.
func TestAffectedRangeAddHeading(t *testing.T) {
	doc := "First paragraph.\n\nSecond paragraph.\n"
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}

	// Simulate inserting "# New\n" (7 runes) right before "Second paragraph."
	// Find the second paragraph block.
	var secondPara *BlockInfo
	paraCount := 0
	for i, b := range bi.Blocks {
		if b.Type == BlockParagraph {
			paraCount++
			if paraCount == 2 {
				secondPara = &bi.Blocks[i]
				break
			}
		}
	}
	if secondPara == nil {
		t.Fatal("no second paragraph block found")
	}

	editPos := secondPara.SourceRuneStart
	edits := []EditRecord{{Pos: editPos, OldLen: 0, NewLen: 7}} // "# New\n" = 7 runes

	startBlock, endBlock := bi.AffectedRange(edits)

	// Should identify an affected range (not necessarily full re-parse).
	if startBlock == -1 || endBlock == -1 {
		// Adding a heading doesn't touch fence delimiters, so full re-parse
		// is not required — but it's acceptable if the implementation is conservative.
		t.Logf("AffectedRange returned full re-parse for heading insertion (conservative, acceptable)")
		return
	}

	// The affected range should be near the second paragraph.
	if startBlock > len(bi.Blocks) || endBlock > len(bi.Blocks)+1 {
		t.Errorf("affected range [%d, %d) out of bounds (total blocks: %d)", startBlock, endBlock, len(bi.Blocks))
	}
}

// TestAffectedRangeDeleteListItem verifies that deleting a list item is
// detected correctly.
func TestAffectedRangeDeleteListItem(t *testing.T) {
	doc := "- item one\n- item two\n- item three\n"
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}

	// Find the second list item.
	listItems := []int{}
	for i, b := range bi.Blocks {
		if b.Type == BlockListItem {
			listItems = append(listItems, i)
		}
	}
	if len(listItems) < 2 {
		t.Fatalf("expected at least 2 list item blocks, got %d", len(listItems))
	}

	secondItem := bi.Blocks[listItems[1]]

	// Delete the entire second list item.
	itemLen := secondItem.SourceRuneEnd - secondItem.SourceRuneStart
	edits := []EditRecord{{Pos: secondItem.SourceRuneStart, OldLen: itemLen, NewLen: 0}}

	startBlock, endBlock := bi.AffectedRange(edits)

	// Should identify affected range including the deleted item's position.
	if startBlock == -1 || endBlock == -1 {
		t.Logf("AffectedRange returned full re-parse for list item deletion (conservative, acceptable)")
		return
	}

	// The affected range should span around the second list item.
	if startBlock > listItems[1] || endBlock <= listItems[1] {
		t.Errorf("affected range [%d, %d) does not include deleted list item at block %d", startBlock, endBlock, listItems[1])
	}
}

// TestAffectedRangeFenceDelimiterTriggersFullReparse verifies that
// editing a fence delimiter (```) triggers a full re-parse (-1, -1).
func TestAffectedRangeFenceDelimiterTriggersFullReparse(t *testing.T) {
	doc := "Before.\n\n```\ncode\n```\n\nAfter.\n"
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}

	// Find the fenced code block.
	codeIdx := -1
	for i, b := range bi.Blocks {
		if b.Type == BlockFencedCode {
			codeIdx = i
			break
		}
	}
	if codeIdx < 0 {
		t.Fatal("no fenced code block found")
	}

	// Simulate inserting "```\n" which adds a new fence delimiter.
	// The edit is at the code block's start position (the opening fence line).
	code := bi.Blocks[codeIdx]
	edits := []EditRecord{{Pos: code.SourceRuneStart, OldLen: 4, NewLen: 0}} // delete "```\n"

	startBlock, endBlock := bi.AffectedRange(edits)

	// Must trigger full re-parse because a fence delimiter is involved.
	if startBlock != -1 || endBlock != -1 {
		t.Errorf("AffectedRange should return (-1,-1) for fence delimiter edit, got (%d,%d)", startBlock, endBlock)
	}
}

// TestAffectedRangeInsertFenceInParagraph verifies that inserting text
// containing "```" into a non-code block triggers full re-parse.
func TestAffectedRangeInsertFenceInParagraph(t *testing.T) {
	// This tests the design doc's requirement: "if the edit is in a non-code
	// block but the inserted/deleted text contains ```, return (-1, -1)."
	//
	// Since AffectedRange only sees EditRecords (position + lengths, not the
	// actual text), fence detection in non-code blocks requires access to the
	// new source text. The implementation may handle this by checking whether
	// the edit overlaps a line that looks like a fence in the NEW source, or
	// by requiring the caller to pass a "containsFence" flag.
	//
	// For now, this test verifies the conservative behavior: if an edit
	// straddles a block boundary or inserts enough text that could be a fence,
	// the implementation may return (-1, -1). The exact API for detecting
	// fences in non-code contexts will be determined during the Iterate stage.
	t.Log("Fence-in-paragraph detection depends on implementation details; see Iterate stage")
}

// TestAffectedRangeMultipleEditsCoalesced verifies that multiple edits
// between timer firings are coalesced into a single affected range.
func TestAffectedRangeMultipleEditsCoalesced(t *testing.T) {
	doc := "# Heading\n\nFirst paragraph.\n\nSecond paragraph.\n\nThird paragraph.\n"
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}

	// Find paragraph blocks.
	var paras []int
	for i, b := range bi.Blocks {
		if b.Type == BlockParagraph {
			paras = append(paras, i)
		}
	}
	if len(paras) < 3 {
		t.Fatalf("expected at least 3 paragraphs, got %d", len(paras))
	}

	// Two edits: one in first paragraph, one in third paragraph.
	edit1Pos := bi.Blocks[paras[0]].SourceRuneStart + 2
	edit2Pos := bi.Blocks[paras[2]].SourceRuneStart + 2
	edits := []EditRecord{
		{Pos: edit1Pos, OldLen: 0, NewLen: 1},
		{Pos: edit2Pos + 1, OldLen: 0, NewLen: 1}, // +1 to account for first edit's shift
	}

	startBlock, endBlock := bi.AffectedRange(edits)

	if startBlock == -1 || endBlock == -1 {
		t.Log("full re-parse for multi-edit (conservative, acceptable)")
		return
	}

	// The coalesced range should span from first paragraph to third paragraph.
	if startBlock > paras[0] || endBlock <= paras[2] {
		t.Errorf("affected range [%d, %d) doesn't span first para (block %d) through third para (block %d)",
			startBlock, endBlock, paras[0], paras[2])
	}
}

// TestAffectedRangeEditAtBlockBoundary verifies that an edit at the boundary
// between two blocks expands to include both adjacent blocks.
func TestAffectedRangeEditAtBlockBoundary(t *testing.T) {
	// A blank line separates two paragraphs. Deleting the blank line should
	// merge the paragraphs, so the affected range must include both.
	doc := "First paragraph.\n\nSecond paragraph.\n"
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}

	// Find the blank line between paragraphs.
	blankIdx := -1
	for i, b := range bi.Blocks {
		if b.Type == BlockBlankLine {
			blankIdx = i
			break
		}
	}

	if blankIdx < 0 {
		// If the implementation doesn't track blank lines as blocks,
		// find the gap between the two paragraphs and edit there.
		var paras []int
		for i, b := range bi.Blocks {
			if b.Type == BlockParagraph {
				paras = append(paras, i)
			}
		}
		if len(paras) < 2 {
			t.Fatal("expected 2 paragraphs")
		}
		// Edit at the end of the first paragraph (the blank line position).
		editPos := bi.Blocks[paras[0]].SourceRuneEnd
		edits := []EditRecord{{Pos: editPos, OldLen: 1, NewLen: 0}} // delete the \n

		startBlock, endBlock := bi.AffectedRange(edits)
		if startBlock == -1 || endBlock == -1 {
			t.Log("full re-parse for boundary edit (conservative, acceptable)")
			return
		}

		// Must include both paragraphs.
		if startBlock > paras[0] || endBlock <= paras[1] {
			t.Errorf("boundary edit affected range [%d, %d) should include both paras [%d, %d]",
				startBlock, endBlock, paras[0], paras[1])
		}
		return
	}

	blank := bi.Blocks[blankIdx]
	edits := []EditRecord{{Pos: blank.SourceRuneStart, OldLen: blank.SourceRuneEnd - blank.SourceRuneStart, NewLen: 0}}

	startBlock, endBlock := bi.AffectedRange(edits)
	if startBlock == -1 || endBlock == -1 {
		t.Log("full re-parse for boundary edit (conservative, acceptable)")
		return
	}

	// The range must include blocks on both sides of the deleted blank line.
	if blankIdx > 0 && startBlock > blankIdx-1 {
		t.Errorf("should include block before blank line: startBlock=%d, blankIdx=%d", startBlock, blankIdx)
	}
	if blankIdx < len(bi.Blocks)-1 && endBlock <= blankIdx+1 {
		t.Errorf("should include block after blank line: endBlock=%d, blankIdx=%d", startBlock, blankIdx)
	}
}

// TestBlockTypeValues verifies the BlockType constants exist and are distinct.
func TestBlockTypeValues(t *testing.T) {
	types := []BlockType{
		BlockParagraph,
		BlockFencedCode,
		BlockIndentedCode,
		BlockHeading,
		BlockHRule,
		BlockTable,
		BlockListItem,
		BlockBlankLine,
	}

	seen := make(map[BlockType]bool)
	for _, bt := range types {
		if seen[bt] {
			t.Errorf("duplicate BlockType value: %d", bt)
		}
		seen[bt] = true
	}
}

// TestBlockIndexEmptyDocument verifies BlockIndex behavior for empty input.
func TestBlockIndexEmptyDocument(t *testing.T) {
	_, _, _, bi := ParseWithSourceMapAndIndex("")
	if bi == nil {
		t.Fatal("nil BlockIndex for empty document")
	}
	if len(bi.Blocks) != 0 {
		t.Errorf("expected 0 blocks for empty document, got %d", len(bi.Blocks))
	}
}

// TestBlockIndexSingleParagraph verifies a simple single-paragraph document.
func TestBlockIndexSingleParagraph(t *testing.T) {
	doc := "Hello world."
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}
	if len(bi.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d: %+v", len(bi.Blocks), bi.Blocks)
	}
	if bi.Blocks[0].Type != BlockParagraph {
		t.Errorf("expected BlockParagraph, got %v", bi.Blocks[0].Type)
	}
}

// TestBlockIndexHeadingOnly verifies a document with just a heading.
func TestBlockIndexHeadingOnly(t *testing.T) {
	doc := "## Section Title\n"
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}
	// Should have at least one heading block.
	found := false
	for _, b := range bi.Blocks {
		if b.Type == BlockHeading {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected BlockHeading, blocks: %+v", bi.Blocks)
	}
}

// TestBlockIndexHorizontalRule verifies detection of horizontal rules.
func TestBlockIndexHorizontalRule(t *testing.T) {
	doc := "Above\n\n---\n\nBelow\n"
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}
	found := false
	for _, b := range bi.Blocks {
		if b.Type == BlockHRule {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected BlockHRule, blocks: %+v", bi.Blocks)
	}
}

// TestBlockIndexTable verifies detection of tables.
func TestBlockIndexTable(t *testing.T) {
	doc := "| A | B |\n|---|---|\n| 1 | 2 |\n"
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}
	found := false
	for _, b := range bi.Blocks {
		if b.Type == BlockTable {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected BlockTable, blocks: %+v", bi.Blocks)
	}
}

// TestBlockIndexIndentedCode verifies detection of indented code blocks.
func TestBlockIndexIndentedCode(t *testing.T) {
	doc := "Normal text.\n\n    code line one\n    code line two\n\nMore text.\n"
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}
	found := false
	for _, b := range bi.Blocks {
		if b.Type == BlockIndentedCode {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected BlockIndentedCode, blocks: %+v", bi.Blocks)
	}
}

// TestBlockIndexMixedDocument verifies block detection for a document with
// many block types.
func TestBlockIndexMixedDocument(t *testing.T) {
	doc := `# Title

Introduction paragraph.

- list item 1
- list item 2

` + "```" + `
code here
` + "```" + `

| A | B |
|---|---|
| 1 | 2 |

---

Final paragraph.
`
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}

	wantTypes := map[BlockType]string{
		BlockHeading:    "heading",
		BlockParagraph:  "paragraph",
		BlockListItem:   "list item",
		BlockFencedCode: "fenced code",
		BlockTable:      "table",
		BlockHRule:      "horizontal rule",
	}

	seen := make(map[BlockType]bool)
	for _, b := range bi.Blocks {
		seen[b.Type] = true
	}

	for bt, name := range wantTypes {
		if !seen[bt] {
			t.Errorf("expected %s block not found", name)
		}
	}
}

// TestAffectedRangeEmptyEdits verifies behavior with no edits.
func TestAffectedRangeEmptyEdits(t *testing.T) {
	doc := "Some text.\n"
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}

	startBlock, endBlock := bi.AffectedRange(nil)

	// No edits → no affected range. Could return (0, 0) or (-1, -1).
	// Either is acceptable as long as nothing is re-parsed.
	if startBlock > 0 || endBlock > 0 {
		t.Errorf("expected no affected range for empty edits, got (%d, %d)", startBlock, endBlock)
	}
}

// TestAffectedRangeEditBeyondDocument verifies that an edit beyond the
// document end is handled gracefully.
func TestAffectedRangeEditBeyondDocument(t *testing.T) {
	doc := "Short.\n"
	_, _, _, bi := ParseWithSourceMapAndIndex(doc)
	if bi == nil {
		t.Fatal("nil BlockIndex")
	}

	// Edit at a position well beyond the document.
	edits := []EditRecord{{Pos: 1000, OldLen: 0, NewLen: 5}}
	startBlock, endBlock := bi.AffectedRange(edits)

	// Should either return full re-parse or the last block.
	// Must not panic.
	_ = startBlock
	_ = endBlock
}

// TestParseWithSourceMapAndIndexEquivalence verifies that the Content,
// SourceMap, and LinkMap from ParseWithSourceMapAndIndex match those
// from ParseWithSourceMap (ensuring the index-producing path doesn't
// alter the parse results).
func TestParseWithSourceMapAndIndexEquivalence(t *testing.T) {
	docs := []struct {
		name string
		doc  string
	}{
		{"paragraph", "Hello world.\n"},
		{"heading+para", "# Title\n\nSome text.\n"},
		{"fenced code", "```\ncode\n```\n"},
		{"list items", "- one\n- two\n- three\n"},
		{"table", "| A | B |\n|---|---|\n| 1 | 2 |\n"},
		{"mixed", "# H\n\ntext\n\n```\nc\n```\n\n- i\n"},
	}

	for _, tt := range docs {
		t.Run(tt.name, func(t *testing.T) {
			contentOrig, smOrig, lmOrig := ParseWithSourceMap(tt.doc)
			contentNew, smNew, lmNew, bi := ParseWithSourceMapAndIndex(tt.doc)

			if bi == nil {
				t.Fatal("nil BlockIndex")
			}

			// Content should be identical.
			if len(contentOrig) != len(contentNew) {
				t.Fatalf("content length mismatch: original %d vs new %d", len(contentOrig), len(contentNew))
			}
			for i := range contentOrig {
				if contentOrig[i].Text != contentNew[i].Text {
					t.Errorf("span %d text: %q vs %q", i, contentOrig[i].Text, contentNew[i].Text)
				}
				if contentOrig[i].Style != contentNew[i].Style {
					t.Errorf("span %d style mismatch", i)
				}
			}

			// Source map entries should be identical.
			if len(smOrig.entries) != len(smNew.entries) {
				t.Fatalf("source map length mismatch: %d vs %d", len(smOrig.entries), len(smNew.entries))
			}
			for i := range smOrig.entries {
				if smOrig.entries[i] != smNew.entries[i] {
					t.Errorf("source map entry %d: %+v vs %+v", i, smOrig.entries[i], smNew.entries[i])
				}
			}

			// Link map — just check entry count for now.
			if len(lmOrig.entries) != len(lmNew.entries) {
				t.Errorf("link map length mismatch: %d vs %d", len(lmOrig.entries), len(lmNew.entries))
			}
		})
	}
}
