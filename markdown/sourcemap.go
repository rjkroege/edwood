package markdown

import (
	"fmt"
	"strings"

	"github.com/rjkroege/edwood/rich"
)

// SourceMap maps positions in rendered content back to positions in source markdown.
type SourceMap struct {
	entries []SourceMapEntry
}

// SourceMapEntry maps a range in rendered content to a range in source markdown.
type SourceMapEntry struct {
	RenderedStart   int // Rune position in rendered content
	RenderedEnd     int
	SourceStart     int // Byte position in source markdown
	SourceEnd       int
	SourceRuneStart int // Rune position in source markdown
	SourceRuneEnd   int
	PrefixLen       int // Length of source prefix not in rendered (e.g., "# " for headings)
}

// ToSource maps a range in rendered content (renderedStart, renderedEnd) to
// the corresponding range in the source markdown as RUNE positions.
// This is used by syncSourceSelection to set body.q0/q1 which expect rune positions.
// When the selection spans formatted elements, it expands to include the full
// source markup (e.g., selecting "bold" in "**bold**" returns the rune range of "**bold**").
func (sm *SourceMap) ToSource(renderedStart, renderedEnd int) (srcStart, srcEnd int) {
	if len(sm.entries) == 0 {
		return renderedStart, renderedEnd
	}

	// Find the entry containing renderedStart
	srcStart = -1
	var startEntry *SourceMapEntry
	for i := range sm.entries {
		e := &sm.entries[i]
		if renderedStart >= e.RenderedStart && renderedStart < e.RenderedEnd {
			startEntry = e
			break
		}
	}

	if startEntry == nil {
		// Position falls in a gap between entries — find nearest entry after.
		for i := range sm.entries {
			e := &sm.entries[i]
			if e.RenderedStart >= renderedStart {
				srcStart = e.SourceRuneStart
				break
			}
		}
		if srcStart == -1 {
			srcStart = renderedStart
		}
	} else {
		// Use the unified formula: map rendered position to source content
		// position past the opening marker, then apply boundary expansion.
		offset := renderedStart - startEntry.RenderedStart
		srcStart = startEntry.SourceRuneStart + entryOpeningLen(startEntry) + offset

		// Boundary expansion: if selection starts at entry start (range selection),
		// include opening markup.
		if renderedStart == startEntry.RenderedStart && renderedStart != renderedEnd {
			srcStart = startEntry.SourceRuneStart
		}
	}

	// Find the entry containing renderedEnd-1 (or handle empty/edge cases)
	srcEnd = -1
	var endEntry *SourceMapEntry
	lookupPos := renderedEnd
	if renderedEnd > renderedStart {
		lookupPos = renderedEnd - 1
	}
	for i := range sm.entries {
		e := &sm.entries[i]
		if lookupPos >= e.RenderedStart && lookupPos < e.RenderedEnd {
			endEntry = e
			break
		}
	}

	if endEntry == nil {
		// Position falls in a gap — find nearest entry before renderedEnd.
		// Map through the gap using 1:1 offset from the entry's source end,
		// which handles unmapped characters like synthetic paragraph-break newlines.
		for i := len(sm.entries) - 1; i >= 0; i-- {
			if sm.entries[i].RenderedEnd <= renderedEnd {
				endEntry = &sm.entries[i]
				break
			}
		}
		if endEntry != nil {
			srcEnd = endEntry.SourceRuneEnd + (renderedEnd - endEntry.RenderedEnd)
		} else {
			srcEnd = renderedEnd
		}
	} else {
		// Boundary expansion: if selection ends at entry end (range selection),
		// include closing markup.
		if renderedEnd == endEntry.RenderedEnd {
			srcEnd = endEntry.SourceRuneEnd
		} else {
			// Use the unified formula: opening marker length + content offset
			offset := renderedEnd - endEntry.RenderedStart
			srcEnd = endEntry.SourceRuneStart + entryOpeningLen(endEntry) + offset
		}
	}

	// A point selection in rendered content must map to a point in source.
	// With the unified formula, start and end should agree for point selections.
	// Keep as safety normalization.
	if renderedStart == renderedEnd && srcStart != srcEnd {
		srcStart = srcEnd
	}

	return srcStart, srcEnd
}

