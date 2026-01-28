package markdown

import (
	"fmt"
	"strings"

	"github.com/rjkroege/edwood/rich"
)

// headingScales maps heading level (1-6) to scale factor.
var headingScales = [7]float64{
	0:     1.0,   // not used (no level 0)
	1:     2.0,   // H1
	2:     1.5,   // H2
	3:     1.25,  // H3
	4:     1.125, // H4
	5:     1.0,   // H5
	6:     0.875, // H6
}

// Parse converts markdown text to styled rich.Content.
func Parse(text string) rich.Content {
	if text == "" {
		return rich.Content{}
	}

	var result rich.Content
	lines := splitLines(text)

	// Track fenced code block state
	inFencedBlock := false
	var codeBlockContent strings.Builder

	// Track indented code block state
	inIndentedBlock := false
	var indentedBlockContent strings.Builder

	// Helper to emit indented code block
	emitIndentedBlock := func() {
		if indentedBlockContent.Len() > 0 {
			codeSpan := rich.Span{
				Text: indentedBlockContent.String(),
				Style: rich.Style{
					Bg:    rich.InlineCodeBg,
					Code:  true,
					Block: true,
					Scale: 1.0,
				},
			}
			result = append(result, codeSpan)
			indentedBlockContent.Reset()
		}
		inIndentedBlock = false
	}

	// Track if we're in a paragraph (consecutive non-block lines)
	inParagraph := false

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
			}
			inParagraph = false
			if !inFencedBlock {
				// Opening fence - start collecting code
				inFencedBlock = true
				codeBlockContent.Reset()
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
					result = append(result, codeSpan)
				}
				continue
			}
		}

		if inFencedBlock {
			// Inside fenced block - collect raw content without parsing
			codeBlockContent.WriteString(line)
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
			}
			inParagraph = false
			inIndentedBlock = true
			// Remove the indent prefix and add to block
			indentedBlockContent.WriteString(stripIndent(line))
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
			if inParagraph {
				// End the paragraph with a newline (with ParaBreak for extra spacing)
				result = append(result, rich.Span{
					Text:  "\n",
					Style: rich.Style{ParaBreak: true, Scale: 1.0},
				})
				inParagraph = false
			}
			continue
		}

		// Check for table (must have header row followed by separator row)
		isRow, _ := isTableRow(line)
		if isRow && i+1 < len(lines) && isTableSeparatorRow(lines[i+1]) {
			// End paragraph before table
			if inParagraph && len(result) > 0 {
				result[len(result)-1].Text += "\n"
			}
			inParagraph = false

			// Parse the table - collect all consecutive table rows
			tableSpans, consumed := parseTableBlock(lines, i)
			result = append(result, tableSpans...)
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
				}
			}
			inParagraph = true
		}

		// Normal line parsing - strip trailing newline for paragraph text
		// But preserve newline for list items so they end with \n
		lineToPass := line
		if !isBlockElement {
			lineToPass = strings.TrimSuffix(line, "\n")
		} else if isListItem {
			// List items keep their trailing newline (if present) in the content
			// but parseLine will handle the line with the newline
		}
		spans := parseLine(lineToPass)

		// Merge consecutive spans with the same style
		// (but don't merge link spans or list item spans - each should remain distinct)
		for _, span := range spans {
			if len(result) > 0 && result[len(result)-1].Style == span.Style && !span.Style.Link && !span.Style.ListItem && !span.Style.ListBullet {
				// Merge with previous span
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
			result = append(result, codeSpan)
		}
	}

	// Handle trailing indented code block
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
	// Check for horizontal rule (---, ***, ___)
	if isHorizontalRule(line) {
		// Emit the HRuleRune marker plus newline if the original line had one
		text := string(rich.HRuleRune)
		if strings.HasSuffix(line, "\n") {
			text += "\n"
		}
		return []rich.Span{{
			Text:  text,
			Style: rich.StyleHRule,
		}}
	}

	// Check for heading (# at start of line)
	level := headingLevel(line)
	if level > 0 {
		// Extract heading text (strip # prefix and leading space)
		content := line[level:] // Remove the # characters
		content = strings.TrimLeft(content, " ")
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
		return parseUnorderedListItem(line, indentLevel, contentStart)
	}

	// Check for ordered list item (1., 2), etc.)
	if isOL, indentLevel, contentStart, itemNumber := isOrderedListItem(line); isOL {
		return parseOrderedListItem(line, indentLevel, contentStart, itemNumber)
	}

	// Parse inline formatting (bold, italic)
	return parseInlineFormatting(line, rich.DefaultStyle())
}

