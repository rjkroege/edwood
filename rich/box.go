package rich

// Box represents a positioned, styled fragment of text.
// This is the layout model - produced by laying out Spans.
type Box struct {
	// Content
	Text  []byte // UTF-8 content (empty for newline/tab)
	Nrune int    // Rune count (-1 for special boxes)
	Bc    rune   // Box character: 0 for text, '\n' for newline, '\t' for tab

	// Style
	Style Style

	// Layout (computed)
	Wid int // Width in pixels

	// Image-specific fields (only used when Style.Image is true)
	ImageData *CachedImage // Loaded image data for rendering
}

// IsNewline returns true if this is a newline box.
func (b *Box) IsNewline() bool {
	return b.Nrune < 0 && b.Bc == '\n'
}

// IsTab returns true if this is a tab box.
func (b *Box) IsTab() bool {
	return b.Nrune < 0 && b.Bc == '\t'
}

// IsImage returns true if this box represents an image.
// An image box has Style.Image=true and ImageData set.
func (b *Box) IsImage() bool {
	return b.Style.Image && b.ImageData != nil
}
