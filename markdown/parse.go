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

	for _, line := range lines {
		spans := parseLine(line)
		// Merge consecutive spans with the same style
		for _, span := range spans {
			if len(result) > 0 && result[len(result)-1].Style == span.Style {
				// Merge with previous span
				result[len(result)-1].Text += span.Text
			} else {
				result = append(result, span)
			}
		}
	}

	return result
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

// parseInlineFormatting parses bold/italic markers in text and returns styled spans.
func parseInlineFormatting(text string, baseStyle rich.Style) []rich.Span {
	var spans []rich.Span
	var currentText strings.Builder
	i := 0

	for i < len(text) {
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
