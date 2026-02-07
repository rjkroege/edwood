package markdown

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/rjkroege/edwood/rich"
)

// headingScales maps heading level (1-6) to scale factor.
var headingScales = [7]float64{
	0: 1.0,   // not used (no level 0)
	1: 2.0,   // H1
	2: 1.5,   // H2
	3: 1.25,  // H3
	4: 1.125, // H4
	5: 1.0,   // H5
	6: 0.875, // H6
}

// listCtx tracks list context for nested blocks within a list item.
type listCtx struct {
	contentCol int  // column where content starts (for continuation detection)
	indentLvl  int  // nesting level (for ListIndent)
	ordered    bool // true for ordered lists
	itemNumber int  // item number for ordered lists
}

// Parse converts markdown text to styled rich.Content.
func Parse(text string) rich.Content {
	return parseInternal(text, nil, nil)
}

// ParseWithSourceMap parses markdown and returns the styled content,
// a source map for mapping rendered positions back to source positions,
// and a link map for tracking which rendered positions contain links.
func ParseWithSourceMap(text string) (rich.Content, *SourceMap, *LinkMap) {
	if text == "" {
		return rich.Content{}, &SourceMap{}, NewLinkMap()
	}
	sm := &SourceMap{}
	lm := NewLinkMap()
	result := parseInternal(text, sm, lm)
	sm.populateRunePositions(text)
	return result, sm, lm
}