// entryOpeningLen computes the rune length of the opening marker for a source
// map entry. For heading-style entries (PrefixLen > 0), this is PrefixLen.
// For symmetric markers (bold, italic, code), it is half the extra source runes.
// For 1:1 entries (plain text) or synthetic entries (table borders with zero-length
// source), returns 0.
func entryOpeningLen(e *SourceMapEntry) int {
	if e.PrefixLen > 0 {
		return e.PrefixLen
	}
	renderedLen := e.RenderedEnd - e.RenderedStart
	sourceRuneLen := e.SourceRuneEnd - e.SourceRuneStart
	if sourceRuneLen <= renderedLen {
		// 1:1 mapping (plain text) or synthetic entry (source shorter than rendered)
		return 0
	}
	extra := sourceRuneLen - renderedLen
	return extra / 2
}

// ToRendered maps a range in source markdown (srcRuneStart, srcRuneEnd as rune positions)
// to the corresponding range in the rendered content (as rune positions).
// Returns (-1, -1) if no mapping exists.
// This is the inverse of ToSource(): given source positions (e.g., from search()),
// find where that content appears in the rendered preview.
func (sm *SourceMap) ToRendered(srcRuneStart, srcRuneEnd int) (renderedStart, renderedEnd int) {
	if len(sm.entries) == 0 {
		return -1, -1
	}

	// Find the entry containing srcRuneStart
	renderedStart = -1
	for i := range sm.entries {
		e := &sm.entries[i]
		if srcRuneStart >= e.SourceRuneStart && srcRuneStart < e.SourceRuneEnd {
			renderedStart = sm.sourceRuneToRendered(e, srcRuneStart)
			break
		}
	}

	if renderedStart == -1 {
		return -1, -1
	}

	// Find the entry containing srcRuneEnd-1 (or handle edge cases)
	renderedEnd = -1
	lookupPos := srcRuneEnd
	if srcRuneEnd > srcRuneStart {
		lookupPos = srcRuneEnd - 1
	}
	for i := range sm.entries {
		e := &sm.entries[i]
		if lookupPos >= e.SourceRuneStart && lookupPos < e.SourceRuneEnd {
			// For end position, if srcRuneEnd is at or past the entry end,
			// map to the full rendered end
			if srcRuneEnd >= e.SourceRuneEnd {
				renderedEnd = e.RenderedEnd
			} else {
				renderedEnd = sm.sourceRuneToRendered(e, srcRuneEnd)
			}
			break
		}
	}

	if renderedEnd == -1 {
		return -1, -1
	}

	return renderedStart, renderedEnd
}

// sourceRuneToRendered maps a single source rune position to a rendered position
// within a given entry. For 1:1 entries (plain text), the offset is direct.
// For formatted entries (bold, italic, heading, code), the source contains
// opening and closing markers around the rendered content.
func (sm *SourceMap) sourceRuneToRendered(e *SourceMapEntry, srcRunePos int) int {
	offset := srcRunePos - e.SourceRuneStart
	renderedLen := e.RenderedEnd - e.RenderedStart
	sourceRuneLen := e.SourceRuneEnd - e.SourceRuneStart

	if renderedLen == sourceRuneLen {
		// 1:1 mapping (plain text, code block content, list content, etc.)
		return e.RenderedStart + offset
	}

	// Formatted element: source has markers around rendered content.
	// Compute the opening marker length in runes.
	// For headings: PrefixLen is the byte length of "# " etc.; since markers are ASCII, bytes=runes.
	// For symmetric markers (**bold**, *italic*, `code`, ***bi***): opening = closing = (extra / 2).
	var openingLen int
	if e.PrefixLen > 0 {
		// Heading-style prefix (e.g., "# ", "## ")
		openingLen = e.PrefixLen
	} else {
		// Symmetric markers (**, *, `, ***)
		extra := sourceRuneLen - renderedLen
		openingLen = extra / 2
	}

	if offset <= openingLen {
		// Within or at the opening marker - map to rendered start
		return e.RenderedStart
	}

	contentOffset := offset - openingLen
	if contentOffset >= renderedLen {
		// Within or at the closing marker - map to rendered end
		return e.RenderedEnd
	}

	return e.RenderedStart + contentOffset
}

