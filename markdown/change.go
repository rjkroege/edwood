package markdown

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/rjkroege/edwood/rich"
)

// EditRecord describes a single edit operation on the source buffer.
type EditRecord struct {
	Pos    int // rune position in source where edit occurred
	OldLen int // runes removed (0 for pure insert)
	NewLen int // runes inserted (0 for pure delete)
}

// BlockType identifies the kind of markdown block.
type BlockType int

const (
	BlockParagraph    BlockType = iota
	BlockFencedCode
	BlockIndentedCode
	BlockHeading
	BlockHRule
	BlockTable
	BlockListItem
	BlockBlankLine
)

// BlockInfo records the source extent of a parsed block.
type BlockInfo struct {
	SourceLineStart int       // first line index (0-based) in splitLines output
	SourceLineEnd   int       // last line index (exclusive)
	SourceRuneStart int       // first rune position in source
	SourceRuneEnd   int       // last rune position in source (exclusive)
	ContentStart    int       // index into Content slice where this block's spans begin
	ContentEnd      int       // index into Content slice where this block's spans end (exclusive)
	SMStart         int       // index into SourceMap.entries for this block
	SMEnd           int       // index into SourceMap.entries for this block (exclusive)
	LMStart         int       // index into LinkMap.entries for this block
	LMEnd           int       // index into LinkMap.entries for this block (exclusive)
	Type            BlockType
}

// BlockIndex maps source positions to block extents.
type BlockIndex struct {
	Blocks         []BlockInfo
	SourceByteLen  int   // byte length of the source text that produced this index
	lineRuneStarts []int // rune offset for each line start (internal, for fence checking)
}

// AffectedRange returns the range of blocks that must be re-parsed
// given the edits. Returns (startBlock, endBlock) indices into
// BlockIndex.Blocks, or (-1, -1) if a full re-parse is needed.
func (bi *BlockIndex) AffectedRange(edits []EditRecord) (int, int) {
	if len(edits) == 0 || len(bi.Blocks) == 0 {
		return 0, 0
	}

	// Coalesce edits into a single source rune range [editStart, editEnd).
	// We use positions from the OLD source (before edits).
	editStart := edits[0].Pos
	editEnd := edits[0].Pos + edits[0].OldLen
	if editEnd < editStart {
		editEnd = editStart
	}

	for _, e := range edits[1:] {
		if e.Pos < editStart {
			editStart = e.Pos
		}
		end := e.Pos + e.OldLen
		if end < e.Pos {
			end = e.Pos
		}
		if end > editEnd {
			editEnd = end
		}
	}

	// For pure inserts (OldLen == 0), editEnd == editStart.
	// We still need to find which block the insert falls in.
	// Expand editEnd by 1 to ensure we match the block containing editStart.
	if editEnd == editStart {
		editEnd = editStart + 1
	}

	// Binary search to find the first block whose SourceRuneEnd > editStart.
	startBlock := sort.Search(len(bi.Blocks), func(i int) bool {
		return bi.Blocks[i].SourceRuneEnd > editStart
	})

	// Binary search to find the first block whose SourceRuneStart >= editEnd.
	endBlock := sort.Search(len(bi.Blocks), func(i int) bool {
		return bi.Blocks[i].SourceRuneStart >= editEnd
	})

	// Clamp startBlock.
	if startBlock >= len(bi.Blocks) {
		startBlock = len(bi.Blocks) - 1
	}

	// endBlock is exclusive, so include the block at endBlock if it overlaps.
	// The search finds the first block that starts AT or AFTER editEnd,
	// so endBlock is already exclusive. But we want to include any block
	// that overlaps the edit range.
	if endBlock < len(bi.Blocks) && bi.Blocks[endBlock].SourceRuneStart < editEnd {
		endBlock++
	}

	// Ensure endBlock > startBlock (at minimum one block).
	if endBlock <= startBlock {
		endBlock = startBlock + 1
	}

	// Expand by one block in each direction to handle boundary effects.
	if startBlock > 0 {
		startBlock--
	}
	if endBlock < len(bi.Blocks) {
		endBlock++
	}

	// Fence check: if any affected block is a fenced code block,
	// check whether the edit touches a fence delimiter line.
	// If the edit is entirely within the code content (not on the
	// opening or closing fence line), incremental re-parse is safe.
	for i := startBlock; i < endBlock && i < len(bi.Blocks); i++ {
		b := &bi.Blocks[i]
		if b.Type != BlockFencedCode {
			continue
		}
		// Determine rune ranges of the opening and closing fence lines.
		// Opening fence = first line of the block.
		// Closing fence = last line of the block (if block spans > 1 line).
		openStart := b.SourceRuneStart
		openEnd := openStart
		if b.SourceLineStart < len(bi.lineRuneStarts)-1 {
			openEnd = bi.lineRuneStarts[b.SourceLineStart+1]
		}
		closeStart := b.SourceRuneEnd
		closeEnd := b.SourceRuneEnd
		if b.SourceLineEnd > b.SourceLineStart+1 {
			closeStart = bi.lineRuneStarts[b.SourceLineEnd-1]
			closeEnd = b.SourceRuneEnd
		}
		// If edit overlaps opening fence or closing fence, full re-parse.
		if editStart < openEnd && editEnd > openStart {
			return -1, -1
		}
		if editStart < closeEnd && editEnd > closeStart {
			return -1, -1
		}
	}

	return startBlock, endBlock
}

