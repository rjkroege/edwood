package markdown

import (
	"strings"

	"github.com/rjkroege/edwood/rich"
)

// SourceMap maps positions in rendered content back to positions in source markdown.
type SourceMap struct {
	entries []SourceMapEntry
}

// SourceMapEntry maps a range in rendered content to a range in source markdown.
type SourceMapEntry struct {
	RenderedStart int // Rune position in rendered content
	RenderedEnd   int
	SourceStart   int // Byte position in source markdown
	SourceEnd     int
	PrefixLen     int // Length of source prefix not in rendered (e.g., "# " for headings)
}

// ToSource maps a range in rendered content (renderedStart, renderedEnd) to
// the corresponding range in the source markdown.
// When the selection spans formatted elements, it expands to include the full
// source markup (e.g., selecting "bold" in "**bold**" returns 0-8).
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
		srcStart = renderedStart
	} else {
		// If the selection starts at the beginning of a formatted element,
		// include the opening marker
		if renderedStart == startEntry.RenderedStart {
			srcStart = startEntry.SourceStart
		} else {
			// Calculate offset within this entry
			offset := renderedStart - startEntry.RenderedStart
			srcStart = startEntry.SourceStart + offset
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
		srcEnd = renderedEnd
	} else {
		// If the selection ends at the end of a formatted element,
		// include the closing marker
		if renderedEnd == endEntry.RenderedEnd {
			srcEnd = endEntry.SourceEnd
		} else {
			// Calculate offset within this entry, accounting for any prefix
			offset := renderedEnd - endEntry.RenderedStart
			srcEnd = endEntry.SourceStart + endEntry.PrefixLen + offset
		}
	}

	return srcStart, srcEnd
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

	for _, line := range lines {
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

		// Check for indented code block (4 spaces or 1 tab)
		if isIndentedCodeLine(line) {
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

		// Check if this is a block-level element (heading, hrule)
		isBlockElement := headingLevel(line) > 0 || isHorizontalRule(line)

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

	return result, sm, lm
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

	// Parse inline formatting
	return parseInlineWithSourceMap(line, rich.DefaultStyle(), sourceOffset, renderedOffset)
}

// parseInlineWithSourceMap parses inline formatting and builds source map and link map entries.
func parseInlineWithSourceMap(text string, baseStyle rich.Style, sourceOffset, renderedOffset int) ([]rich.Span, []SourceMapEntry, []LinkEntry) {
	var spans []rich.Span
	var entries []SourceMapEntry
	var linkEntries []LinkEntry
	var currentText strings.Builder
	i := 0
	srcPos := sourceOffset
	rendPos := renderedOffset

	flushPlain := func() {
		if currentText.Len() > 0 {
			t := currentText.String()
			spans = append(spans, rich.Span{
				Text:  t,
				Style: baseStyle,
			})
			currentText.Reset()
		}
	}

	for i < len(text) {
		// Check for [ (potential link) - must be checked early
		if text[i] == '[' {
			// Try to parse as link: [text](url)
			linkEnd := strings.Index(text[i+1:], "]")
			if linkEnd != -1 {
				closeBracket := i + 1 + linkEnd
				// Check if immediately followed by (
				if closeBracket+1 < len(text) && text[closeBracket+1] == '(' {
					// Find closing )
					urlEnd := strings.Index(text[closeBracket+2:], ")")
					if urlEnd != -1 {
						// We have a valid link pattern
						flushPlain()

						linkText := text[i+1 : closeBracket]
						url := text[closeBracket+2 : closeBracket+2+urlEnd]
						sourceLen := 1 + linkEnd + 1 + 1 + urlEnd + 1 // [text](url)

						// Parse link text with Link style as base
						// Use LinkBlue for the foreground color (standard blue for hyperlinks)
						linkStyle := rich.Style{
							Fg:    rich.LinkBlue,
							Bg:    baseStyle.Bg,
							Link:  true,
							Scale: baseStyle.Scale,
						}

						// Track the start of the link in rendered content
						linkRenderedStart := rendPos

						if linkText == "" {
							// Empty link text - nothing to render
						} else {
							// Parse link text for bold/italic (recursively, but without link detection)
							linkSpans, linkSourceEntries, _ := parseInlineWithSourceMapNoLinks(linkText, linkStyle, srcPos+1, rendPos)
							spans = append(spans, linkSpans...)
							entries = append(entries, linkSourceEntries...)
							for _, span := range linkSpans {
								rendPos += len([]rune(span.Text))
							}
						}

						// Track the end of the link in rendered content
						linkRenderedEnd := rendPos

						// Add link entry if there's actual content
						if linkRenderedEnd > linkRenderedStart {
							linkEntries = append(linkEntries, LinkEntry{
								Start: linkRenderedStart,
								End:   linkRenderedEnd,
								URL:   url,
							})
						}

						srcPos += sourceLen
						i = closeBracket + 2 + urlEnd + 1
						continue
					}
				}
			}
			// Not a valid link, treat [ as regular text
			currentText.WriteByte(text[i])
			entries = append(entries, SourceMapEntry{
				RenderedStart: rendPos,
				RenderedEnd:   rendPos + 1,
				SourceStart:   srcPos,
				SourceEnd:     srcPos + 1,
			})
			rendPos++
			srcPos++
			i++
			continue
		}

		// Check for ` (code span)
		if text[i] == '`' {
			end := strings.Index(text[i+1:], "`")
			if end != -1 {
				flushPlain()
				codeText := text[i+1 : i+1+end]
				codeLen := len([]rune(codeText))
				sourceLen := 1 + end + 1 // `code`

				spans = append(spans, rich.Span{
					Text: codeText,
					Style: rich.Style{
						Fg:    baseStyle.Fg,
						Bg:    baseStyle.Bg,
						Code:  true,
						Scale: baseStyle.Scale,
					},
				})
				entries = append(entries, SourceMapEntry{
					RenderedStart: rendPos,
					RenderedEnd:   rendPos + codeLen,
					SourceStart:   srcPos,
					SourceEnd:     srcPos + sourceLen,
				})
				rendPos += codeLen
				srcPos += sourceLen
				i = i + 1 + end + 1
				continue
			}
			currentText.WriteByte(text[i])
			entries = append(entries, SourceMapEntry{
				RenderedStart: rendPos,
				RenderedEnd:   rendPos + 1,
				SourceStart:   srcPos,
				SourceEnd:     srcPos + 1,
			})
			rendPos++
			srcPos++
			i++
			continue
		}

		// Check for *** (bold+italic)
		if i+2 < len(text) && text[i:i+3] == "***" {
			end := strings.Index(text[i+3:], "***")
			if end != -1 {
				flushPlain()
				innerText := text[i+3 : i+3+end]
				innerLen := len([]rune(innerText))
				sourceLen := 3 + end + 3 // ***text***

				spans = append(spans, rich.Span{
					Text: innerText,
					Style: rich.Style{
						Fg:     baseStyle.Fg,
						Bg:     baseStyle.Bg,
						Bold:   true,
						Italic: true,
						Scale:  baseStyle.Scale,
					},
				})
				entries = append(entries, SourceMapEntry{
					RenderedStart: rendPos,
					RenderedEnd:   rendPos + innerLen,
					SourceStart:   srcPos,
					SourceEnd:     srcPos + sourceLen,
				})
				rendPos += innerLen
				srcPos += sourceLen
				i = i + 3 + end + 3
				continue
			}
		}

		// Check for ** (bold)
		if i+1 < len(text) && text[i:i+2] == "**" {
			end := strings.Index(text[i+2:], "**")
			if end != -1 {
				flushPlain()
				innerText := text[i+2 : i+2+end]
				innerLen := len([]rune(innerText))
				sourceLen := 2 + end + 2 // **text**

				spans = append(spans, rich.Span{
					Text: innerText,
					Style: rich.Style{
						Fg:     baseStyle.Fg,
						Bg:     baseStyle.Bg,
						Bold:   true,
						Italic: baseStyle.Italic,
						Scale:  baseStyle.Scale,
					},
				})
				entries = append(entries, SourceMapEntry{
					RenderedStart: rendPos,
					RenderedEnd:   rendPos + innerLen,
					SourceStart:   srcPos,
					SourceEnd:     srcPos + sourceLen,
				})
				rendPos += innerLen
				srcPos += sourceLen
				i = i + 2 + end + 2
				continue
			}
			// No closing ** found
			currentText.WriteString("**")
			entries = append(entries, SourceMapEntry{
				RenderedStart: rendPos,
				RenderedEnd:   rendPos + 2,
				SourceStart:   srcPos,
				SourceEnd:     srcPos + 2,
			})
			rendPos += 2
			srcPos += 2
			i += 2
			continue
		}

		// Check for * (italic)
		if text[i] == '*' {
			end := strings.Index(text[i+1:], "*")
			if end != -1 {
				flushPlain()
				innerText := text[i+1 : i+1+end]
				innerLen := len([]rune(innerText))
				sourceLen := 1 + end + 1 // *text*

				spans = append(spans, rich.Span{
					Text: innerText,
					Style: rich.Style{
						Fg:     baseStyle.Fg,
						Bg:     baseStyle.Bg,
						Bold:   baseStyle.Bold,
						Italic: true,
						Scale:  baseStyle.Scale,
					},
				})
				entries = append(entries, SourceMapEntry{
					RenderedStart: rendPos,
					RenderedEnd:   rendPos + innerLen,
					SourceStart:   srcPos,
					SourceEnd:     srcPos + sourceLen,
				})
				rendPos += innerLen
				srcPos += sourceLen
				i = i + 1 + end + 1
				continue
			}
		}

		// Regular character
		currentText.WriteByte(text[i])
		entries = append(entries, SourceMapEntry{
			RenderedStart: rendPos,
			RenderedEnd:   rendPos + 1,
			SourceStart:   srcPos,
			SourceEnd:     srcPos + 1,
		})
		rendPos++
		srcPos++
		i++
	}

	// Flush any remaining text
	flushPlain()

	// If no spans were created, return a single span with original text
	if len(spans) == 0 && text != "" {
		spans = []rich.Span{{
			Text:  text,
			Style: baseStyle,
		}}
		entries = []SourceMapEntry{{
			RenderedStart: renderedOffset,
			RenderedEnd:   renderedOffset + len([]rune(text)),
			SourceStart:   sourceOffset,
			SourceEnd:     sourceOffset + len(text),
		}}
	}

	return spans, entries, linkEntries
}

// parseInlineWithSourceMapNoLinks parses inline formatting but NOT links.
// Used for parsing text inside link labels to avoid infinite recursion.
func parseInlineWithSourceMapNoLinks(text string, baseStyle rich.Style, sourceOffset, renderedOffset int) ([]rich.Span, []SourceMapEntry, []LinkEntry) {
	var spans []rich.Span
	var entries []SourceMapEntry
	var currentText strings.Builder
	i := 0
	srcPos := sourceOffset
	rendPos := renderedOffset

	flushPlain := func() {
		if currentText.Len() > 0 {
			t := currentText.String()
			spans = append(spans, rich.Span{
				Text:  t,
				Style: baseStyle,
			})
			currentText.Reset()
		}
	}

	for i < len(text) {
		// Check for ` (code span)
		if text[i] == '`' {
			end := strings.Index(text[i+1:], "`")
			if end != -1 {
				flushPlain()
				codeText := text[i+1 : i+1+end]
				codeLen := len([]rune(codeText))
				sourceLen := 1 + end + 1 // `code`

				spans = append(spans, rich.Span{
					Text: codeText,
					Style: rich.Style{
						Fg:    baseStyle.Fg,
						Bg:    baseStyle.Bg,
						Code:  true,
						Link:  baseStyle.Link,
						Scale: baseStyle.Scale,
					},
				})
				entries = append(entries, SourceMapEntry{
					RenderedStart: rendPos,
					RenderedEnd:   rendPos + codeLen,
					SourceStart:   srcPos,
					SourceEnd:     srcPos + sourceLen,
				})
				rendPos += codeLen
				srcPos += sourceLen
				i = i + 1 + end + 1
				continue
			}
			currentText.WriteByte(text[i])
			entries = append(entries, SourceMapEntry{
				RenderedStart: rendPos,
				RenderedEnd:   rendPos + 1,
				SourceStart:   srcPos,
				SourceEnd:     srcPos + 1,
			})
			rendPos++
			srcPos++
			i++
			continue
		}

		// Check for *** (bold+italic)
		if i+2 < len(text) && text[i:i+3] == "***" {
			end := strings.Index(text[i+3:], "***")
			if end != -1 {
				flushPlain()
				innerText := text[i+3 : i+3+end]
				innerLen := len([]rune(innerText))
				sourceLen := 3 + end + 3 // ***text***

				spans = append(spans, rich.Span{
					Text: innerText,
					Style: rich.Style{
						Fg:     baseStyle.Fg,
						Bg:     baseStyle.Bg,
						Bold:   true,
						Italic: true,
						Link:   baseStyle.Link,
						Scale:  baseStyle.Scale,
					},
				})
				entries = append(entries, SourceMapEntry{
					RenderedStart: rendPos,
					RenderedEnd:   rendPos + innerLen,
					SourceStart:   srcPos,
					SourceEnd:     srcPos + sourceLen,
				})
				rendPos += innerLen
				srcPos += sourceLen
				i = i + 3 + end + 3
				continue
			}
		}

		// Check for ** (bold)
		if i+1 < len(text) && text[i:i+2] == "**" {
			end := strings.Index(text[i+2:], "**")
			if end != -1 {
				flushPlain()
				innerText := text[i+2 : i+2+end]
				innerLen := len([]rune(innerText))
				sourceLen := 2 + end + 2 // **text**

				spans = append(spans, rich.Span{
					Text: innerText,
					Style: rich.Style{
						Fg:     baseStyle.Fg,
						Bg:     baseStyle.Bg,
						Bold:   true,
						Italic: baseStyle.Italic,
						Link:   baseStyle.Link,
						Scale:  baseStyle.Scale,
					},
				})
				entries = append(entries, SourceMapEntry{
					RenderedStart: rendPos,
					RenderedEnd:   rendPos + innerLen,
					SourceStart:   srcPos,
					SourceEnd:     srcPos + sourceLen,
				})
				rendPos += innerLen
				srcPos += sourceLen
				i = i + 2 + end + 2
				continue
			}
			// No closing ** found
			currentText.WriteString("**")
			entries = append(entries, SourceMapEntry{
				RenderedStart: rendPos,
				RenderedEnd:   rendPos + 2,
				SourceStart:   srcPos,
				SourceEnd:     srcPos + 2,
			})
			rendPos += 2
			srcPos += 2
			i += 2
			continue
		}

		// Check for * (italic)
		if text[i] == '*' {
			end := strings.Index(text[i+1:], "*")
			if end != -1 {
				flushPlain()
				innerText := text[i+1 : i+1+end]
				innerLen := len([]rune(innerText))
				sourceLen := 1 + end + 1 // *text*

				spans = append(spans, rich.Span{
					Text: innerText,
					Style: rich.Style{
						Fg:     baseStyle.Fg,
						Bg:     baseStyle.Bg,
						Bold:   baseStyle.Bold,
						Italic: true,
						Link:   baseStyle.Link,
						Scale:  baseStyle.Scale,
					},
				})
				entries = append(entries, SourceMapEntry{
					RenderedStart: rendPos,
					RenderedEnd:   rendPos + innerLen,
					SourceStart:   srcPos,
					SourceEnd:     srcPos + sourceLen,
				})
				rendPos += innerLen
				srcPos += sourceLen
				i = i + 1 + end + 1
				continue
			}
		}

		// Regular character
		currentText.WriteByte(text[i])
		entries = append(entries, SourceMapEntry{
			RenderedStart: rendPos,
			RenderedEnd:   rendPos + 1,
			SourceStart:   srcPos,
			SourceEnd:     srcPos + 1,
		})
		rendPos++
		srcPos++
		i++
	}

	// Flush any remaining text
	flushPlain()

	// If no spans were created, return a single span with original text
	if len(spans) == 0 && text != "" {
		spans = []rich.Span{{
			Text:  text,
			Style: baseStyle,
		}}
		entries = []SourceMapEntry{{
			RenderedStart: renderedOffset,
			RenderedEnd:   renderedOffset + len([]rune(text)),
			SourceStart:   sourceOffset,
			SourceEnd:     sourceOffset + len(text),
		}}
	}

	return spans, entries, nil
}