// ParseWithSourceMap parses markdown and returns the styled content,
// a source map for mapping rendered positions back to source positions,
// and a link map for tracking which rendered positions contain links.
func ParseWithSourceMap(text string) (rich.Content, *SourceMap, *LinkMap) {
	if text == "" {
		return rich.Content{}, &SourceMap{}, NewLinkMap()
	}

	var result rich.Content
	sm := &SourceMap{}
	lm := NewLinkMap()
	lines := splitLines(text)

	sourcePos := 0   // Current position in source
	renderedPos := 0 // Current position in rendered content

	// Track fenced code block state
	inFencedBlock := false
	var codeBlockContent strings.Builder
	codeBlockSourceStart := 0 // Source position where code content starts (after opening fence line)

	// Track indented code block state
	inIndentedBlock := false
	var indentedBlockContent strings.Builder
	indentedBlockSourceStart := 0

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

			// Create source map entry
			codeLen := len([]rune(codeContent))
			entry := SourceMapEntry{
				RenderedStart: renderedPos,
				RenderedEnd:   renderedPos + codeLen,
				SourceStart:   indentedBlockSourceStart,
				SourceEnd:     sourcePos,
			}
			sm.entries = append(sm.entries, entry)
			renderedPos += codeLen

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
		// Check for fenced code block delimiter
		if isFenceDelimiter(line) {
			// If we were in an indented block, emit it first
			if inIndentedBlock {
				emitIndentedBlock()
			}
			// End paragraph with newline before code block
			if inParagraph && len(result) > 0 {
				result[len(result)-1].Text += "\n"
				renderedPos++
			}
			inParagraph = false
			if !inFencedBlock {
				// Opening fence - start collecting code
				inFencedBlock = true
				codeBlockContent.Reset()
				// Skip past the fence line in source, remember where code content starts
				sourcePos += len(line)
				codeBlockSourceStart = sourcePos
				continue
			} else {
				// Closing fence - emit the code block
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

					// Create source map entry for the code content
					// Maps rendered code to source code (excluding fence lines)
					codeLen := len([]rune(codeContent))
					entry := SourceMapEntry{
						RenderedStart: renderedPos,
						RenderedEnd:   renderedPos + codeLen,
						SourceStart:   codeBlockSourceStart,
						SourceEnd:     sourcePos, // sourcePos is at start of closing fence
					}
					sm.entries = append(sm.entries, entry)
					renderedPos += codeLen

					// Merge or append the code span
					if len(result) > 0 && result[len(result)-1].Style == codeSpan.Style {
						result[len(result)-1].Text += codeSpan.Text
					} else {
						result = append(result, codeSpan)
					}
				}
				// Skip past the closing fence line in source
				sourcePos += len(line)
				continue
			}
		}

		if inFencedBlock {
			// Inside fenced block - collect raw content without parsing
			codeBlockContent.WriteString(line)
			sourcePos += len(line)
			continue
		}

		// Check for list items BEFORE checking for indented code blocks
		// This ensures deeply nested list items (with 4+ spaces or tabs) are recognized
		isULEarly, _, _ := isUnorderedListItem(line)
		isOLEarly, _, _, _ := isOrderedListItem(line)
		isListItemEarly := isULEarly || isOLEarly

		// Check for indented code block (4 spaces or 1 tab)
		// But NOT if it's a list item - list items take precedence
		if isIndentedCodeLine(line) && !isListItemEarly {
			// End paragraph with newline before code block
			if inParagraph && len(result) > 0 {
				result[len(result)-1].Text += "\n"
				renderedPos++
			}
			inParagraph = false
			if !inIndentedBlock {
				// Start of indented block
				inIndentedBlock = true
				indentedBlockSourceStart = sourcePos
				indentedBlockContent.Reset()
			}
			// Add the line content (with indent stripped)
			indentedBlockContent.WriteString(stripIndent(line))
			sourcePos += len(line)
			continue
		}

		// Not an indented line - if we were in an indented block, emit it
		if inIndentedBlock {
			emitIndentedBlock()
		}

		// Check for blank line (paragraph break)
		trimmedLine := strings.TrimRight(line, "\n")
		if trimmedLine == "" {
			// Blank line = paragraph break
			if inParagraph && len(result) > 0 {
				// End the paragraph with a newline (with ParaBreak for extra spacing)
				result = append(result, rich.Span{
					Text:  "\n",
					Style: rich.Style{ParaBreak: true, Scale: 1.0},
				})
				renderedPos++
			}
			inParagraph = false
			sourcePos += len(line)
			continue
		}

		// Check for table (must have header row followed by separator row)
		isRow, _ := isTableRow(line)
		if isRow && i+1 < len(lines) && isTableSeparatorRow(lines[i+1]) {
			// End paragraph before table
			if inParagraph && len(result) > 0 {
				result[len(result)-1].Text += "\n"
				renderedPos++
			}
			inParagraph = false

			// Parse the table - collect all consecutive table rows
			tableSpans, tableEntries, consumed := parseTableBlockWithSourceMap(lines, i, sourcePos, renderedPos)
			result = append(result, tableSpans...)
			sm.entries = append(sm.entries, tableEntries...)

			// Update positions based on consumed lines
			for j := 0; j < consumed; j++ {
				sourcePos += len(lines[i+j])
			}
			for _, span := range tableSpans {
				renderedPos += len([]rune(span.Text))
			}
			i += consumed - 1 // -1 because loop will increment
			continue
		}

		// Check if this is a block-level element (heading, hrule, list item)
		isUL, _, _ := isUnorderedListItem(line)
		isOL, _, _, _ := isOrderedListItem(line)
		isListItem := isUL || isOL
		isBlockElement := headingLevel(line) > 0 || isHorizontalRule(line) || isListItem

		if isBlockElement {
			// Block elements start fresh - end previous paragraph with newline
			if inParagraph && len(result) > 0 {
				result[len(result)-1].Text += "\n"
				renderedPos++
			}
			inParagraph = false
		} else {
			// Regular text line - join with previous paragraph text
			if inParagraph && len(result) > 0 {
				// Add space to end of last span for paragraph continuation
				lastSpan := &result[len(result)-1]
				if strings.HasSuffix(lastSpan.Text, "\n") {
					lastSpan.Text = strings.TrimSuffix(lastSpan.Text, "\n") + " "
				} else if !strings.HasSuffix(lastSpan.Text, " ") {
					lastSpan.Text += " "
					renderedPos++
				}
			}
			inParagraph = true
		}

		// For regular text, strip trailing newline (paragraph text is joined with spaces)
		lineToPass := line
		if !isBlockElement {
			lineToPass = strings.TrimSuffix(line, "\n")
		}

		// Normal line parsing
		spans, entries, linkEntries := parseLineWithSourceMap(lineToPass, sourcePos, renderedPos)
		sm.entries = append(sm.entries, entries...)
		for _, le := range linkEntries {
			lm.Add(le.Start, le.End, le.URL)
		}

		// Update rendered position based on spans
		for _, span := range spans {
			renderedPos += len([]rune(span.Text))
		}

		// Update source position (use original line length, not stripped)
		sourcePos += len(line)

		// Merge consecutive spans with the same style
		for _, span := range spans {
			if len(result) > 0 && result[len(result)-1].Style == span.Style {
				result[len(result)-1].Text += span.Text
			} else {
				result = append(result, span)
			}
		}
	}

	// Handle unclosed fenced code block - treat remaining content as code
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

			// Create source map entry
			codeLen := len([]rune(codeContent))
			entry := SourceMapEntry{
				RenderedStart: renderedPos,
				RenderedEnd:   renderedPos + codeLen,
				SourceStart:   codeBlockSourceStart,
				SourceEnd:     sourcePos,
			}
			sm.entries = append(sm.entries, entry)
			renderedPos += codeLen

			result = append(result, codeSpan)
		}
	}

	// Handle trailing indented code block
	if inIndentedBlock {
		emitIndentedBlock()
	}

	// Post-process: populate SourceRuneStart/SourceRuneEnd from byte positions.
	// Build a byte-to-rune mapping table for the source text.
	sm.populateRunePositions(text)

	return result, sm, lm
}