// ParseWithSourceMapAndIndex parses markdown and returns the styled content,
// a source map, a link map, and a block index recording block boundaries.
// The content, source map, and link map are identical to ParseWithSourceMap;
// the block index is built by a separate lightweight line scan.
func ParseWithSourceMapAndIndex(text string) (rich.Content, *SourceMap, *LinkMap, *BlockIndex) {
	content, sm, lm := ParseWithSourceMap(text)
	bi := buildBlockIndex(text)
	return content, sm, lm, bi
}

// buildBlockIndex scans the source text to identify block boundaries.
// It mirrors the block-detection logic of ParseWithSourceMap but only
// records block extents — it does not parse inline content.
func buildBlockIndex(text string) *BlockIndex {
	if text == "" {
		return &BlockIndex{}
	}

	lines := splitLines(text)

	// Compute rune offset for each line start.
	lineRuneStarts := make([]int, len(lines)+1)
	runePos := 0
	for i, line := range lines {
		lineRuneStarts[i] = runePos
		runePos += utf8.RuneCountInString(line)
	}
	lineRuneStarts[len(lines)] = runePos

	bi := &BlockIndex{SourceByteLen: len(text), lineRuneStarts: lineRuneStarts}

	inFencedBlock := false
	fencedBlockStart := 0

	inIndentedBlock := false
	indentedBlockStart := 0

	inParagraph := false
	paragraphStart := 0

	emitBlock := func(typ BlockType, startLine, endLine int) {
		bi.Blocks = append(bi.Blocks, BlockInfo{
			SourceLineStart: startLine,
			SourceLineEnd:   endLine,
			SourceRuneStart: lineRuneStarts[startLine],
			SourceRuneEnd:   lineRuneStarts[endLine],
			Type:            typ,
		})
	}

	emitIndentedBlock := func(currentLine int) {
		if inIndentedBlock {
			emitBlock(BlockIndentedCode, indentedBlockStart, currentLine)
			inIndentedBlock = false
		}
	}

	emitParagraph := func(currentLine int) {
		if inParagraph {
			emitBlock(BlockParagraph, paragraphStart, currentLine)
			inParagraph = false
		}
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check for fenced code block delimiter.
		if isFenceDelimiter(line) {
			if inIndentedBlock {
				emitIndentedBlock(i)
			}
			if inParagraph {
				emitParagraph(i)
			}
			if !inFencedBlock {
				// Opening fence.
				inFencedBlock = true
				fencedBlockStart = i
				continue
			} else {
				// Closing fence — emit fenced code block including both fences.
				inFencedBlock = false
				emitBlock(BlockFencedCode, fencedBlockStart, i+1)
				continue
			}
		}

		if inFencedBlock {
			// Inside fenced block — skip, will be emitted at closing fence.
			continue
		}

		// Check for list items BEFORE indented code (mirrors parser priority).
		isULEarly, _, _ := isUnorderedListItem(line)
		isOLEarly, _, _, _ := isOrderedListItem(line)
		isListItemEarly := isULEarly || isOLEarly

		// Check for indented code block.
		if isIndentedCodeLine(line) && !isListItemEarly {
			if inParagraph {
				emitParagraph(i)
			}
			if !inIndentedBlock {
				inIndentedBlock = true
				indentedBlockStart = i
			}
			continue
		}

		// Not an indented line — emit pending indented block.
		if inIndentedBlock {
			emitIndentedBlock(i)
		}

		// Check for blank line.
		trimmedLine := strings.TrimRight(line, "\n")
		if trimmedLine == "" {
			if inParagraph {
				emitParagraph(i)
			}
			emitBlock(BlockBlankLine, i, i+1)
			continue
		}

		// Check for table.
		isRow, _ := isTableRow(line)
		if isRow && i+1 < len(lines) && isTableSeparatorRow(lines[i+1]) {
			if inParagraph {
				emitParagraph(i)
			}
			// Count consecutive table rows.
			tableStart := i
			consumed := 0
			for j := i; j < len(lines); j++ {
				isTableLine, _ := isTableRow(lines[j])
				isSep := isTableSeparatorRow(lines[j])
				if isTableLine || isSep {
					consumed++
				} else {
					break
				}
			}
			emitBlock(BlockTable, tableStart, tableStart+consumed)
			i += consumed - 1 // -1 because loop increments
			continue
		}

		// Check for block-level elements.
		isUL, _, _ := isUnorderedListItem(line)
		isOL, _, _, _ := isOrderedListItem(line)
		isListItem := isUL || isOL
		level := headingLevel(line)
		isHRule := isHorizontalRule(line)

		if level > 0 {
			if inParagraph {
				emitParagraph(i)
			}
			emitBlock(BlockHeading, i, i+1)
			continue
		}

		if isHRule {
			if inParagraph {
				emitParagraph(i)
			}
			emitBlock(BlockHRule, i, i+1)
			continue
		}

		if isListItem {
			if inParagraph {
				emitParagraph(i)
			}
			// Determine contentCol for list continuation detection
			var contentCol int
			if isUL {
				_, _, cs := isUnorderedListItem(line)
				contentCol = cs
			} else {
				_, _, cs, _ := isOrderedListItem(line)
				contentCol = cs
			}
			_, indentLvl, _ := isUnorderedListItem(line)
			if isOL {
				_, indentLvl, _, _ = isOrderedListItem(line)
			}

			// Scan ahead for continuation lines (multi-line list items)
			listItemEnd := i + 1
			inListFenced := false
			for j := i + 1; j < len(lines); j++ {
				contLine := lines[j]

				if inListFenced {
					stripped, ok := stripListIndent(contLine, contentCol)
					if ok && isFenceDelimiter(stripped) {
						inListFenced = false
						listItemEnd = j + 1
						continue
					}
					// Inside list fenced block — accumulate
					listItemEnd = j + 1
					continue
				}

				// Check if this is another list item at same or lower indent
				isULC, ci, _ := isUnorderedListItem(contLine)
				isOLC, oi, _, _ := isOrderedListItem(contLine)
				if isULC && ci <= indentLvl {
					break
				}
				if isOLC && oi <= indentLvl {
					break
				}

				stripped, ok := stripListIndent(contLine, contentCol)
				if ok {
					if isFenceDelimiter(stripped) {
						inListFenced = true
						listItemEnd = j + 1
						continue
					}
					// Other continuation (indented code, text, etc.)
					listItemEnd = j + 1
					continue
				}

				// Blank line — end list item
				trimmedCont := strings.TrimRight(contLine, "\n")
				if trimmedCont == "" {
					break
				}
				// Not indented enough — end list item
				break
			}

			emitBlock(BlockListItem, i, listItemEnd)
			i = listItemEnd - 1 // -1 because loop increments
			continue
		}

		// Regular paragraph text.
		if !inParagraph {
			inParagraph = true
			paragraphStart = i
		}
		// Continue collecting paragraph lines.
	}

	// Handle unclosed blocks at end of input.
	if inFencedBlock {
		emitBlock(BlockFencedCode, fencedBlockStart, len(lines))
	}
	if inIndentedBlock {
		emitIndentedBlock(len(lines))
	}
	if inParagraph {
		emitParagraph(len(lines))
	}

	return bi
}

