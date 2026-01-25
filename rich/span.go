package rich

// Span represents a run of text with uniform style.
// This is the input model - what markdown parsing produces.
type Span struct {
	Text  string
	Style Style
}

// Content is a sequence of styled spans representing a document.
type Content []Span

// Plain creates Content from unstyled text.
func Plain(text string) Content {
	return Content{{Text: text, Style: DefaultStyle()}}
}

// Len returns total rune count.
func (c Content) Len() int {
	n := 0
	for _, s := range c {
		n += len([]rune(s.Text))
	}
	return n
}