// populateRunePositions fills in SourceRuneStart/SourceRuneEnd for all entries
// by converting byte positions to rune positions using the source text.
func (sm *SourceMap) populateRunePositions(source string) {
	if len(sm.entries) == 0 {
		return
	}

	sourceLen := len(source)

	// Build byte-to-rune position lookup table.
	// b2r[byteOffset] = runeOffset for each valid byte boundary.
	b2r := make([]int, sourceLen+1)
	runePos := 0
	bi := 0
	for _, r := range source {
		b2r[bi] = runePos
		bi += len(string(r))
		runePos++
	}
	b2r[sourceLen] = runePos

	for i := range sm.entries {
		e := &sm.entries[i]
		start := e.SourceStart
		end := e.SourceEnd
		if start < 0 {
			start = 0
		}
		if start > sourceLen {
			start = sourceLen
		}
		if end < 0 {
			end = 0
		}
		if end > sourceLen {
			end = sourceLen
		}
		e.SourceRuneStart = b2r[start]
		e.SourceRuneEnd = b2r[end]
	}
}

// parseLineWithSourceMap parses a single line and returns spans, source map entries, and link entries.
func parseLineWithSourceMap(line string, sourceOffset, renderedOffset int) ([]rich.Span, []SourceMapEntry, []LinkEntry) {
	// Check for horizontal rule (---, ***, ___)
	if isHorizontalRule(line) {
		// Emit the HRuleRune marker plus newline if the original line had one
		text := string(rich.HRuleRune)
		hasNewline := strings.HasSuffix(line, "\n")
		if hasNewline {
			text += "\n"
		}

		var entries []SourceMapEntry

		// Source line without newline (e.g., "---" from "---\n")
		sourceWithoutNewline := strings.TrimSuffix(line, "\n")

		// Entry for HRuleRune maps to the hrule characters (without newline)
		entries = append(entries, SourceMapEntry{
			RenderedStart: renderedOffset,
			RenderedEnd:   renderedOffset + 1, // Just HRuleRune
			SourceStart:   sourceOffset,
			SourceEnd:     sourceOffset + len(sourceWithoutNewline),
		})

		// If there's a newline, add a separate entry for it
		if hasNewline {
			entries = append(entries, SourceMapEntry{
				RenderedStart: renderedOffset + 1,
				RenderedEnd:   renderedOffset + 2, // The newline
				SourceStart:   sourceOffset + len(sourceWithoutNewline),
				SourceEnd:     sourceOffset + len(line),
			})
		}

		span := rich.Span{
			Text:  text,
			Style: rich.StyleHRule,
		}
		return []rich.Span{span}, entries, nil
	}

	// Check for heading (# at start of line)
	level := headingLevel(line)
	if level > 0 {
		// Extract heading text (strip # prefix and leading space)
		prefixLen := level
		content := line[level:]
		if len(content) > 0 && content[0] == ' ' {
			content = content[1:]
			prefixLen++ // Include the space in prefix
		}

		renderedLen := len([]rune(content))
		entry := SourceMapEntry{
			RenderedStart: renderedOffset,
			RenderedEnd:   renderedOffset + renderedLen,
			SourceStart:   sourceOffset,
			SourceEnd:     sourceOffset + len(line),
			PrefixLen:     prefixLen,
		}

		span := rich.Span{
			Text: content,
			Style: rich.Style{
				Bold:  true,
				Scale: headingScales[level],
			},
		}
		return []rich.Span{span}, []SourceMapEntry{entry}, nil
	}

	// Check for unordered list item (-, *, +)
	if isUL, indentLevel, contentStart := isUnorderedListItem(line); isUL {
		return parseUnorderedListItemWithSourceMap(line, indentLevel, contentStart, sourceOffset, renderedOffset)
	}

	// Check for ordered list item (1., 2), etc.)
	if isOL, indentLevel, contentStart, itemNumber := isOrderedListItem(line); isOL {
		return parseOrderedListItemWithSourceMap(line, indentLevel, contentStart, itemNumber, sourceOffset, renderedOffset)
	}

	// Parse inline formatting
	var entries []SourceMapEntry
	var linkEntries []LinkEntry
	spans := parseInline(line, rich.DefaultStyle(), InlineOpts{
		SourceMap:      &entries,
		LinkMap:        &linkEntries,
		SourceOffset:   sourceOffset,
		RenderedOffset: renderedOffset,
	})
	return spans, entries, linkEntries
}

