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

	bi := &BlockIndex{lineRuneStarts: lineRuneStarts}

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
			emitBlock(BlockListItem, i, i+1)
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