// parseInlineFormatting parses code spans, bold/italic markers, links, and images in text and returns styled spans.
func parseInlineFormatting(text string, baseStyle rich.Style) []rich.Span {
	var spans []rich.Span
	var currentText strings.Builder
	i := 0

	for i < len(text) {
		// Check for ![ (potential image) - must be checked before link
		if text[i] == '!' && i+1 < len(text) && text[i+1] == '[' {
			// Try to parse as image: ![alt](url)
			altEnd := strings.Index(text[i+2:], "]")
			if altEnd != -1 {
				closeBracket := i + 2 + altEnd
				// Check if immediately followed by (
				if closeBracket+1 < len(text) && text[closeBracket+1] == '(' {
					// Find closing ) - handle titles like ![alt](url "title") or ![alt](url 'title')
					urlStart := closeBracket + 2
					urlEnd := -1
					for j := urlStart; j < len(text); j++ {
						if text[j] == ')' {
							urlEnd = j
							break
						}
					}
					if urlEnd != -1 {
						// We have a valid image pattern
						// Flush any accumulated plain text
						if currentText.Len() > 0 {
							spans = append(spans, rich.Span{
								Text:  currentText.String(),
								Style: baseStyle,
							})
							currentText.Reset()
						}

						// Extract alt text and URL
						altText := text[i+2 : closeBracket]
						urlPart := text[urlStart:urlEnd]
						// Parse URL (may contain title with width tag)
						url, title := parseURLPart(urlPart)

						// Create image placeholder span
						placeholderText := "[Image: " + altText + "]"
						if altText == "" {
							placeholderText = "[Image]"
						}
						imageStyle := rich.Style{
							Fg:       rich.LinkBlue, // Use blue like links for now
							Bg:       baseStyle.Bg,
							Image:    true,
							ImageURL: url,
							ImageAlt:   altText,
							ImageWidth: parseImageWidth(title),
							Scale:      baseStyle.Scale,
						}
						spans = append(spans, rich.Span{
							Text:  placeholderText,
							Style: imageStyle,
						})

						// Skip past the entire image syntax
						i = urlEnd + 1
						continue
					}
				}
			}
			// Not a valid image, treat ! as regular text
			currentText.WriteByte(text[i])
			i++
			continue
		}

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
						// Flush any accumulated plain text
						if currentText.Len() > 0 {
							spans = append(spans, rich.Span{
								Text:  currentText.String(),
								Style: baseStyle,
							})
							currentText.Reset()
						}

						// Extract link text and parse it for inline formatting
						linkText := text[i+1 : closeBracket]
						// Parse link text with Link style as base
						// Use LinkBlue for the foreground color (standard blue for hyperlinks)
						linkStyle := rich.Style{
							Fg:    rich.LinkBlue,
							Bg:    baseStyle.Bg,
							Link:  true,
							Scale: baseStyle.Scale,
						}
						if linkText == "" {
							// Empty link text
							spans = append(spans, rich.Span{
								Text:  "",
								Style: linkStyle,
							})
						} else {
							// Parse link text for bold/italic
							linkSpans := parseInlineFormattingNoLinks(linkText, linkStyle)
							spans = append(spans, linkSpans...)
						}

						// Skip past the entire link
						i = closeBracket + 2 + urlEnd + 1
						continue
					}
				}
			}
			// Not a valid link, treat [ as regular text
			currentText.WriteByte(text[i])
			i++
			continue
		}

		// Check for ` (code span) - must be checked before bold/italic
		// so that asterisks inside code spans are preserved literally
		if text[i] == '`' {
			// Find closing `
			end := strings.Index(text[i+1:], "`")
			if end != -1 {
				// Flush any accumulated plain text
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				// Add code span (content between backticks is NOT further parsed)
				spans = append(spans, rich.Span{
					Text: text[i+1 : i+1+end],
					Style: rich.Style{
						Fg:    baseStyle.Fg,
						Bg:    rich.InlineCodeBg,
						Code:  true,
						Scale: baseStyle.Scale,
					},
				})
				i = i + 1 + end + 1
				continue
			}
			// No closing ` found, treat as literal text
			currentText.WriteByte(text[i])
			i++
			continue
		}

		// Check for *** (bold+italic)
		if i+2 < len(text) && text[i:i+3] == "***" {
			// Find closing ***
			end := strings.Index(text[i+3:], "***")
			if end != -1 {
				// Flush any accumulated plain text
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				// Add bold+italic span
				spans = append(spans, rich.Span{
					Text: text[i+3 : i+3+end],
					Style: rich.Style{
						Fg:     baseStyle.Fg,
						Bg:     baseStyle.Bg,
						Bold:   true,
						Italic: true,
						Scale:  baseStyle.Scale,
					},
				})
				i = i + 3 + end + 3
				continue
			}
		}

		// Check for ** (bold)
		if i+1 < len(text) && text[i:i+2] == "**" {
			// Find closing **
			end := strings.Index(text[i+2:], "**")
			if end != -1 {
				// Flush any accumulated plain text
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				// Add bold span
				spans = append(spans, rich.Span{
					Text: text[i+2 : i+2+end],
					Style: rich.Style{
						Fg:     baseStyle.Fg,
						Bg:     baseStyle.Bg,
						Bold:   true,
						Italic: baseStyle.Italic,
						Scale:  baseStyle.Scale,
					},
				})
				i = i + 2 + end + 2
				continue
			}
			// No closing ** found, treat as literal text
			currentText.WriteString("**")
			i += 2
			continue
		}

		// Check for * (italic)
		if text[i] == '*' {
			// Find closing *
			end := strings.Index(text[i+1:], "*")
			if end != -1 {
				// Flush any accumulated plain text
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				// Add italic span
				spans = append(spans, rich.Span{
					Text: text[i+1 : i+1+end],
					Style: rich.Style{
						Fg:     baseStyle.Fg,
						Bg:     baseStyle.Bg,
						Bold:   baseStyle.Bold,
						Italic: true,
						Scale:  baseStyle.Scale,
					},
				})
				i = i + 1 + end + 1
				continue
			}
		}

		// Regular character
		currentText.WriteByte(text[i])
		i++
	}

	// Flush any remaining text
	if currentText.Len() > 0 {
		spans = append(spans, rich.Span{
			Text:  currentText.String(),
			Style: baseStyle,
		})
	}

	// If no spans were created, return a single span with original text
	if len(spans) == 0 {
		return []rich.Span{{
			Text:  text,
			Style: baseStyle,
		}}
	}

	return spans
}