// parseUnorderedListItemWithSourceMap parses an unordered list line with source mapping.
// It emits: bullet span ("•") + space span + content spans (with inline formatting).
// The source map maps the entire rendered line (including bullet) back to the source
// line (including leading whitespace and marker).
func parseUnorderedListItemWithSourceMap(line string, indentLevel, contentStart, sourceOffset, renderedOffset int) ([]rich.Span, []SourceMapEntry, []LinkEntry) {
	var spans []rich.Span
	var entries []SourceMapEntry
	var linkEntries []LinkEntry

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

	// Create source map entry for bullet: maps "•" to leading whitespace + marker
	// Source is: leading whitespace + "- " (contentStart bytes)
	// Rendered is: "•" (1 rune)
	entries = append(entries, SourceMapEntry{
		RenderedStart: renderedOffset,
		RenderedEnd:   renderedOffset + 1, // "•" is 1 rune
		SourceStart:   sourceOffset,
		SourceEnd:     sourceOffset + contentStart - 1, // everything up to space after marker
	})
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

	// Source map for space
	entries = append(entries, SourceMapEntry{
		RenderedStart: renderedOffset,
		RenderedEnd:   renderedOffset + 1,
		SourceStart:   sourceOffset + contentStart - 1, // space in source
		SourceEnd:     sourceOffset + contentStart,
	})
	renderedOffset++

	// Get the content after the marker
	content := ""
	if contentStart < len(line) {
		content = line[contentStart:]
	}

	// If content is empty, we're done
	if content == "" {
		return spans, entries, linkEntries
	}

	// Parse inline formatting in the content, using itemStyle as the base
	contentSpans := parseInline(content, itemStyle, InlineOpts{
		SourceMap:      &entries,
		LinkMap:        &linkEntries,
		SourceOffset:   sourceOffset + contentStart,
		RenderedOffset: renderedOffset,
	})
	spans = append(spans, contentSpans...)

	return spans, entries, linkEntries
}