// StitchResult holds the merged output of an incremental update.
type StitchResult struct {
	Content  rich.Content
	SM       *SourceMap
	LM       *LinkMap
	BlockIdx *BlockIndex
}

// ParseRegion parses a contiguous range of source lines and returns
// the content, source map, and link map for that region.
// sourceRuneOffset is the rune position of the first line in the full source.
// sourceByteOffset is the byte position of the first line in the full source.
// Both offsets are applied to source map entries so they reference positions
// in the full document.
// Rendered positions in the returned source map and link map are relative
// to the region (starting from 0); the caller (Stitch) handles global
// rendered positioning.
func ParseRegion(lines []string, sourceRuneOffset int, sourceByteOffset ...int) (rich.Content, *SourceMap, *LinkMap) {
	if len(lines) == 0 {
		return nil, &SourceMap{}, NewLinkMap()
	}

	text := strings.Join(lines, "")
	content, sm, lm := ParseWithSourceMap(text)

	// Compute the byte offset. If provided explicitly, use it.
	// Otherwise, compute from the region text: for all-ASCII text,
	// rune offset == byte offset. For non-ASCII, the caller should
	// provide the byte offset explicitly.
	byteOff := sourceRuneOffset // default: assume ASCII
	if len(sourceByteOffset) > 0 {
		byteOff = sourceByteOffset[0]
	}

	// Shift source byte positions to be absolute in the full document.
	// Rune positions (SourceRuneStart/End) are NOT shifted here because
	// populateRunePositions produces incorrect values for mid-character
	// byte offsets. After stitching, the caller must call
	// sm.PopulateRunePositions(fullSource) to derive rune positions from
	// the globally-correct byte positions.
	for i := range sm.entries {
		sm.entries[i].SourceStart += byteOff
		sm.entries[i].SourceEnd += byteOff
	}

	return content, sm, lm
}