// parseInternal is the unified markdown parser. When sm and lm are non-nil,
// source map entries and link entries are accumulated; otherwise only
// rich.Content is produced.
func parseInternal(text string, sm *SourceMap, lm *LinkMap) rich.Content {
	if text == "" {
		return rich.Content{}
	}

	tracking := sm != nil

	var result rich.Content
	lines := splitLines(text)

	sourcePos := 0
	renderedPos := 0

	// Track fenced code block state
	inFencedBlock := false
	var codeBlockContent strings.Builder
	codeBlockSourceStart := 0

	// Track indented code block state
	inIndentedBlock := false
	var indentedBlockContent strings.Builder
	indentedBlockSourceStart := 0

	// Track list context for nested blocks
	var activeList *listCtx
	inListCodeBlock := false
	var listCodeContent strings.Builder
	listCodeSourceStart := 0

	// Helper to emit a code block accumulated within a list item
	emitListCodeBlock := func() {
		if listCodeContent.Len() > 0 {
			codeContent := listCodeContent.String()
			codeSpan := rich.Span{
				Text: codeContent,
				Style: rich.Style{
					Bg:         rich.InlineCodeBg,
					Code:       true,
					Block:      true,
					ListItem:   true,
					ListIndent: activeList.indentLvl,
					Scale:      1.0,
				},
			}

			if tracking {
				codeLen := len([]rune(codeContent))
				sm.entries = append(sm.entries, SourceMapEntry{
					RenderedStart: renderedPos,
					RenderedEnd:   renderedPos + codeLen,
					SourceStart:   listCodeSourceStart,
					SourceEnd:     sourcePos,
				})
				renderedPos += codeLen
			}

			result = append(result, codeSpan)
			listCodeContent.Reset()
		}
		inListCodeBlock = false
	}

	// Track blockquote-within-list state
	inListBlockquote := false
	listBlockquoteDepth := 0
	listBlockquoteHadNewline := false

	// Helper to end a blockquote within a list item
	endListBlockquote := func() {
		if inListBlockquote && len(result) > 0 && listBlockquoteHadNewline {
			result[len(result)-1].Text += "\n"
			if tracking {
				renderedPos++
			}
		}
		inListBlockquote = false
		listBlockquoteDepth = 0
		listBlockquoteHadNewline = false
	}

	// Helper to end the active list context
	endListContext := func() {
		if inListCodeBlock {
			emitListCodeBlock()
		}
		if inListBlockquote {
			endListBlockquote()
		}
		activeList = nil
	}

	// Track blockquote state
	inBlockquote := false
	blockquoteDepth := 0
	blockquoteLineHadNewline := false

	// Track fenced code block within blockquote
	inBQCodeBlock := false
	var bqCodeContent strings.Builder
	bqCodeSourceStart := 0
	bqCodeDepth := 0

	// Helper to emit a code block accumulated within a blockquote
	emitBQCodeBlock := func() {
		if bqCodeContent.Len() > 0 {
			codeContent := bqCodeContent.String()
			codeSpan := rich.Span{
				Text: codeContent,
				Style: rich.Style{
					Bg:              rich.InlineCodeBg,
					Code:            true,
					Block:           true,
					Blockquote:      true,
					BlockquoteDepth: bqCodeDepth,
					Scale:           1.0,
				},
			}

			if tracking {
				codeLen := len([]rune(codeContent))
				sm.entries = append(sm.entries, SourceMapEntry{
					RenderedStart: renderedPos,
					RenderedEnd:   renderedPos + codeLen,
					SourceStart:   bqCodeSourceStart,
					SourceEnd:     sourcePos,
				})
				renderedPos += codeLen
			}

			result = append(result, codeSpan)
			bqCodeContent.Reset()
		}
		inBQCodeBlock = false
	}

	// Helper to end the current blockquote by appending \n to the last span
	endBlockquote := func() {
		if inBQCodeBlock {
			emitBQCodeBlock()
		}
		if inBlockquote && len(result) > 0 && blockquoteLineHadNewline {
			result[len(result)-1].Text += "\n"
			if tracking {
				renderedPos++
			}
		}
		inBlockquote = false
		blockquoteDepth = 0
		blockquoteLineHadNewline = false
	}

	// Track paragraph state for joining lines
	inParagraph := false

	// Helper to emit indented code block
	emitIndentedBlock := func() {
		if indentedBlockContent.Len() > 0 {
			codeContent := indentedBlockContent.String()
			codeSpan := rich.Span{
				Text: codeContent,
				Style: rich.Style{
					Bg:    rich.InlineCodeBg,
					Code:  true,
					Block: true,
					Scale: 1.0,
				},
			}

			if tracking {
				codeLen := len([]rune(codeContent))
				sm.entries = append(sm.entries, SourceMapEntry{
					RenderedStart: renderedPos,
					RenderedEnd:   renderedPos + codeLen,
					SourceStart:   indentedBlockSourceStart,
					SourceEnd:     sourcePos,
				})
				renderedPos += codeLen
			}

			// Merge or append
			if len(result) > 0 && result[len(result)-1].Style == codeSpan.Style {
				result[len(result)-1].Text += codeSpan.Text
			} else {
				result = append(result, codeSpan)
			}
			indentedBlockContent.Reset()
		}
		inIndentedBlock = false
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// --- List context: handle continuation lines within a list item ---
		if activeList != nil {
			// Inside a list code block: accumulate or close
			if inListCodeBlock {
				stripped, ok := stripListIndent(line, activeList.contentCol)
				if ok && isFenceDelimiter(stripped) {
					emitListCodeBlock()
					if tracking {
						sourcePos += len(line)
					}
					continue
				}
				if ok {
					listCodeContent.WriteString(stripped)
				} else {
					trimmed := strings.TrimRight(line, "\n")
					if trimmed == "" {
						listCodeContent.WriteString(line)
					} else {
						emitListCodeBlock()
						endListContext()
						goto normalDispatch
					}
				}
				if tracking {
					sourcePos += len(line)
				}
				continue
			}

			// Not in a list code block — check if this is a continuation line
			isULCont, contIndent, _ := isUnorderedListItem(line)
			isOLCont, contOLIndent, _, _ := isOrderedListItem(line)
			if isULCont && contIndent <= activeList.indentLvl {
				endListContext()
				goto normalDispatch
			}
			if isOLCont && contOLIndent <= activeList.indentLvl {
				endListContext()
				goto normalDispatch
			}

			stripped, ok := stripListIndent(line, activeList.contentCol)
			if ok {
				if isFenceDelimiter(stripped) {
					inListCodeBlock = true
					listCodeContent.Reset()
					if tracking {
						sourcePos += len(line)
						listCodeSourceStart = sourcePos
					}
					continue
				}
				if isIndentedCodeLine(stripped) {
					codeContent := stripIndent(stripped)
					codeSpan := rich.Span{
						Text: codeContent,
						Style: rich.Style{
							Bg:         rich.InlineCodeBg,
							Code:       true,
							Block:      true,
							ListItem:   true,
							ListIndent: activeList.indentLvl,
							Scale:      1.0,
						},
					}

					if tracking {
						codeLen := len([]rune(codeContent))
						sm.entries = append(sm.entries, SourceMapEntry{
							RenderedStart: renderedPos,
							RenderedEnd:   renderedPos + codeLen,
							SourceStart:   sourcePos,
							SourceEnd:     sourcePos + len(line),
						})
						renderedPos += codeLen
					}

					result = append(result, codeSpan)
					if tracking {
						sourcePos += len(line)
					}
					continue
				}
				// Check for blockquote within list item
				if isBQ, depth, bqContentStart := isBlockquoteLine(stripped); isBQ {
					lineHadNewline := strings.HasSuffix(stripped, "\n")
					content := strings.TrimSuffix(stripped[bqContentStart:], "\n")

					if inBlockquote {
						endBlockquote()
					}

					if content == "" {
						if tracking {
							sourcePos += len(line)
						}
						continue
					}

					if inListBlockquote && depth == listBlockquoteDepth {
						if len(result) > 0 {
							lastSpan := &result[len(result)-1]
							if !strings.HasSuffix(lastSpan.Text, " ") {
								lastSpan.Text += " "
								if tracking {
									renderedPos++
								}
							}
						}
					} else if inListBlockquote && depth != listBlockquoteDepth {
						endListBlockquote()
					}

					bqBaseStyle := rich.Style{
						Blockquote:      true,
						BlockquoteDepth: depth,
						ListItem:        true,
						ListIndent:      activeList.indentLvl,
						Scale:           1.0,
					}

					var bqSMEntries *[]SourceMapEntry
					var bqLMEntries *[]LinkEntry
					var bqEntries []SourceMapEntry
					var bqLinkEntries []LinkEntry
					listIndentBytes := len(line) - len(stripped)
					if tracking {
						bqSMEntries = &bqEntries
						bqLMEntries = &bqLinkEntries
					}
					contentSpans := parseInline(content, bqBaseStyle, InlineOpts{
						SourceMap:      bqSMEntries,
						LinkMap:        bqLMEntries,
						SourceOffset:   sourcePos + listIndentBytes + bqContentStart,
						RenderedOffset: renderedPos,
					})

					if tracking {
						if len(bqEntries) > 0 {
							bqEntries[0].SourceStart = sourcePos
							bqEntries[0].PrefixLen = listIndentBytes + bqContentStart
							bqEntries[0].Kind = KindPrefix
						}
						sm.entries = append(sm.entries, bqEntries...)
						for _, le := range bqLinkEntries {
							lm.Add(le.Start, le.End, le.URL)
						}
						for _, span := range contentSpans {
							renderedPos += len([]rune(span.Text))
						}
						sourcePos += len(line)
					}

					for _, span := range contentSpans {
						if len(result) > 0 && result[len(result)-1].Style == span.Style && !span.Style.Link {
							result[len(result)-1].Text += span.Text
						} else {
							result = append(result, span)
						}
					}

					inListBlockquote = true
					listBlockquoteDepth = depth
					listBlockquoteHadNewline = lineHadNewline
					continue
				}
				endListContext()
				goto normalDispatch
			}

			trimmedCont := strings.TrimRight(line, "\n")
			if trimmedCont == "" {
				endListContext()
				goto normalDispatch
			}

			endListContext()
			goto normalDispatch
		}

	normalDispatch:
		// Check for fenced code block delimiter
		if isFenceDelimiter(line) {
			if inIndentedBlock {
				emitIndentedBlock()
			}
			if inBlockquote {
				endBlockquote()
			}
			if inParagraph && len(result) > 0 {
				result[len(result)-1].Text += "\n"
				if tracking {
					renderedPos++
				}
			}
			inParagraph = false
			if !inFencedBlock {
				inFencedBlock = true
				codeBlockContent.Reset()
				if tracking {
					sourcePos += len(line)
					codeBlockSourceStart = sourcePos
				}
				continue
			} else {
				inFencedBlock = false
				codeContent := codeBlockContent.String()
				if codeContent != "" {
					codeSpan := rich.Span{
						Text: codeContent,
						Style: rich.Style{
							Bg:    rich.InlineCodeBg,
							Code:  true,
							Block: true,
							Scale: 1.0,
						},
					}

					if tracking {
						codeLen := len([]rune(codeContent))
						sm.entries = append(sm.entries, SourceMapEntry{
							RenderedStart: renderedPos,
							RenderedEnd:   renderedPos + codeLen,
							SourceStart:   codeBlockSourceStart,
							SourceEnd:     sourcePos,
						})
						renderedPos += codeLen
					}

					// Merge or append the code span
					if len(result) > 0 && result[len(result)-1].Style == codeSpan.Style {
						result[len(result)-1].Text += codeSpan.Text
					} else {
						result = append(result, codeSpan)
					}
				}
				if tracking {
					sourcePos += len(line)
				}
				continue
			}
		}

		if inFencedBlock {
			codeBlockContent.WriteString(line)
			if tracking {
				sourcePos += len(line)
			}
			continue
		}

		isULEarly, _, _ := isUnorderedListItem(line)
		isOLEarly, _, _, _ := isOrderedListItem(line)
		isListItemEarly := isULEarly || isOLEarly

		if isIndentedCodeLine(line) && !isListItemEarly {
			if inBlockquote {
				endBlockquote()
			}
			if inParagraph && len(result) > 0 {
				result[len(result)-1].Text += "\n"
				if tracking {
					renderedPos++
				}
			}
			inParagraph = false
			if !inIndentedBlock {
				inIndentedBlock = true
				indentedBlockSourceStart = sourcePos
				indentedBlockContent.Reset()
			}
			indentedBlockContent.WriteString(stripIndent(line))
			if tracking {
				sourcePos += len(line)
			}
			continue
		}

		if inIndentedBlock {
			emitIndentedBlock()
		}

		// Check for blank line (paragraph break)
		trimmedLine := strings.TrimRight(line, "\n")
		if trimmedLine == "" {
			wasInBlockquote := inBlockquote
			if inBlockquote {
				endBlockquote()
			}
			if inParagraph || wasInBlockquote {
				if len(result) > 0 {
					result = append(result, rich.Span{
						Text:  "\n",
						Style: rich.Style{ParaBreak: true, Scale: 1.0},
					})
					if tracking {
						renderedPos++
					}
				}
			}
			inParagraph = false
			if tracking {
				sourcePos += len(line)
			}
			continue
		}

		// Check for table
		isRow, _ := isTableRow(line)
		if isRow && i+1 < len(lines) && isTableSeparatorRow(lines[i+1]) {
			if inBlockquote {
				endBlockquote()
			}
			if inParagraph && len(result) > 0 {
				result[len(result)-1].Text += "\n"
				if tracking {
					renderedPos++
				}
			}
			inParagraph = false

			var smEntriesPtr *[]SourceMapEntry
			if tracking {
				smEntriesPtr = &sm.entries
			}
			tableSpans, consumed := parseTableBlockInternal(lines, i, sourcePos, renderedPos, smEntriesPtr)
			result = append(result, tableSpans...)

			if tracking {
				for j := 0; j < consumed; j++ {
					sourcePos += len(lines[i+j])
				}
				for _, span := range tableSpans {
					renderedPos += len([]rune(span.Text))
				}
			}
			i += consumed - 1
			continue
		}

		// Check for blockquote
		if isBQ, depth, contentStart := isBlockquoteLine(line); isBQ {
			if inParagraph && len(result) > 0 {
				result[len(result)-1].Text += "\n"
				if tracking {
					renderedPos++
				}
			}
			inParagraph = false

			lineHadNewline := strings.HasSuffix(line, "\n")
			content := strings.TrimSuffix(line[contentStart:], "\n")

			// --- Fenced code block within blockquote ---
			if inBQCodeBlock {
				if depth != bqCodeDepth {
					// Depth changed: emit code block and fall through to normal BQ handling
					emitBQCodeBlock()
				} else if isFenceDelimiter(content) {
					// Closing fence
					emitBQCodeBlock()
					inBlockquote = false
					blockquoteDepth = 0
					blockquoteLineHadNewline = false
					if tracking {
						sourcePos += len(line)
					}
					continue
				} else {
					// Accumulate code content
					bqCodeContent.WriteString(content + "\n")
					if tracking {
						sourcePos += len(line)
					}
					continue
				}
			}

			if !inBQCodeBlock && isFenceDelimiter(content) {
				// Opening fence: finalize any current BQ text (trailing newline)
				if inBlockquote && len(result) > 0 && blockquoteLineHadNewline {
					result[len(result)-1].Text += "\n"
					if tracking {
						renderedPos++
					}
				}
				inBQCodeBlock = true
				bqCodeContent.Reset()
				bqCodeDepth = depth
				if tracking {
					sourcePos += len(line)
					bqCodeSourceStart = sourcePos
				}
				inBlockquote = true
				blockquoteDepth = depth
				blockquoteLineHadNewline = false
				continue
			}

			if content == "" {
				if inBlockquote {
					endBlockquote()
					result = append(result, rich.Span{
						Text: "\n",
						Style: rich.Style{
							ParaBreak:       true,
							Blockquote:      true,
							BlockquoteDepth: depth,
							Scale:           1.0,
						},
					})
					if tracking {
						renderedPos++
					}
				}
				if tracking {
					sourcePos += len(line)
				}
				continue
			}

			if inBlockquote && depth != blockquoteDepth {
				endBlockquote()
			}

			if inBlockquote && depth == blockquoteDepth {
				if len(result) > 0 {
					lastSpan := &result[len(result)-1]
					if !strings.HasSuffix(lastSpan.Text, " ") {
						lastSpan.Text += " "
						if tracking {
							renderedPos++
						}
					}
				}
			}

			bqBaseStyle := rich.Style{
				Blockquote:      true,
				BlockquoteDepth: depth,
				Scale:           1.0,
			}

			var bqSMEntries *[]SourceMapEntry
			var bqLMEntries *[]LinkEntry
			var bqEntries []SourceMapEntry
			var bqLinkEntries []LinkEntry
			if tracking {
				bqSMEntries = &bqEntries
				bqLMEntries = &bqLinkEntries
			}
			contentSpans := parseInline(content, bqBaseStyle, InlineOpts{
				SourceMap:      bqSMEntries,
				LinkMap:        bqLMEntries,
				SourceOffset:   sourcePos + contentStart,
				RenderedOffset: renderedPos,
			})

			if tracking {
				if len(bqEntries) > 0 {
					bqEntries[0].SourceStart = sourcePos
					bqEntries[0].PrefixLen = contentStart
					bqEntries[0].Kind = KindPrefix
				}
				sm.entries = append(sm.entries, bqEntries...)
				for _, le := range bqLinkEntries {
					lm.Add(le.Start, le.End, le.URL)
				}
				for _, span := range contentSpans {
					renderedPos += len([]rune(span.Text))
				}
				sourcePos += len(line)
			}

			for _, span := range contentSpans {
				if len(result) > 0 && result[len(result)-1].Style == span.Style && !span.Style.Link {
					result[len(result)-1].Text += span.Text
				} else {
					result = append(result, span)
				}
			}

			inBlockquote = true
			blockquoteDepth = depth
			blockquoteLineHadNewline = lineHadNewline
			continue
		}

		if inBlockquote {
			endBlockquote()
		}

		// Check if this is a block-level element
		isUL, ulIndent, ulContentStart := isUnorderedListItem(line)
		isOL, olIndent, olContentStart, olItemNum := isOrderedListItem(line)
		isListItem := isUL || isOL
		isBlockElement := headingLevel(line) > 0 || isHorizontalRule(line) || isListItem

		if isBlockElement {
			if inParagraph && len(result) > 0 {
				result[len(result)-1].Text += "\n"
				if tracking {
					renderedPos++
				}
			}
			inParagraph = false
		} else {
			if inParagraph && len(result) > 0 {
				lastSpan := &result[len(result)-1]
				if strings.HasSuffix(lastSpan.Text, "\n") {
					lastSpan.Text = strings.TrimSuffix(lastSpan.Text, "\n") + " "
				} else if !strings.HasSuffix(lastSpan.Text, " ") {
					lastSpan.Text += " "
					if tracking {
						renderedPos++
					}
				}
			}
			inParagraph = true
		}

		lineToPass := line
		if !isBlockElement {
			lineToPass = strings.TrimSuffix(line, "\n")
		}

		// Normal line parsing
		var smEntriesPtr *[]SourceMapEntry
		var lmEntriesPtr *[]LinkEntry
		var lineEntries []SourceMapEntry
		var lineLinkEntries []LinkEntry
		if tracking {
			smEntriesPtr = &lineEntries
			lmEntriesPtr = &lineLinkEntries
		}
		spans := parseLineInternal(lineToPass, sourcePos, renderedPos, smEntriesPtr, lmEntriesPtr)

		if tracking {
			sm.entries = append(sm.entries, lineEntries...)
			for _, le := range lineLinkEntries {
				lm.Add(le.Start, le.End, le.URL)
			}
			for _, span := range spans {
				renderedPos += len([]rune(span.Text))
			}
			sourcePos += len(line)
		}

		// Merge consecutive spans with the same style
		// (but don't merge link spans or list item spans - each should remain distinct)
		for _, span := range spans {
			if len(result) > 0 && result[len(result)-1].Style == span.Style && !span.Style.Link && !span.Style.ListItem && !span.Style.ListBullet {
				result[len(result)-1].Text += span.Text
			} else {
				result = append(result, span)
			}
		}

		if isListItem {
			if isUL {
				activeList = &listCtx{
					contentCol: ulContentStart,
					indentLvl:  ulIndent,
				}
			} else {
				activeList = &listCtx{
					contentCol: olContentStart,
					indentLvl:  olIndent,
					ordered:    true,
					itemNumber: olItemNum,
				}
			}
		}
	}

	if activeList != nil {
		endListContext()
	}

	if inBQCodeBlock {
		emitBQCodeBlock()
	}

	if inBlockquote {
		endBlockquote()
	}

	if inFencedBlock {
		codeContent := codeBlockContent.String()
		if codeContent != "" {
			codeSpan := rich.Span{
				Text: codeContent,
				Style: rich.Style{
					Bg:    rich.InlineCodeBg,
					Code:  true,
					Block: true,
					Scale: 1.0,
				},
			}

			if tracking {
				codeLen := len([]rune(codeContent))
				sm.entries = append(sm.entries, SourceMapEntry{
					RenderedStart: renderedPos,
					RenderedEnd:   renderedPos + codeLen,
					SourceStart:   codeBlockSourceStart,
					SourceEnd:     sourcePos,
				})
				renderedPos += codeLen
			}

			result = append(result, codeSpan)
		}
	}

	if inIndentedBlock {
		emitIndentedBlock()
	}

	return result
}