// parseOrderedListItemWithSourceMap parses an ordered list line with source mapping.
// It emits: number span ("N.") + space span + content spans (with inline formatting).
func parseOrderedListItemWithSourceMap(line string, indentLevel, contentStart, itemNumber, sourceOffset, renderedOffset int) ([]rich.Span, []SourceMapEntry, []LinkEntry) {
	var spans []rich.Span
	var entries []SourceMapEntry
	var linkEntries []LinkEntry

	// Emit the number marker (e.g., "1.")
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

	// Calculate rendered length of number (e.g., "1." = 2, "10." = 3)
	numberLen := len([]rune(numberText))

	// Create source map entry for number: maps "N." to leading whitespace + marker
	// Source is: leading whitespace + "N." or "N)" (contentStart - 1 bytes, minus the space)
	entries = append(entries, SourceMapEntry{
		RenderedStart: renderedOffset,
		RenderedEnd:   renderedOffset + numberLen,
		SourceStart:   sourceOffset,
		SourceEnd:     sourceOffset + contentStart - 1, // everything up to space after marker
	})
	renderedOffset += numberLen

	// Emit the space after number
	itemStyle := rich.Style{
		ListItem:   true,
		ListIndent: indentLevel,
		Scale:      1.0,
	}
	spans = append(spans, rich.Span{
		Text:  " ",
		Style: itemStyle,
	})

	// Source map for space
	entries = append(entries, SourceMapEntry{
		RenderedStart: renderedOffset,
		RenderedEnd:   renderedOffset + 1,
		SourceStart:   sourceOffset + contentStart - 1,
		SourceEnd:     sourceOffset + contentStart,
	})
	renderedOffset++

	// Get the content after the marker
	content := ""
	if contentStart < len(line) {
		content = line[contentStart:]
	}

	// If content is empty, we're done
	if content == "" {
		return spans, entries, linkEntries
	}

	// Parse inline formatting in the content, using itemStyle as the base
	contentSpans := parseInline(content, itemStyle, InlineOpts{
		SourceMap:      &entries,
		LinkMap:        &linkEntries,
		SourceOffset:   sourceOffset + contentStart,
		RenderedOffset: renderedOffset,
	})
	spans = append(spans, contentSpans...)

	return spans, entries, linkEntries
}

