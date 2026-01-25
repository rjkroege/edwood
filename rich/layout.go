package rich

import "unicode/utf8"

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