// isIndentedCodeLine returns true if the line is an indented code line
// (starts with 4 spaces or 1 tab).
func isIndentedCodeLine(line string) bool {
	if len(line) == 0 {
		return false
	}
	// Check for tab indent
	if line[0] == '\t' {
		return true
	}
	// Check for 4-space indent
	if len(line) >= 4 && line[0:4] == "    " {
		return true
	}
	return false
}

// stripIndent removes the leading indent (4 spaces or 1 tab) from a line.
func stripIndent(line string) string {
	if len(line) == 0 {
		return line
	}
	if line[0] == '\t' {
		return line[1:]
	}
	if len(line) >= 4 && line[0:4] == "    " {
		return line[4:]
	}
	return line
}

// stripListIndent strips leading whitespace up to contentCol columns from a line.
// Returns the stripped line and true if the line was indented to at least contentCol,
// or ("", false) if the line has insufficient indentation.
// Tabs count as advancing to the next tab stop (every 4 columns).
func stripListIndent(line string, contentCol int) (string, bool) {
	col := 0
	byteIdx := 0
	for byteIdx < len(line) && col < contentCol {
		if line[byteIdx] == ' ' {
			col++
			byteIdx++
		} else if line[byteIdx] == '\t' {
			col += 4 - (col % 4) // advance to next tab stop
			byteIdx++
		} else {
			// Non-whitespace before reaching contentCol
			return "", false
		}
	}
	if col < contentCol {
		return "", false
	}
	return line[byteIdx:], true
}

