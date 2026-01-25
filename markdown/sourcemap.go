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

// ParseWithSourceMap parses markdown and returns both the styled content
// and a source map for mapping rendered positions back to source positions.
func ParseWithSourceMap(text string) (rich.Content, *SourceMap) {
	if text == "" {
		return rich.Content{}, &SourceMap{}
	}

	var result rich.Content
	sm := &SourceMap{}
	lines := splitLines(text)

	sourcePos := 0  // Current position in source
	renderedPos := 0 // Current position in rendered content

	for _, line := range lines {
		spans, entries := parseLineWithSourceMap(line, sourcePos, renderedPos)
		sm.entries = append(sm.entries, entries...)

		// Update rendered position based on spans
		for _, span := range spans {
			renderedPos += len([]rune(span.Text))
		}

		// Update source position
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

	return result, sm
}

// parseLineWithSourceMap parses a single line and returns spans plus source map entries.
func parseLineWithSourceMap(line string, sourceOffset, renderedOffset int) ([]rich.Span, []SourceMapEntry) {
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
		return []rich.Span{span}, []SourceMapEntry{entry}
	}

	// Parse inline formatting
	return parseInlineWithSourceMap(line, rich.DefaultStyle(), sourceOffset, renderedOffset)
}

// parseInlineWithSourceMap parses inline formatting and builds source map entries.
func parseInlineWithSourceMap(text string, baseStyle rich.Style, sourceOffset, renderedOffset int) ([]rich.Span, []SourceMapEntry) {
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

	return spans, entries
}
