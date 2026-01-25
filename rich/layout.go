package rich

import (
	"unicode/utf8"

	"github.com/rjkroege/edwood/draw"
)

// contentToBoxes converts Content (styled spans) into a sequence of Boxes.
// Each Box represents either a run of text, a newline, or a tab.
// Text is split on newlines and tabs, which become their own boxes.
func contentToBoxes(c Content) []Box {
	var boxes []Box

	for _, span := range c {
		if span.Text == "" {
			continue
		}
		boxes = appendSpanBoxes(boxes, span)
	}

	return boxes
}

// appendSpanBoxes appends boxes from a single span to the slice.
// It splits the span text on newlines and tabs.
func appendSpanBoxes(boxes []Box, span Span) []Box {
	text := span.Text
	style := span.Style

	for len(text) > 0 {
		// Find the next special character (newline or tab)
		idx := -1
		var special rune
		for i, r := range text {
			if r == '\n' || r == '\t' {
				idx = i
				special = r
				break
			}
		}

		if idx == -1 {
			// No more special characters, emit the rest as a text box
			boxes = append(boxes, Box{
				Text:  []byte(text),
				Nrune: utf8.RuneCountInString(text),
				Bc:    0,
				Style: style,
			})
			break
		}

		// Emit text before the special character (if any)
		if idx > 0 {
			prefix := text[:idx]
			boxes = append(boxes, Box{
				Text:  []byte(prefix),
				Nrune: utf8.RuneCountInString(prefix),
				Bc:    0,
				Style: style,
			})
		}

		// Emit the special character box
		boxes = append(boxes, Box{
			Text:  nil,
			Nrune: -1,
			Bc:    special,
			Style: style,
		})

		// Continue with the rest of the text (after the special character)
		text = text[idx+1:]
	}

	return boxes
}

// boxWidth calculates the width of a box in pixels using font metrics.
// For text boxes, it measures the text width using the font.
// For newline and tab boxes, it returns 0 (tabs are handled separately by tabBoxWidth).
func boxWidth(box *Box, font draw.Font) int {
	if box.IsNewline() || box.IsTab() {
		return 0
	}
	if len(box.Text) == 0 {
		return 0
	}
	return font.BytesWidth(box.Text)
}

// tabBoxWidth calculates the width of a tab box based on its position.
// Tab stops are aligned to multiples of maxtab pixels, relative to minX.
// xPos is the current X position, minX is the left edge of the frame.
func tabBoxWidth(box *Box, xPos, minX, maxtab int) int {
	if !box.IsTab() {
		return 0
	}
	// Calculate position relative to frame origin
	relPos := xPos - minX
	// Find distance to next tab stop
	return maxtab - (relPos % maxtab)
}