// isBlockquoteLine checks if a line starts with a blockquote marker.
// Returns: (isBlockquote bool, depth int, contentStart int)
// - isBlockquote: true if the line starts with `>`
// - depth: nesting level (number of `>` markers)
// - contentStart: byte index where inner content begins (after all `> ` prefixes)
func isBlockquoteLine(line string) (bool, int, int) {
	if len(line) == 0 || line[0] != '>' {
		return false, 0, 0
	}

	depth := 0
	i := 0
	for i < len(line) && line[i] == '>' {
		depth++
		i++
		// Skip optional space after each >
		if i < len(line) && line[i] == ' ' {
			i++
		}
	}

	return true, depth, i
}

// applyBlockquoteStyle sets blockquote fields on all spans.
func applyBlockquoteStyle(spans []rich.Span, depth int) {
	for i := range spans {
		spans[i].Style.Blockquote = true
		spans[i].Style.BlockquoteDepth = depth
	}
}

// isFenceDelimiter returns true if the line is a fenced code block delimiter (```).
// This handles lines like "```", "```go", "```python ", etc.
func isFenceDelimiter(line string) bool {
	// Strip trailing newline for comparison
	trimmed := strings.TrimSuffix(line, "\n")
	// Must start with at least 3 backticks
	if len(trimmed) < 3 {
		return false
	}
	if trimmed[0:3] != "```" {
		return false
	}
	// Rest can be language identifier or empty
	// Language identifier: letters, digits, spaces only (no more backticks)
	rest := trimmed[3:]
	for _, r := range rest {
		if r == '`' {
			return false // Additional backticks not allowed in fence opener
		}
	}
	return true
}

// splitLines splits text into lines, preserving trailing newlines on each line.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}

	var lines []string
	for {
		idx := strings.Index(text, "\n")
		if idx == -1 {
			// Last line, no trailing newline
			if text != "" {
				lines = append(lines, text)
			}
			break
		}
		// Include the newline in the line
		lines = append(lines, text[:idx+1])
		text = text[idx+1:]
	}
	return lines
}

// parseLine parses a single line and returns the appropriate spans.
func parseLine(line string) []rich.Span {
	return parseLineInternal(line, 0, 0, nil, nil)
}

// parseLineInternal parses a single line and returns spans, optionally accumulating
// source map entries and link entries when the pointer parameters are non-nil.
func parseLineInternal(line string, sourceOffset, renderedOffset int,
	smEntries *[]SourceMapEntry, lmEntries *[]LinkEntry) []rich.Span {

	// Check for horizontal rule (---, ***, ___)
	if isHorizontalRule(line) {
		// Emit the HRuleRune marker plus newline if the original line had one
		text := string(rich.HRuleRune)
		hasNewline := strings.HasSuffix(line, "\n")
		if hasNewline {
			text += "\n"
		}

		if smEntries != nil {
			sourceWithoutNewline := strings.TrimSuffix(line, "\n")
			*smEntries = append(*smEntries, SourceMapEntry{
				RenderedStart: renderedOffset,
				RenderedEnd:   renderedOffset + 1,
				SourceStart:   sourceOffset,
				SourceEnd:     sourceOffset + len(sourceWithoutNewline),
			})
			if hasNewline {
				*smEntries = append(*smEntries, SourceMapEntry{
					RenderedStart: renderedOffset + 1,
					RenderedEnd:   renderedOffset + 2,
					SourceStart:   sourceOffset + len(sourceWithoutNewline),
					SourceEnd:     sourceOffset + len(line),
				})
			}
		}

		return []rich.Span{{
			Text:  text,
			Style: rich.StyleHRule,
		}}
	}

	// Check for heading (# at start of line)
	level := headingLevel(line)
	if level > 0 {
		// Extract heading text (strip # prefix and one leading space)
		prefixLen := level
		content := line[level:]
		if len(content) > 0 && content[0] == ' ' {
			content = content[1:]
			prefixLen++
		}

		if smEntries != nil {
			renderedLen := len([]rune(content))
			*smEntries = append(*smEntries, SourceMapEntry{
				RenderedStart: renderedOffset,
				RenderedEnd:   renderedOffset + renderedLen,
				SourceStart:   sourceOffset,
				SourceEnd:     sourceOffset + len(line),
				PrefixLen:     prefixLen,
				Kind:          KindPrefix,
			})
		}

		return []rich.Span{{
			Text: content,
			Style: rich.Style{
				Bold:  true,
				Scale: headingScales[level],
			},
		}}
	}

	// Check for unordered list item (-, *, +)
	if isUL, indentLevel, contentStart := isUnorderedListItem(line); isUL {
		return parseUnorderedListItemInternal(line, indentLevel, contentStart, sourceOffset, renderedOffset, smEntries, lmEntries)
	}

	// Check for ordered list item (1., 2), etc.)
	if isOL, indentLevel, contentStart, itemNumber := isOrderedListItem(line); isOL {
		return parseOrderedListItemInternal(line, indentLevel, contentStart, itemNumber, sourceOffset, renderedOffset, smEntries, lmEntries)
	}

	// Parse inline formatting (bold, italic)
	return parseInline(line, rich.DefaultStyle(), InlineOpts{
		SourceMap:      smEntries,
		LinkMap:        lmEntries,
		SourceOffset:   sourceOffset,
		RenderedOffset: renderedOffset,
	})
}