// parseInlineFormattingWithListStyle parses inline formatting while preserving list style fields.
// All output spans will have ListItem, ListIndent, ListOrdered, and ListNumber from baseStyle.
func parseInlineFormattingWithListStyle(text string, baseStyle rich.Style) []rich.Span {
	var spans []rich.Span
	var currentText strings.Builder
	i := 0

	for i < len(text) {
		// Check for ![ (potential image) - must be checked before link
		if text[i] == '!' && i+1 < len(text) && text[i+1] == '[' {
			altEnd := strings.Index(text[i+2:], "]")
			if altEnd != -1 {
				closeBracket := i + 2 + altEnd
				if closeBracket+1 < len(text) && text[closeBracket+1] == '(' {
					urlStart := closeBracket + 2
					urlEnd := -1
					for j := urlStart; j < len(text); j++ {
						if text[j] == ')' {
							urlEnd = j
							break
						}
					}
					if urlEnd != -1 {
						if currentText.Len() > 0 {
							spans = append(spans, rich.Span{
								Text:  currentText.String(),
								Style: baseStyle,
							})
							currentText.Reset()
						}
						altText := text[i+2 : closeBracket]
						urlPart := text[urlStart:urlEnd]
						url, title := parseURLPart(urlPart)
						placeholderText := "[Image: " + altText + "]"
						if altText == "" {
							placeholderText = "[Image]"
						}
						imageStyle := rich.Style{
							Fg:          rich.LinkBlue,
							Bg:          baseStyle.Bg,
							Image:       true,
							ImageURL:    url,
							ImageAlt:    altText,
							ImageWidth:  parseImageWidth(title),
							ListItem:    baseStyle.ListItem,
							ListIndent:  baseStyle.ListIndent,
							ListOrdered: baseStyle.ListOrdered,
							ListNumber:  baseStyle.ListNumber,
							Scale:       baseStyle.Scale,
						}
						spans = append(spans, rich.Span{
							Text:  placeholderText,
							Style: imageStyle,
						})
						i = urlEnd + 1
						continue
					}
				}
			}
			currentText.WriteByte(text[i])
			i++
			continue
		}

		// Check for [ (potential link) - must be checked early
		if text[i] == '[' {
			linkEnd := strings.Index(text[i+1:], "]")
			if linkEnd != -1 {
				closeBracket := i + 1 + linkEnd
				if closeBracket+1 < len(text) && text[closeBracket+1] == '(' {
					urlEnd := strings.Index(text[closeBracket+2:], ")")
					if urlEnd != -1 {
						if currentText.Len() > 0 {
							spans = append(spans, rich.Span{
								Text:  currentText.String(),
								Style: baseStyle,
							})
							currentText.Reset()
						}
						linkText := text[i+1 : closeBracket]
						linkStyle := rich.Style{
							Fg:          rich.LinkBlue,
							Bg:          baseStyle.Bg,
							Link:        true,
							ListItem:    baseStyle.ListItem,
							ListIndent:  baseStyle.ListIndent,
							ListOrdered: baseStyle.ListOrdered,
							ListNumber:  baseStyle.ListNumber,
							Scale:       baseStyle.Scale,
						}
						if linkText == "" {
							spans = append(spans, rich.Span{
								Text:  "",
								Style: linkStyle,
							})
						} else {
							linkSpans := parseInlineFormattingWithListStyleNoLinks(linkText, linkStyle)
							spans = append(spans, linkSpans...)
						}
						i = closeBracket + 2 + urlEnd + 1
						continue
					}
				}
			}
			currentText.WriteByte(text[i])
			i++
			continue
		}

		// Check for ` (code span)
		if text[i] == '`' {
			end := strings.Index(text[i+1:], "`")
			if end != -1 {
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				spans = append(spans, rich.Span{
					Text: text[i+1 : i+1+end],
					Style: rich.Style{
						Fg:          baseStyle.Fg,
						Bg:          rich.InlineCodeBg,
						Code:        true,
						ListItem:    baseStyle.ListItem,
						ListIndent:  baseStyle.ListIndent,
						ListOrdered: baseStyle.ListOrdered,
						ListNumber:  baseStyle.ListNumber,
						Scale:       baseStyle.Scale,
					},
				})
				i = i + 1 + end + 1
				continue
			}
			currentText.WriteByte(text[i])
			i++
			continue
		}

		// Check for *** (bold+italic)
		if i+2 < len(text) && text[i:i+3] == "***" {
			end := strings.Index(text[i+3:], "***")
			if end != -1 {
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				spans = append(spans, rich.Span{
					Text: text[i+3 : i+3+end],
					Style: rich.Style{
						Fg:          baseStyle.Fg,
						Bg:          baseStyle.Bg,
						Bold:        true,
						Italic:      true,
						ListItem:    baseStyle.ListItem,
						ListIndent:  baseStyle.ListIndent,
						ListOrdered: baseStyle.ListOrdered,
						ListNumber:  baseStyle.ListNumber,
						Scale:       baseStyle.Scale,
					},
				})
				i = i + 3 + end + 3
				continue
			}
		}

		// Check for ** (bold)
		if i+1 < len(text) && text[i:i+2] == "**" {
			end := strings.Index(text[i+2:], "**")
			if end != -1 {
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				spans = append(spans, rich.Span{
					Text: text[i+2 : i+2+end],
					Style: rich.Style{
						Fg:          baseStyle.Fg,
						Bg:          baseStyle.Bg,
						Bold:        true,
						Italic:      baseStyle.Italic,
						ListItem:    baseStyle.ListItem,
						ListIndent:  baseStyle.ListIndent,
						ListOrdered: baseStyle.ListOrdered,
						ListNumber:  baseStyle.ListNumber,
						Scale:       baseStyle.Scale,
					},
				})
				i = i + 2 + end + 2
				continue
			}
			currentText.WriteString("**")
			i += 2
			continue
		}

		// Check for * (italic)
		if text[i] == '*' {
			end := strings.Index(text[i+1:], "*")
			if end != -1 {
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				spans = append(spans, rich.Span{
					Text: text[i+1 : i+1+end],
					Style: rich.Style{
						Fg:          baseStyle.Fg,
						Bg:          baseStyle.Bg,
						Bold:        baseStyle.Bold,
						Italic:      true,
						ListItem:    baseStyle.ListItem,
						ListIndent:  baseStyle.ListIndent,
						ListOrdered: baseStyle.ListOrdered,
						ListNumber:  baseStyle.ListNumber,
						Scale:       baseStyle.Scale,
					},
				})
				i = i + 1 + end + 1
				continue
			}
		}

		currentText.WriteByte(text[i])
		i++
	}

	if currentText.Len() > 0 {
		spans = append(spans, rich.Span{
			Text:  currentText.String(),
			Style: baseStyle,
		})
	}

	if len(spans) == 0 {
		return []rich.Span{{
			Text:  text,
			Style: baseStyle,
		}}
	}

	return spans
}

