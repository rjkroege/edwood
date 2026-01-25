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

	// Plain text
	return []rich.Span{{
		Text:  line,
		Style: rich.DefaultStyle(),
	}}
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