// parseURLPart extracts the URL and optional title from a URL part.
// Handles formats like: "url", "url 'title'", or "url \"title\"".
// Returns (url, title) where title is the unquoted title string, or "" if absent.
func parseURLPart(urlPart string) (string, string) {
	urlPart = strings.TrimSpace(urlPart)
	if urlPart == "" {
		return "", ""
	}

	// Check for title with double quotes: url "title"
	if idx := strings.Index(urlPart, " \""); idx != -1 {
		url := strings.TrimSpace(urlPart[:idx])
		title := urlPart[idx+2:]
		// Strip trailing quote
		if len(title) > 0 && title[len(title)-1] == '"' {
			title = title[:len(title)-1]
		}
		return url, title
	}

	// Check for title with single quotes: url 'title'
	if idx := strings.Index(urlPart, " '"); idx != -1 {
		url := strings.TrimSpace(urlPart[:idx])
		title := urlPart[idx+2:]
		// Strip trailing quote
		if len(title) > 0 && title[len(title)-1] == '\'' {
			title = title[:len(title)-1]
		}
		return url, title
	}

	return urlPart, ""
}

// parseImageWidth extracts the width in pixels from a title string containing "width=Npx".
// Returns 0 if not found or invalid.
func parseImageWidth(title string) int {
	const prefix = "width="
	const suffix = "px"

	idx := strings.Index(title, prefix)
	if idx == -1 {
		return 0
	}

	// Extract everything after "width="
	rest := title[idx+len(prefix):]

	// Find "px" suffix
	pxIdx := strings.Index(rest, suffix)
	if pxIdx == -1 || pxIdx == 0 {
		return 0
	}

	// Parse the number between "width=" and "px"
	numStr := rest[:pxIdx]
	n := 0
	for _, c := range numStr {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}

	return n
}

// isHorizontalRule returns true if the line is a horizontal rule.
// A horizontal rule is a line containing only 3+ of the same character
// (hyphens, asterisks, or underscores), optionally with spaces between them.
// Examples: "---", "***", "___", "- - -", "* * *", "_ _ _"
func isHorizontalRule(line string) bool {
	// Strip trailing newline for comparison
	trimmed := strings.TrimSuffix(line, "\n")
	if trimmed == "" {
		return false
	}

	// Remove all spaces and check what's left
	noSpaces := strings.ReplaceAll(trimmed, " ", "")
	if len(noSpaces) < 3 {
		return false
	}

	// Check if all remaining characters are the same and valid hrule char
	first := noSpaces[0]
	if first != '-' && first != '*' && first != '_' {
		return false
	}

	for i := 1; i < len(noSpaces); i++ {
		if noSpaces[i] != first {
			return false
		}
	}

	return true
}

// headingLevel returns the heading level (1-6) if line starts with # prefix,
// or 0 if not a heading.
func headingLevel(line string) int {
	if len(line) == 0 || line[0] != '#' {
		return 0
	}

	level := 0
	for i := 0; i < len(line) && i < 6; i++ {
		if line[i] == '#' {
			level++
		} else {
			break
		}
	}

	// Must be followed by a space (or be end of line after stripping newline)
	if level > 0 && level < len(line) {
		next := line[level]
		if next != ' ' && next != '\n' {
			return 0 // Not a valid heading (e.g., "##text" with no space)
		}
	}

	return level
}

// isOrderedListItem returns true if line starts with an ordered list marker.
// Returns: (isListItem bool, indentLevel int, contentStart int, itemNumber int)
// - isListItem: true if the line is an ordered list item
// - indentLevel: the nesting level (0 = top level, 1 = first nested level, etc.)
// - contentStart: the byte index where the item content begins (after "1. ")
// - itemNumber: the number from the list marker (e.g., 1 for "1.")
//
// Ordered list markers are: one or more digits followed by '.' or ')' and a space.
// Indentation is counted as: 2 spaces or 1 tab = 1 indent level.
func isOrderedListItem(line string) (bool, int, int, int) {
	if len(line) == 0 {
		return false, 0, 0, 0
	}

	// Count leading whitespace and calculate indent level
	// 2 spaces = 1 indent level, 1 tab = 1 indent level
	i := 0
	spaceCount := 0
	tabCount := 0
	for i < len(line) {
		if line[i] == ' ' {
			spaceCount++
			i++
		} else if line[i] == '\t' {
			tabCount++
			i++
		} else {
			break
		}
	}

	// Calculate indent level: each tab counts as 1 level, each 2 spaces counts as 1 level
	indentLevel := tabCount + spaceCount/2

	// After whitespace, check for digits
	if i >= len(line) {
		return false, 0, 0, 0
	}

	// Must start with a digit
	if line[i] < '0' || line[i] > '9' {
		return false, 0, 0, 0
	}

	// Parse the number
	numStart := i
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	numEnd := i

	// Must have at least one digit
	if numEnd == numStart {
		return false, 0, 0, 0
	}

	// Parse the number value
	itemNumber := 0
	for j := numStart; j < numEnd; j++ {
		itemNumber = itemNumber*10 + int(line[j]-'0')
	}

	// Must be followed by '.' or ')'
	if i >= len(line) {
		return false, 0, 0, 0
	}

	delimiter := line[i]
	if delimiter != '.' && delimiter != ')' {
		return false, 0, 0, 0
	}
	i++

	// Delimiter must be followed by a space
	if i >= len(line) || line[i] != ' ' {
		return false, 0, 0, 0
	}

	// Content starts after "N. " or "N) "
	contentStart := i + 1

	return true, indentLevel, contentStart, itemNumber
}

// parseUnorderedListItem parses an unordered list line and returns styled spans.
func parseUnorderedListItem(line string, indentLevel int, contentStart int) []rich.Span {
	return parseUnorderedListItemInternal(line, indentLevel, contentStart, 0, 0, nil, nil)
}