// parseInlineFormattingWithListStyleNoLinks parses inline formatting while preserving list style fields,
// but does not parse links (to avoid infinite recursion when parsing link text).
func parseInlineFormattingWithListStyleNoLinks(text string, baseStyle rich.Style) []rich.Span {
	var spans []rich.Span
	var currentText strings.Builder
	i := 0

	for i < len(text) {
		// Check for ` (code span)
		if text[i] == '`' {
			end := strings.Index(text[i+1:], "`")
			if end != -1 {
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				spans = append(spans, rich.Span{
					Text: text[i+1 : i+1+end],
					Style: rich.Style{
						Fg:          baseStyle.Fg,
						Bg:          rich.InlineCodeBg,
						Code:        true,
						Link:        baseStyle.Link,
						ListItem:    baseStyle.ListItem,
						ListIndent:  baseStyle.ListIndent,
						ListOrdered: baseStyle.ListOrdered,
						ListNumber:  baseStyle.ListNumber,
						Scale:       baseStyle.Scale,
					},
				})
				i = i + 1 + end + 1
				continue
			}
			currentText.WriteByte(text[i])
			i++
			continue
		}

		// Check for *** (bold+italic)
		if i+2 < len(text) && text[i:i+3] == "***" {
			end := strings.Index(text[i+3:], "***")
			if end != -1 {
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				spans = append(spans, rich.Span{
					Text: text[i+3 : i+3+end],
					Style: rich.Style{
						Fg:          baseStyle.Fg,
						Bg:          baseStyle.Bg,
						Bold:        true,
						Italic:      true,
						Link:        baseStyle.Link,
						ListItem:    baseStyle.ListItem,
						ListIndent:  baseStyle.ListIndent,
						ListOrdered: baseStyle.ListOrdered,
						ListNumber:  baseStyle.ListNumber,
						Scale:       baseStyle.Scale,
					},
				})
				i = i + 3 + end + 3
				continue
			}
		}

		// Check for ** (bold)
		if i+1 < len(text) && text[i:i+2] == "**" {
			end := strings.Index(text[i+2:], "**")
			if end != -1 {
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				spans = append(spans, rich.Span{
					Text: text[i+2 : i+2+end],
					Style: rich.Style{
						Fg:          baseStyle.Fg,
						Bg:          baseStyle.Bg,
						Bold:        true,
						Italic:      baseStyle.Italic,
						Link:        baseStyle.Link,
						ListItem:    baseStyle.ListItem,
						ListIndent:  baseStyle.ListIndent,
						ListOrdered: baseStyle.ListOrdered,
						ListNumber:  baseStyle.ListNumber,
						Scale:       baseStyle.Scale,
					},
				})
				i = i + 2 + end + 2
				continue
			}
			currentText.WriteString("**")
			i += 2
			continue
		}

		// Check for * (italic)
		if text[i] == '*' {
			end := strings.Index(text[i+1:], "*")
			if end != -1 {
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				spans = append(spans, rich.Span{
					Text: text[i+1 : i+1+end],
					Style: rich.Style{
						Fg:          baseStyle.Fg,
						Bg:          baseStyle.Bg,
						Bold:        baseStyle.Bold,
						Italic:      true,
						Link:        baseStyle.Link,
						ListItem:    baseStyle.ListItem,
						ListIndent:  baseStyle.ListIndent,
						ListOrdered: baseStyle.ListOrdered,
						ListNumber:  baseStyle.ListNumber,
						Scale:       baseStyle.Scale,
					},
				})
				i = i + 1 + end + 1
				continue
			}
		}

		currentText.WriteByte(text[i])
		i++
	}

	if currentText.Len() > 0 {
		spans = append(spans, rich.Span{
			Text:  currentText.String(),
			Style: baseStyle,
		})
	}

	if len(spans) == 0 {
		return []rich.Span{{
			Text:  text,
			Style: baseStyle,
		}}
	}

	return spans
}