// Stitch merges newly parsed content for blocks [startBlock, endBlock)
// into the existing parse result, adjusting positions in the suffix.
// sourceDelta is the change in source rune count (newLen - oldLen).
// sourceBytesDelta is the change in source byte count.
// renderedDelta is computed internally from the old/new region sizes.
func Stitch(
	old StitchResult,
	newRegion StitchResult,
	startBlock, endBlock int,
	sourceDelta int,
	sourceBytesDelta int,
) StitchResult {
	if old.BlockIdx == nil || len(old.BlockIdx.Blocks) == 0 {
		return newRegion
	}

	blocks := old.BlockIdx.Blocks

	// Clamp block indices.
	if startBlock < 0 {
		startBlock = 0
	}
	if endBlock > len(blocks) {
		endBlock = len(blocks)
	}

	// Compute prefix/suffix boundaries from block info.
	// Prefix: content and entries from blocks [0, startBlock).
	// Suffix: content and entries from blocks [endBlock, len(blocks)).

	// We need to find the content, SM, and LM ranges for prefix and suffix.
	// Since BlockInfo doesn't always have ContentStart/End populated,
	// we compute these boundaries from source rune positions by scanning
	// the content and source map entries.

	var prefixContentEnd int   // number of content spans in prefix
	var prefixSMEnd int        // number of SM entries in prefix
	var prefixLMEnd int        // number of LM entries in prefix
	var prefixRenderedEnd int  // rendered rune position at end of prefix

	var suffixContentStart int  // first content span index in suffix
	var suffixSMStart int       // first SM entry index in suffix
	var suffixLMStart int       // first LM entry index in suffix
	var suffixRenderedStart int // rendered rune position at start of suffix

	if startBlock > 0 {
		// The prefix ends where startBlock begins.
		prefixSourceEnd := blocks[startBlock].SourceRuneStart

		// Find the content span boundary: count spans whose text
		// comes entirely before prefixSourceEnd.
		prefixContentEnd, prefixRenderedEnd = findContentBoundary(old.Content, old.SM, prefixSourceEnd)

		// Find SM entry boundary.
		prefixSMEnd = findSMBoundary(old.SM, prefixSourceEnd)

		// Find LM entry boundary.
		prefixLMEnd = findLMBoundary(old.LM, prefixRenderedEnd)
	}

	if endBlock < len(blocks) {
		// The suffix starts where endBlock begins (in OLD positions).
		suffixSourceStart := blocks[endBlock].SourceRuneStart

		suffixContentStart, suffixRenderedStart = findContentBoundary(old.Content, old.SM, suffixSourceStart)
		suffixSMStart = findSMBoundary(old.SM, suffixSourceStart)
		suffixLMStart = findLMBoundary(old.LM, suffixRenderedStart)
	} else {
		suffixContentStart = len(old.Content)
		if old.SM != nil {
			suffixSMStart = len(old.SM.entries)
		}
		if old.LM != nil {
			suffixLMStart = len(old.LM.entries)
		}
		suffixRenderedStart = old.Content.Len()
	}

	// Compute the rendered delta: the change in rendered rune count for
	// the affected region.
	regionRenderedLen := newRegion.Content.Len()
	oldRegionRenderedLen := suffixRenderedStart - prefixRenderedEnd
	renderedDelta := regionRenderedLen - oldRegionRenderedLen

	// Build result content: prefix + shifted region + shifted suffix.
	var resultContent rich.Content
	resultContent = append(resultContent, old.Content[:prefixContentEnd]...)
	resultContent = append(resultContent, newRegion.Content...)
	resultContent = append(resultContent, old.Content[suffixContentStart:]...)

	// Build result source map: prefix entries + shifted region entries + shifted suffix entries.
	resultSM := &SourceMap{}
	if old.SM != nil {
		resultSM.entries = append(resultSM.entries, old.SM.entries[:prefixSMEnd]...)
	}

	// Shift new region SM entries: rendered positions need to be shifted
	// by prefixRenderedEnd (region positions start from 0).
	if newRegion.SM != nil {
		for _, e := range newRegion.SM.entries {
			e.RenderedStart += prefixRenderedEnd
			e.RenderedEnd += prefixRenderedEnd
			resultSM.entries = append(resultSM.entries, e)
		}
	}

	// Shift suffix SM entries: source byte positions shift by sourceBytesDelta,
	// rendered positions shift by renderedDelta.
	// Note: SourceRuneStart/End are NOT shifted here. The caller must call
	// PopulateRunePositions(fullSource) after stitching to derive correct
	// rune positions from the globally-shifted byte positions.
	if old.SM != nil {
		for _, e := range old.SM.entries[suffixSMStart:] {
			e.SourceStart += sourceBytesDelta
			e.SourceEnd += sourceBytesDelta
			e.RenderedStart += renderedDelta
			e.RenderedEnd += renderedDelta
			resultSM.entries = append(resultSM.entries, e)
		}
	}

	// Build result link map: prefix entries + shifted region entries + shifted suffix entries.
	resultLM := NewLinkMap()
	if old.LM != nil {
		resultLM.entries = append(resultLM.entries, old.LM.entries[:prefixLMEnd]...)
	}

	// Shift new region LM entries by prefixRenderedEnd.
	if newRegion.LM != nil {
		for _, e := range newRegion.LM.entries {
			e.Start += prefixRenderedEnd
			e.End += prefixRenderedEnd
			resultLM.entries = append(resultLM.entries, e)
		}
	}

	// Shift suffix LM entries by renderedDelta.
	if old.LM != nil {
		for _, e := range old.LM.entries[suffixLMStart:] {
			e.Start += renderedDelta
			e.End += renderedDelta
			resultLM.entries = append(resultLM.entries, e)
		}
	}

	// Build result block index: prefix blocks + region blocks + shifted suffix blocks.
	resultBI := &BlockIndex{}
	if old.BlockIdx != nil {
		resultBI.Blocks = append(resultBI.Blocks, blocks[:startBlock]...)
	}
	if newRegion.BlockIdx != nil {
		resultBI.Blocks = append(resultBI.Blocks, newRegion.BlockIdx.Blocks...)
	}
	if old.BlockIdx != nil {
		for _, b := range blocks[endBlock:] {
			b.SourceRuneStart += sourceDelta
			b.SourceRuneEnd += sourceDelta
			b.SourceLineStart += 0 // line indices are not shifted here
			b.SourceLineEnd += 0
			resultBI.Blocks = append(resultBI.Blocks, b)
		}
	}

	return StitchResult{
		Content:  resultContent,
		SM:       resultSM,
		LM:       resultLM,
		BlockIdx: resultBI,
	}
}