// parseUnorderedListItemInternal parses an unordered list line with optional source mapping.
// It emits: bullet span ("•") + space span + content spans (with inline formatting).
func parseUnorderedListItemInternal(line string, indentLevel, contentStart, sourceOffset, renderedOffset int,
	smEntries *[]SourceMapEntry, lmEntries *[]LinkEntry) []rich.Span {
	var spans []rich.Span

	// Emit the bullet marker (•)
	bulletStyle := rich.Style{
		ListBullet: true,
		ListIndent: indentLevel,
		Scale:      1.0,
	}
	spans = append(spans, rich.Span{
		Text:  "•",
		Style: bulletStyle,
	})

	if smEntries != nil {
		*smEntries = append(*smEntries, SourceMapEntry{
			RenderedStart: renderedOffset,
			RenderedEnd:   renderedOffset + 1,
			SourceStart:   sourceOffset,
			SourceEnd:     sourceOffset + contentStart - 1,
		})
	}
	renderedOffset++

	// Emit the space after bullet
	itemStyle := rich.Style{
		ListItem:   true,
		ListIndent: indentLevel,
		Scale:      1.0,
	}
	spans = append(spans, rich.Span{
		Text:  " ",
		Style: itemStyle,
	})

	if smEntries != nil {
		*smEntries = append(*smEntries, SourceMapEntry{
			RenderedStart: renderedOffset,
			RenderedEnd:   renderedOffset + 1,
			SourceStart:   sourceOffset + contentStart - 1,
			SourceEnd:     sourceOffset + contentStart,
		})
	}
	renderedOffset++

	// Get the content after the marker
	content := ""
	if contentStart < len(line) {
		content = line[contentStart:]
	}

	// If content is empty, we're done
	if content == "" {
		return spans
	}

	// Parse inline formatting in the content, using itemStyle as the base
	contentSpans := parseInline(content, itemStyle, InlineOpts{
		SourceMap:      smEntries,
		LinkMap:        lmEntries,
		SourceOffset:   sourceOffset + contentStart,
		RenderedOffset: renderedOffset,
	})
	spans = append(spans, contentSpans...)

	return spans
}

// parseOrderedListItem parses an ordered list line and returns styled spans.
func parseOrderedListItem(line string, indentLevel int, contentStart int, itemNumber int) []rich.Span {
	return parseOrderedListItemInternal(line, indentLevel, contentStart, itemNumber, 0, 0, nil, nil)
}

// parseOrderedListItemInternal parses an ordered list line with optional source mapping.
// It emits: number span ("N.") + space span + content spans (with inline formatting).
func parseOrderedListItemInternal(line string, indentLevel, contentStart, itemNumber, sourceOffset, renderedOffset int,
	smEntries *[]SourceMapEntry, lmEntries *[]LinkEntry) []rich.Span {
	var spans []rich.Span

	// Emit the number marker (e.g., "1.")
	// Always normalize to "N." format regardless of original delimiter
	bulletStyle := rich.Style{
		ListBullet:  true,
		ListOrdered: true,
		ListNumber:  itemNumber,
		ListIndent:  indentLevel,
		Scale:       1.0,
	}
	numberText := fmt.Sprintf("%d.", itemNumber)
	spans = append(spans, rich.Span{
		Text:  numberText,
		Style: bulletStyle,
	})

	numberLen := len([]rune(numberText))
	if smEntries != nil {
		*smEntries = append(*smEntries, SourceMapEntry{
			RenderedStart: renderedOffset,
			RenderedEnd:   renderedOffset + numberLen,
			SourceStart:   sourceOffset,
			SourceEnd:     sourceOffset + contentStart - 1,
		})
	}
	renderedOffset += numberLen

	// Emit the space after number
	itemStyle := rich.Style{
		ListItem:    true,
		ListOrdered: true,
		ListNumber:  itemNumber,
		ListIndent:  indentLevel,
		Scale:       1.0,
	}
	spans = append(spans, rich.Span{
		Text:  " ",
		Style: itemStyle,
	})

	if smEntries != nil {
		*smEntries = append(*smEntries, SourceMapEntry{
			RenderedStart: renderedOffset,
			RenderedEnd:   renderedOffset + 1,
			SourceStart:   sourceOffset + contentStart - 1,
			SourceEnd:     sourceOffset + contentStart,
		})
	}
	renderedOffset++

	// Get the content after the marker
	content := ""
	if contentStart < len(line) {
		content = line[contentStart:]
	}

	// If content is empty, we're done
	if content == "" {
		return spans
	}

	// Parse inline formatting in the content, using itemStyle as the base
	contentSpans := parseInline(content, itemStyle, InlineOpts{
		SourceMap:      smEntries,
		LinkMap:        lmEntries,
		SourceOffset:   sourceOffset + contentStart,
		RenderedOffset: renderedOffset,
	})
	spans = append(spans, contentSpans...)

	return spans
}

// isUnorderedListItem returns true if line starts with an unordered list marker.
// Returns: (isListItem bool, indentLevel int, contentStart int)
// - isListItem: true if the line is an unordered list item
// - indentLevel: the nesting level (0 = top level, 1 = first nested level, etc.)
// - contentStart: the byte index where the item content begins (after "- ")
//
// Unordered list markers are: -, *, + followed by a space.
// Indentation is counted as: 2 spaces or 1 tab = 1 indent level.
func isUnorderedListItem(line string) (bool, int, int) {
	if len(line) == 0 {
		return false, 0, 0
	}

	// Count leading whitespace and calculate indent level
	// 2 spaces = 1 indent level, 1 tab = 1 indent level
	i := 0
	spaceCount := 0
	tabCount := 0
	for i < len(line) {
		if line[i] == ' ' {
			spaceCount++
			i++
		} else if line[i] == '\t' {
			tabCount++
			i++
		} else {
			break
		}
	}

	// Calculate indent level: each tab counts as 1 level, each 2 spaces counts as 1 level
	indentLevel := tabCount + spaceCount/2

	// After whitespace, check for list marker (-, *, +)
	if i >= len(line) {
		return false, 0, 0
	}

	marker := line[i]
	if marker != '-' && marker != '*' && marker != '+' {
		return false, 0, 0
	}

	// Marker must be followed by a space
	if i+1 >= len(line) || line[i+1] != ' ' {
		return false, 0, 0
	}

	// Check for double markers like "--" which are not list items
	if i+1 < len(line) && line[i+1] == marker {
		return false, 0, 0
	}

	// Content starts after "marker "
	contentStart := i + 2

	return true, indentLevel, contentStart
}

// =============================================================================
// Table Detection Functions (Phase 15B)
// =============================================================================

// isTableRow returns true if line is a table row (starts with |).
// Also returns the cell contents (trimmed) if it is a table row.
func isTableRow(line string) (bool, []string) {
	// Strip trailing newline for comparison
	trimmed := strings.TrimSuffix(line, "\n")
	if trimmed == "" {
		return false, nil
	}

	// Must start with |
	if trimmed[0] != '|' {
		return false, nil
	}

	// Split by | and extract cells
	cells := splitTableCells(trimmed)
	if len(cells) == 0 {
		return false, nil
	}

	return true, cells
}

// splitTableCells splits a table row into cells, trimming whitespace from each.
func splitTableCells(line string) []string {
	// Remove leading and trailing |
	trimmed := strings.TrimPrefix(line, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")

	// Split by |
	parts := strings.Split(trimmed, "|")

	// Trim whitespace from each cell
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = strings.TrimSpace(p)
	}

	return cells
}

// isTableSeparatorRow returns true if the line is a table separator row.
// A separator row contains cells with only dashes (and optional alignment colons).
func isTableSeparatorRow(line string) bool {
	isSep, _ := parseTableSeparator(line)
	return isSep
}