// parseTableBlockWithSourceMap parses a table starting at the given line index.
// Returns the spans for the table, source map entries, and the number of lines consumed.
func parseTableBlockWithSourceMap(lines []string, startIdx int, sourceOffset, renderedOffset int) ([]rich.Span, []SourceMapEntry, int) {
	if startIdx >= len(lines) {
		return nil, nil, 0
	}

	// First line should be header row
	isRow, _ := isTableRow(lines[startIdx])
	if !isRow {
		return nil, nil, 0
	}

	// Second line should be separator row
	if startIdx+1 >= len(lines) || !isTableSeparatorRow(lines[startIdx+1]) {
		return nil, nil, 0
	}

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
		return nil, nil, 0
	}

	// Parse alignment from separator row (line index 1)
	_, aligns := parseTableSeparator(tableLines[1])

	// Parse all rows into cells (skip separator at index 1)
	var allCells [][]string
	for i, line := range tableLines {
		if i == 1 {
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

	borderStyle := rich.Style{
		Table: true,
		Code:  true,
		Block: true,
		Bg:    rich.InlineCodeBg,
		Scale: 1.0,
	}

	var spans []rich.Span
	var entries []SourceMapEntry
	srcPos := sourceOffset
	rendPos := renderedOffset

	// Top border (synthetic — no source mapping, zero-length source range)
	topText := topBorder + "\n"
	topLen := len([]rune(topText))
	spans = append(spans, rich.Span{
		Text:  topText,
		Style: borderStyle,
	})
	entries = append(entries, SourceMapEntry{
		RenderedStart: rendPos,
		RenderedEnd:   rendPos + topLen,
		SourceStart:   srcPos,
		SourceEnd:     srcPos, // zero-length: synthetic line
	})
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
			// Separator maps to the source separator line
			entries = append(entries, SourceMapEntry{
				RenderedStart: rendPos,
				RenderedEnd:   rendPos + sepLen,
				SourceStart:   srcPos,
				SourceEnd:     srcPos + len(line),
			})
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

			renderedLen := len([]rune(lineText))
			entries = append(entries, SourceMapEntry{
				RenderedStart: rendPos,
				RenderedEnd:   rendPos + renderedLen,
				SourceStart:   srcPos,
				SourceEnd:     srcPos + len(line),
			})
			rendPos += renderedLen
			srcPos += len(line)
		}
	}

	// Bottom border with trailing newline (synthetic — no source mapping, zero-length source range)
	bottomLen := len([]rune(bottomBorder)) + 1 // +1 for trailing newline
	spans = append(spans, rich.Span{
		Text:  bottomBorder + "\n",
		Style: borderStyle,
	})
	entries = append(entries, SourceMapEntry{
		RenderedStart: rendPos,
		RenderedEnd:   rendPos + bottomLen,
		SourceStart:   srcPos,
		SourceEnd:     srcPos, // zero-length: synthetic line
	})
	rendPos += bottomLen

	return spans, entries, consumed
}
