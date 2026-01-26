package markdown

import (
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

		// Check for indented code block (4 spaces or 1 tab)
		if isIndentedCodeLine(line) {
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

		// Check if this is a block-level element (heading, hrule)
		isBlockElement := headingLevel(line) > 0 || isHorizontalRule(line)

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
		lineToPass := line
		if !isBlockElement {
			lineToPass = strings.TrimSuffix(line, "\n")
		}
		spans := parseLine(lineToPass)

		// Merge consecutive spans with the same style
		// (but don't merge link spans - each link should remain distinct
		// for proper LinkMap tracking)
		for _, span := range spans {
			if len(result) > 0 && result[len(result)-1].Style == span.Style && !span.Style.Link {
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

	// Parse inline formatting (bold, italic)
	return parseInlineFormatting(line, rich.DefaultStyle())
}

// parseInlineFormatting parses code spans, bold/italic markers, and links in text and returns styled spans.
func parseInlineFormatting(text string, baseStyle rich.Style) []rich.Span {
	var spans []rich.Span
	var currentText strings.Builder
	i := 0

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