// parseTableSeparator parses a table separator row and returns the alignment for each column.
func parseTableSeparator(line string) (bool, []rich.Alignment) {
	// Strip trailing newline for comparison
	trimmed := strings.TrimSuffix(line, "\n")
	if trimmed == "" {
		return false, nil
	}

	// Must start with |
	if trimmed[0] != '|' {
		return false, nil
	}

	cells := splitTableCells(trimmed)
	if len(cells) == 0 {
		return false, nil
	}

	aligns := make([]rich.Alignment, len(cells))
	for i, cell := range cells {
		cell = strings.TrimSpace(cell)
		align, ok := parseSeparatorCell(cell)
		if !ok {
			return false, nil
		}
		aligns[i] = align
	}

	return true, aligns
}

// parseSeparatorCell checks if a cell is a valid separator cell (dashes with optional alignment colons).
// Returns the alignment and whether it's valid.
func parseSeparatorCell(cell string) (rich.Alignment, bool) {
	// Minimum valid separator cell is "---" (3 chars) or ":--" / "--:" (also 3 chars)
	if len(cell) < 3 {
		return rich.AlignLeft, false
	}

	// Check for alignment markers
	hasLeftColon := strings.HasPrefix(cell, ":")
	hasRightColon := strings.HasSuffix(cell, ":")

	// Remove colons for dash check
	inner := cell
	if hasLeftColon {
		inner = inner[1:]
	}
	if hasRightColon && len(inner) > 0 {
		inner = inner[:len(inner)-1]
	}

	// Must have at least 1 dash (after removing colons)
	// CommonMark requires at least 1 dash; we require 1 for compatibility
	if len(inner) < 1 {
		return rich.AlignLeft, false
	}

	// Rest must be all dashes
	for _, c := range inner {
		if c != '-' {
			return rich.AlignLeft, false
		}
	}

	// Determine alignment
	if hasLeftColon && hasRightColon {
		return rich.AlignCenter, true
	}
	if hasRightColon {
		return rich.AlignRight, true
	}
	// Default is left (including explicit left with just leading colon)
	return rich.AlignLeft, true
}

// calculateColumnWidths calculates the maximum width for each column.
// Width is measured in runes (not bytes) for correct handling of multi-byte UTF-8 characters.
func calculateColumnWidths(rows [][]string) []int {
	if len(rows) == 0 {
		return nil
	}

	// Find the number of columns from the first row
	numCols := len(rows[0])
	widths := make([]int, numCols)

	for _, row := range rows {
		for i, cell := range row {
			if i < numCols {
				w := utf8.RuneCountInString(cell)
				if w > widths[i] {
					widths[i] = w
				}
			}
		}
	}

	return widths
}

// padCell pads a cell's content to the given width according to the alignment.
// AlignLeft: content + trailing spaces
// AlignRight: leading spaces + content
// AlignCenter: balanced padding (extra space on right if odd)
// Width is measured in runes (not bytes) for correct handling of multi-byte UTF-8 characters.
func padCell(content string, width int, align rich.Alignment) string {
	padding := width - utf8.RuneCountInString(content)
	if padding <= 0 {
		return content
	}
	switch align {
	case rich.AlignRight:
		return strings.Repeat(" ", padding) + content
	case rich.AlignCenter:
		left := padding / 2
		right := padding - left
		return strings.Repeat(" ", left) + content + strings.Repeat(" ", right)
	default: // AlignLeft
		return content + strings.Repeat(" ", padding)
	}
}

// rebuildTableRow rebuilds a table row line from cells, column widths, and alignments.
func rebuildTableRow(cells []string, widths []int, aligns []rich.Alignment) string {
	var b strings.Builder
	b.WriteByte('|')
	for j, cell := range cells {
		w := 0
		if j < len(widths) {
			w = widths[j]
		}
		a := rich.AlignLeft
		if j < len(aligns) {
			a = aligns[j]
		}
		b.WriteString(" " + padCell(cell, w, a) + " |")
	}
	return b.String()
}

// rebuildSeparatorRow rebuilds the separator line with dashes padded to column widths.
func rebuildSeparatorRow(widths []int, aligns []rich.Alignment) string {
	var b strings.Builder
	b.WriteByte('|')
	for j, w := range widths {
		a := rich.AlignLeft
		if j < len(aligns) {
			a = aligns[j]
		}
		b.WriteByte(' ')
		switch a {
		case rich.AlignCenter:
			b.WriteByte(':')
			b.WriteString(strings.Repeat("-", w-2))
			b.WriteByte(':')
		case rich.AlignRight:
			b.WriteString(strings.Repeat("-", w-1))
			b.WriteByte(':')
		default: // AlignLeft (with or without leading colon)
			b.WriteString(strings.Repeat("-", w))
		}
		b.WriteString(" |")
	}
	return b.String()
}

// parseTableBlock parses a table starting at the given line index.
// Returns the spans for the table and the number of lines consumed.
func parseTableBlock(lines []string, startIdx int) ([]rich.Span, int) {
	return parseTableBlockInternal(lines, startIdx, 0, 0, nil)
}