// parseInlineFormattingNoLinks parses code spans, bold/italic markers but NOT links.
// Used for parsing text inside link labels to avoid infinite recursion.
func parseInlineFormattingNoLinks(text string, baseStyle rich.Style) []rich.Span {
	var spans []rich.Span
	var currentText strings.Builder
	i := 0

	for i < len(text) {
		// Check for ` (code span) - must be checked before bold/italic
		if text[i] == '`' {
			end := strings.Index(text[i+1:], "`")
			if end != -1 {
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				spans = append(spans, rich.Span{
					Text: text[i+1 : i+1+end],
					Style: rich.Style{
						Fg:    baseStyle.Fg,
						Bg:    rich.InlineCodeBg,
						Code:  true,
						Link:  baseStyle.Link,
						Scale: baseStyle.Scale,
					},
				})
				i = i + 1 + end + 1
				continue
			}
			currentText.WriteByte(text[i])
			i++
			continue
		}

		// Check for *** (bold+italic)
		if i+2 < len(text) && text[i:i+3] == "***" {
			end := strings.Index(text[i+3:], "***")
			if end != -1 {
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				spans = append(spans, rich.Span{
					Text: text[i+3 : i+3+end],
					Style: rich.Style{
						Fg:     baseStyle.Fg,
						Bg:     baseStyle.Bg,
						Bold:   true,
						Italic: true,
						Link:   baseStyle.Link,
						Scale:  baseStyle.Scale,
					},
				})
				i = i + 3 + end + 3
				continue
			}
		}

		// Check for ** (bold)
		if i+1 < len(text) && text[i:i+2] == "**" {
			end := strings.Index(text[i+2:], "**")
			if end != -1 {
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				spans = append(spans, rich.Span{
					Text: text[i+2 : i+2+end],
					Style: rich.Style{
						Fg:     baseStyle.Fg,
						Bg:     baseStyle.Bg,
						Bold:   true,
						Italic: baseStyle.Italic,
						Link:   baseStyle.Link,
						Scale:  baseStyle.Scale,
					},
				})
				i = i + 2 + end + 2
				continue
			}
			currentText.WriteString("**")
			i += 2
			continue
		}

		// Check for * (italic)
		if text[i] == '*' {
			end := strings.Index(text[i+1:], "*")
			if end != -1 {
				if currentText.Len() > 0 {
					spans = append(spans, rich.Span{
						Text:  currentText.String(),
						Style: baseStyle,
					})
					currentText.Reset()
				}
				spans = append(spans, rich.Span{
					Text: text[i+1 : i+1+end],
					Style: rich.Style{
						Fg:     baseStyle.Fg,
						Bg:     baseStyle.Bg,
						Bold:   baseStyle.Bold,
						Italic: true,
						Link:   baseStyle.Link,
						Scale:  baseStyle.Scale,
					},
				})
				i = i + 1 + end + 1
				continue
			}
		}

		currentText.WriteByte(text[i])
		i++
	}

	if currentText.Len() > 0 {
		spans = append(spans, rich.Span{
			Text:  currentText.String(),
			Style: baseStyle,
		})
	}

	if len(spans) == 0 {
		return []rich.Span{{
			Text:  text,
			Style: baseStyle,
		}}
	}

	return spans
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
// It emits: bullet span + space span + content spans (with inline formatting).
func parseUnorderedListItem(line string, indentLevel int, contentStart int) []rich.Span {
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
	contentSpans := parseInlineFormattingWithListStyle(content, itemStyle)
	spans = append(spans, contentSpans...)

	return spans
}

// parseOrderedListItem parses an ordered list line and returns styled spans.
// It emits: number span + space span + content spans (with inline formatting).
func parseOrderedListItem(line string, indentLevel int, contentStart int, itemNumber int) []rich.Span {
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
	spans = append(spans, rich.Span{
		Text:  fmt.Sprintf("%d.", itemNumber),
		Style: bulletStyle,
	})

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
	contentSpans := parseInlineFormattingWithListStyle(content, itemStyle)
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
				w := len(cell)
				if w > widths[i] {
					widths[i] = w
				}
			}
		}
	}

	return widths
}

// parseTableBlock parses a table starting at the given line index.
// Returns the spans for the table and the number of lines consumed.
func parseTableBlock(lines []string, startIdx int) ([]rich.Span, int) {
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

	// Build spans for each table row
	var spans []rich.Span

	for i, line := range tableLines {
		// Normalize line ending
		lineText := strings.TrimSuffix(line, "\n")

		// Determine if this is header row, separator row, or data row
		isHeader := i == 0
		isSeparator := i == 1 && isTableSeparatorRow(line)

		// Add newline unless it's the last line
		if i < len(tableLines)-1 {
			lineText += "\n"
		}

		style := rich.Style{
			Table:       true,
			TableHeader: isHeader,
			Code:        true, // Tables use code/monospace font
			Block:       true, // Tables are block-level elements
			Bg:          rich.InlineCodeBg,
			Scale:       1.0,
		}

		// Headers are also bold
		if isHeader {
			style.Bold = true
		}

		// Separator rows are styled same as data rows (not header, not bold)
		if isSeparator {
			style.TableHeader = false
			style.Bold = false
		}

		spans = append(spans, rich.Span{
			Text:  lineText,
			Style: style,
		})
	}

	return spans, consumed
}