// findContentBoundary finds the index into content and the rendered rune
// position at which a given source rune position falls. It returns the
// number of complete content spans before sourceRunePos and the total
// rendered rune count of those spans.
func findContentBoundary(content rich.Content, sm *SourceMap, sourceRunePos int) (spanIdx, renderedPos int) {
	if sm == nil || len(sm.entries) == 0 {
		return 0, 0
	}

	// Find the rendered position corresponding to sourceRunePos by scanning
	// source map entries. The rendered position at sourceRunePos is the
	// RenderedStart of the first entry whose SourceRuneStart >= sourceRunePos.
	targetRendered := 0
	found := false
	for _, e := range sm.entries {
		if e.SourceRuneStart >= sourceRunePos {
			targetRendered = e.RenderedStart
			found = true
			break
		}
	}
	if !found {
		// sourceRunePos is beyond all entries — everything is prefix.
		return len(content), content.Len()
	}

	// Now find which content span boundary corresponds to targetRendered.
	rp := 0
	for i, s := range content {
		spanLen := len([]rune(s.Text))
		if rp+spanLen > targetRendered {
			// This span contains or starts at the target rendered position.
			// If the span starts exactly at targetRendered, the boundary is here.
			if rp == targetRendered {
				return i, rp
			}
			// The span straddles the boundary — include it in prefix.
			// (This can happen when block boundaries don't align with span boundaries.)
			return i, rp
		}
		rp += spanLen
	}
	return len(content), rp
}