// parseTableBlockInternal parses a table starting at the given line index,
// optionally accumulating source map entries when smEntries is non-nil.
// Returns the spans for the table, source map entries, and the number of lines consumed.
func parseTableBlockInternal(lines []string, startIdx int, sourceOffset, renderedOffset int,
	smEntries *[]SourceMapEntry) ([]rich.Span, int) {
	if startIdx >= len(lines) {
		return nil, 0
	}

	// First line should be header row
	isRow, _ := isTableRow(lines[startIdx])
	if !isRow {
		return nil, 0
	}

	// Second line should be separator row
	if startIdx+1 >= len(lines) || !isTableSeparatorRow(lines[startIdx+1]) {
		return nil, 0
	}

	tracking := smEntries != nil

	// Collect all table lines (header, separator, and data rows)
	var tableLines []string
	consumed := 0

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		isTableLine, _ := isTableRow(line)
		isSep := isTableSeparatorRow(line)

		// A line is part of the table if it's a table row or separator
		if isTableLine || isSep {
			tableLines = append(tableLines, line)
			consumed++
		} else {
			// Non-table line ends the table
			break
		}
	}

	if consumed < 2 {
		// Need at least header + separator
		return nil, 0
	}

	// Parse alignment from separator row (line index 1)
	_, aligns := parseTableSeparator(tableLines[1])

	// Parse all rows into cells (skip separator at index 1)
	var allCells [][]string
	for i, line := range tableLines {
		if i == 1 {
			// Separator row — skip for column width calculation
			allCells = append(allCells, nil)
			continue
		}
		_, cells := isTableRow(line)
		allCells = append(allCells, cells)
	}

	// Collect non-nil rows for width calculation
	var dataCells [][]string
	for _, cells := range allCells {
		if cells != nil {
			dataCells = append(dataCells, cells)
		}
	}

	// Calculate column widths
	widths := calculateColumnWidths(dataCells)

	// Ensure minimum width of 3 for separator dashes
	for i, w := range widths {
		if w < 3 {
			widths[i] = 3
		}
	}

	// Build box-drawing grid lines
	topBorder := buildGridLine(widths, '┌', '┬', '┐', '─')
	headerSep := buildGridLine(widths, '├', '┼', '┤', '─')
	bottomBorder := buildGridLine(widths, '└', '┴', '┘', '─')

	// Style for border/separator lines (not header, not bold)
	borderStyle := rich.Style{
		Table: true,
		Code:  true,
		Block: true,
		Bg:    rich.InlineCodeBg,
		Scale: 1.0,
	}

	var spans []rich.Span
	srcPos := sourceOffset
	rendPos := renderedOffset

	// Top border
	topText := topBorder + "\n"
	topLen := len([]rune(topText))
	spans = append(spans, rich.Span{
		Text:  topText,
		Style: borderStyle,
	})
	if tracking {
		*smEntries = append(*smEntries, SourceMapEntry{
			RenderedStart: rendPos,
			RenderedEnd:   rendPos + topLen,
			SourceStart:   srcPos,
			SourceEnd:     srcPos, // zero-length: synthetic line
			Kind:          KindSynthetic,
		})
	}
	rendPos += topLen

	for i, line := range tableLines {
		isHeader := i == 0
		isSeparator := i == 1

		if isSeparator {
			// Replace ASCII separator with box-drawing header separator
			sepText := headerSep + "\n"
			sepLen := len([]rune(sepText))
			spans = append(spans, rich.Span{
				Text:  sepText,
				Style: borderStyle,
			})
			if tracking {
				*smEntries = append(*smEntries, SourceMapEntry{
					RenderedStart: rendPos,
					RenderedEnd:   rendPos + sepLen,
					SourceStart:   srcPos,
					SourceEnd:     srcPos + len(line),
					Kind:          KindSynthetic,
				})
			}
			rendPos += sepLen
			srcPos += len(line)
		} else {
			cells := allCells[i]
			lineText := replaceDelimiters(rebuildTableRow(cells, widths, aligns))
			lineText += "\n" // all rows get newline since bottom border follows

			style := rich.Style{
				Table:       true,
				TableHeader: isHeader,
				Code:        true,
				Block:       true,
				Bg:          rich.InlineCodeBg,
				Scale:       1.0,
			}
			if isHeader {
				style.Bold = true
			}

			spans = append(spans, rich.Span{
				Text:  lineText,
				Style: style,
			})

			// Per-cell source map entries
			if tracking {
				cellPositions := splitTableCellPositions(line)
				for j, cp := range cellPositions {
					if j >= len(widths) {
						break
					}

					// Compute rendered rune position of cell j's content.
					chunkStart := 1
					for k := 0; k < j; k++ {
						chunkStart += widths[k] + 3
					}

					// Content offset within the padded area depends on alignment
					a := rich.AlignLeft
					if j < len(aligns) {
						a = aligns[j]
					}
					contentRuneLen := len([]rune(cp.Content))
					var contentOffset int
					switch a {
					case rich.AlignRight:
						contentOffset = widths[j] - contentRuneLen
					case rich.AlignCenter:
						contentOffset = (widths[j] - contentRuneLen) / 2
					default: // AlignLeft
						contentOffset = 0
					}

					rendCellStart := rendPos + chunkStart + 1 + contentOffset
					rendCellEnd := rendCellStart + contentRuneLen

					if contentRuneLen == 0 {
						rendCellEnd = rendCellStart
					}

					*smEntries = append(*smEntries, SourceMapEntry{
						RenderedStart: rendCellStart,
						RenderedEnd:   rendCellEnd,
						SourceStart:   srcPos + cp.ByteStart,
						SourceEnd:     srcPos + cp.ByteEnd,
						Kind:          KindTableCell,
						CellBorderPos: rendPos + chunkStart - 1,
					})
				}
			}

			rendPos += len([]rune(lineText))
			srcPos += len(line)
		}
	}

	// Bottom border with trailing newline
	bottomText := bottomBorder + "\n"
	bottomLen := len([]rune(bottomText))
	spans = append(spans, rich.Span{
		Text:  bottomText,
		Style: borderStyle,
	})
	if tracking {
		*smEntries = append(*smEntries, SourceMapEntry{
			RenderedStart: rendPos,
			RenderedEnd:   rendPos + bottomLen,
			SourceStart:   srcPos,
			SourceEnd:     srcPos, // zero-length: synthetic line
			Kind:          KindSynthetic,
		})
	}

	return spans, consumed
}

// buildGridLine builds a box-drawing horizontal line from column widths.
// left/mid/right are corner/tee characters; fill is the horizontal rule char.
// Each column segment is fill×(width+2) to account for the padding spaces around cell content.
func buildGridLine(widths []int, left, mid, right, fill rune) string {
	var b strings.Builder
	b.WriteRune(left)
	for i, w := range widths {
		for j := 0; j < w+2; j++ { // +2 for padding spaces
			b.WriteRune(fill)
		}
		if i < len(widths)-1 {
			b.WriteRune(mid)
		}
	}
	b.WriteRune(right)
	return b.String()
}

// cellPosition describes the rune/byte position of a cell's trimmed content
// within a table row source line.
type cellPosition struct {
	Content   string
	RuneStart int // rune offset of content within the line
	RuneEnd   int // rune offset of content end within the line
	ByteStart int // byte offset of content within the line
	ByteEnd   int // byte offset of content end within the line
}

// splitTableCellPositions walks a table row source line and returns the
// position of each cell's trimmed content. It splits on '|' delimiters,
// and for each segment between pipes, finds the trimmed content's start/end
// positions in both rune and byte offsets.
func splitTableCellPositions(line string) []cellPosition {
	// Strip trailing newline for position tracking
	trimmed := strings.TrimSuffix(line, "\n")

	// We need to iterate rune-by-rune, splitting on '|' and tracking positions.
	runes := []rune(trimmed)
	var cells []cellPosition

	// Skip leading '|'
	idx := 0
	if idx < len(runes) && runes[idx] == '|' {
		idx++
	}

	// Process segments between pipes
	for idx < len(runes) {
		// Find the next '|' or end of line
		segStart := idx
		for idx < len(runes) && runes[idx] != '|' {
			idx++
		}
		segEnd := idx

		// Skip the closing '|' if present
		if idx < len(runes) && runes[idx] == '|' {
			idx++
		}

		// The segment is runes[segStart:segEnd]
		// Find the trimmed content within this segment
		contentStart := segStart
		contentEnd := segEnd

		// Trim leading spaces
		for contentStart < contentEnd && runes[contentStart] == ' ' {
			contentStart++
		}
		// Trim trailing spaces
		for contentEnd > contentStart && runes[contentEnd-1] == ' ' {
			contentEnd--
		}

		// Convert rune positions to byte positions
		byteStart := len(string(runes[:contentStart]))
		byteEnd := len(string(runes[:contentEnd]))

		content := string(runes[contentStart:contentEnd])
		cells = append(cells, cellPosition{
			Content:   content,
			RuneStart: contentStart,
			RuneEnd:   contentEnd,
			ByteStart: byteStart,
			ByteEnd:   byteEnd,
		})
	}

	return cells
}

// replaceDelimiters replaces ASCII pipe '|' delimiters with box-drawing '│' (U+2502)
// in a table row string built by rebuildTableRow.
func replaceDelimiters(row string) string {
	return strings.ReplaceAll(row, "|", "│")
}