// findSMBoundary finds the index of the first source map entry whose
// SourceRuneStart >= sourceRunePos.
func findSMBoundary(sm *SourceMap, sourceRunePos int) int {
	if sm == nil {
		return 0
	}
	for i, e := range sm.entries {
		if e.SourceRuneStart >= sourceRunePos {
			return i
		}
	}
	return len(sm.entries)
}

// findLMBoundary finds the index of the first link map entry whose
// Start >= renderedPos.
func findLMBoundary(lm *LinkMap, renderedPos int) int {
	if lm == nil {
		return 0
	}
	for i, e := range lm.entries {
		if e.Start >= renderedPos {
			return i
		}
	}
	return len(lm.entries)
}

// IncrementalUpdate attempts an incremental re-parse of the changed region.
// It takes the previous parse result, the new (post-edit) full source text,
// and the accumulated edits since the last update.
// Returns the updated StitchResult and true on success, or a zero StitchResult
// and false if a full re-parse is required (e.g., fence delimiter edits).
func IncrementalUpdate(old StitchResult, newSource string, edits []EditRecord) (StitchResult, bool) {
	if old.BlockIdx == nil || len(old.BlockIdx.Blocks) == 0 || len(edits) == 0 {
		return StitchResult{}, false
	}

	startBlock, endBlock := old.BlockIdx.AffectedRange(edits)
	if startBlock == -1 || endBlock == -1 {
		return StitchResult{}, false
	}

	// Compute source rune delta from the coalesced edits.
	sourceDelta := 0
	for _, e := range edits {
		sourceDelta += e.NewLen - e.OldLen
	}

	// Compute byte delta from stored old source byte length.
	sourceBytesDelta := len(newSource) - old.BlockIdx.SourceByteLen

	newLines := splitLines(newSource)

	// Compute the source rune range of affected blocks in old source.
	oldBlockStart := old.BlockIdx.Blocks[startBlock].SourceRuneStart
	lastBlock := endBlock - 1
	oldBlockEnd := old.BlockIdx.Blocks[lastBlock].SourceRuneEnd

	// Find new block boundaries: affected region in the new source.
	newBlockEnd := oldBlockEnd + sourceDelta
	if newBlockEnd < oldBlockStart {
		newBlockEnd = oldBlockStart
	}

	// Find line range in the new source covering [oldBlockStart, newBlockEnd).
	newLineStart := 0
	newLineEnd := len(newLines)
	runePos := 0
	for i, l := range newLines {
		lineEnd := runePos + utf8.RuneCountInString(l)
		if runePos <= oldBlockStart && lineEnd > oldBlockStart {
			newLineStart = i
		}
		if lineEnd >= newBlockEnd && newLineEnd == len(newLines) {
			newLineEnd = i + 1
			break
		}
		runePos = lineEnd
	}

	// Extend the region to include any trailing blank lines at the boundary.
	// This ensures paragraph separators are included in the region, producing
	// the correct paragraph break in the parsed output.
	for newLineEnd < len(newLines) {
		line := strings.TrimRight(newLines[newLineEnd], "\n")
		if line != "" {
			break
		}
		newLineEnd++
	}

	// Compute source rune offset and byte offset for the region.
	sourceRuneOffset := 0
	sourceByteOffset := 0
	for _, l := range newLines[:newLineStart] {
		sourceRuneOffset += utf8.RuneCountInString(l)
		sourceByteOffset += len(l)
	}

	// Parse the affected region.
	regionLines := newLines[newLineStart:newLineEnd]
	regionContent, regionSM, regionLM := ParseRegion(
		regionLines,
		sourceRuneOffset,
		sourceByteOffset,
	)

	// Build the region block index.
	regionText := strings.Join(regionLines, "")
	regionBI := buildBlockIndex(regionText)

	// Shift region block index rune positions by sourceRuneOffset.
	for i := range regionBI.Blocks {
		regionBI.Blocks[i].SourceRuneStart += sourceRuneOffset
		regionBI.Blocks[i].SourceRuneEnd += sourceRuneOffset
	}

	newRegion := StitchResult{
		Content:  regionContent,
		SM:       regionSM,
		LM:       regionLM,
		BlockIdx: regionBI,
	}

	result := Stitch(old, newRegion, startBlock, endBlock, sourceDelta, sourceBytesDelta)

	// Update the SourceByteLen on the result block index.
	result.BlockIdx.SourceByteLen = len(newSource)

	// Re-populate rune positions from globally-correct byte positions.
	result.SM.PopulateRunePositions(newSource)

	return result, true
}
