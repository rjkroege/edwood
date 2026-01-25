package rich

import (
	"testing"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/edwoodtest"
)

// ensure draw.Font is used (suppresses unused import warning until boxWidth is implemented)
var _ draw.Font

func TestContentToBoxes(t *testing.T) {
	tests := []struct {
		name    string
		content Content
		want    []Box // expected boxes
	}{
		{
			name:    "empty content",
			content: Content{},
			want:    []Box{},
		},
		{
			name:    "empty span",
			content: Content{{Text: "", Style: DefaultStyle()}},
			want:    []Box{},
		},
		{
			name:    "simple text",
			content: Plain("hello"),
			want: []Box{
				{Text: []byte("hello"), Nrune: 5, Bc: 0, Style: DefaultStyle()},
			},
		},
		{
			name:    "single newline",
			content: Plain("\n"),
			want: []Box{
				{Text: nil, Nrune: -1, Bc: '\n', Style: DefaultStyle()},
			},
		},
		{
			name:    "single tab",
			content: Plain("\t"),
			want: []Box{
				{Text: nil, Nrune: -1, Bc: '\t', Style: DefaultStyle()},
			},
		},
		{
			name:    "text with newline",
			content: Plain("hello\nworld"),
			want: []Box{
				{Text: []byte("hello"), Nrune: 5, Bc: 0, Style: DefaultStyle()},
				{Text: nil, Nrune: -1, Bc: '\n', Style: DefaultStyle()},
				{Text: []byte("world"), Nrune: 5, Bc: 0, Style: DefaultStyle()},
			},
		},
		{
			name:    "text with tab",
			content: Plain("hello\tworld"),
			want: []Box{
				{Text: []byte("hello"), Nrune: 5, Bc: 0, Style: DefaultStyle()},
				{Text: nil, Nrune: -1, Bc: '\t', Style: DefaultStyle()},
				{Text: []byte("world"), Nrune: 5, Bc: 0, Style: DefaultStyle()},
			},
		},
		{
			name:    "multiple newlines",
			content: Plain("a\n\nb"),
			want: []Box{
				{Text: []byte("a"), Nrune: 1, Bc: 0, Style: DefaultStyle()},
				{Text: nil, Nrune: -1, Bc: '\n', Style: DefaultStyle()},
				{Text: nil, Nrune: -1, Bc: '\n', Style: DefaultStyle()},
				{Text: []byte("b"), Nrune: 1, Bc: 0, Style: DefaultStyle()},
			},
		},
		{
			name:    "trailing newline",
			content: Plain("hello\n"),
			want: []Box{
				{Text: []byte("hello"), Nrune: 5, Bc: 0, Style: DefaultStyle()},
				{Text: nil, Nrune: -1, Bc: '\n', Style: DefaultStyle()},
			},
		},
		{
			name:    "leading newline",
			content: Plain("\nhello"),
			want: []Box{
				{Text: nil, Nrune: -1, Bc: '\n', Style: DefaultStyle()},
				{Text: []byte("hello"), Nrune: 5, Bc: 0, Style: DefaultStyle()},
			},
		},
		{
			name: "styled span",
			content: Content{
				{Text: "bold", Style: StyleBold},
			},
			want: []Box{
				{Text: []byte("bold"), Nrune: 4, Bc: 0, Style: StyleBold},
			},
		},
		{
			name: "multiple styled spans",
			content: Content{
				{Text: "hello ", Style: DefaultStyle()},
				{Text: "world", Style: StyleBold},
			},
			want: []Box{
				{Text: []byte("hello "), Nrune: 6, Bc: 0, Style: DefaultStyle()},
				{Text: []byte("world"), Nrune: 5, Bc: 0, Style: StyleBold},
			},
		},
		{
			name: "styled span with newline",
			content: Content{
				{Text: "hello\n", Style: StyleBold},
				{Text: "world", Style: StyleItalic},
			},
			want: []Box{
				{Text: []byte("hello"), Nrune: 5, Bc: 0, Style: StyleBold},
				{Text: nil, Nrune: -1, Bc: '\n', Style: StyleBold},
				{Text: []byte("world"), Nrune: 5, Bc: 0, Style: StyleItalic},
			},
		},
		{
			name:    "unicode text",
			content: Plain("日本語"),
			want: []Box{
				{Text: []byte("日本語"), Nrune: 3, Bc: 0, Style: DefaultStyle()},
			},
		},
		{
			name:    "unicode with newline",
			content: Plain("日本\n語"),
			want: []Box{
				{Text: []byte("日本"), Nrune: 2, Bc: 0, Style: DefaultStyle()},
				{Text: nil, Nrune: -1, Bc: '\n', Style: DefaultStyle()},
				{Text: []byte("語"), Nrune: 1, Bc: 0, Style: DefaultStyle()},
			},
		},
		{
			name:    "mixed tabs and newlines",
			content: Plain("a\tb\nc\td"),
			want: []Box{
				{Text: []byte("a"), Nrune: 1, Bc: 0, Style: DefaultStyle()},
				{Text: nil, Nrune: -1, Bc: '\t', Style: DefaultStyle()},
				{Text: []byte("b"), Nrune: 1, Bc: 0, Style: DefaultStyle()},
				{Text: nil, Nrune: -1, Bc: '\n', Style: DefaultStyle()},
				{Text: []byte("c"), Nrune: 1, Bc: 0, Style: DefaultStyle()},
				{Text: nil, Nrune: -1, Bc: '\t', Style: DefaultStyle()},
				{Text: []byte("d"), Nrune: 1, Bc: 0, Style: DefaultStyle()},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contentToBoxes(tt.content)

			if len(got) != len(tt.want) {
				t.Fatalf("contentToBoxes() returned %d boxes, want %d\ngot: %v\nwant: %v",
					len(got), len(tt.want), formatBoxes(got), formatBoxes(tt.want))
			}

			for i := range got {
				if !boxesEqual(got[i], tt.want[i]) {
					t.Errorf("box[%d] = %v, want %v", i, formatBox(got[i]), formatBox(tt.want[i]))
				}
			}
		})
	}
}

// boxesEqual compares two boxes for equality.
func boxesEqual(a, b Box) bool {
	if string(a.Text) != string(b.Text) {
		return false
	}
	if a.Nrune != b.Nrune {
		return false
	}
	if a.Bc != b.Bc {
		return false
	}
	// Width is computed during layout, not during contentToBoxes
	// so we don't compare it here
	return stylesEqual(a.Style, b.Style)
}

// formatBox returns a string representation of a box for debugging.
func formatBox(b Box) string {
	if b.IsNewline() {
		return "{\\n}"
	}
	if b.IsTab() {
		return "{\\t}"
	}
	return "{" + string(b.Text) + "}"
}

// formatBoxes returns a string representation of boxes for debugging.
func formatBoxes(boxes []Box) string {
	result := "["
	for i, b := range boxes {
		if i > 0 {
			result += ", "
		}
		result += formatBox(b)
	}
	result += "]"
	return result
}

func TestBoxWidth(t *testing.T) {
	// Mock font with fixed character width of 10 pixels
	font := edwoodtest.NewFont(10, 14)

	tests := []struct {
		name string
		box  Box
		want int
	}{
		{
			name: "empty text box",
			box:  Box{Text: []byte{}, Nrune: 0, Bc: 0},
			want: 0,
		},
		{
			name: "single character",
			box:  Box{Text: []byte("a"), Nrune: 1, Bc: 0},
			want: 10, // 1 rune * 10 pixels
		},
		{
			name: "five characters",
			box:  Box{Text: []byte("hello"), Nrune: 5, Bc: 0},
			want: 50, // 5 runes * 10 pixels
		},
		{
			name: "unicode characters",
			box:  Box{Text: []byte("日本語"), Nrune: 3, Bc: 0},
			want: 30, // 3 runes * 10 pixels
		},
		{
			name: "newline box has zero width",
			box:  Box{Nrune: -1, Bc: '\n'},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := boxWidth(&tt.box, font)
			if got != tt.want {
				t.Errorf("boxWidth() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTabBoxWidth(t *testing.T) {
	maxtab := 80 // Tab stop every 80 pixels (8 characters * 10 pixels)

	tests := []struct {
		name   string
		xPos   int   // Current X position on the line
		minX   int   // Left edge of frame (for tab alignment)
		want   int   // Expected tab width
	}{
		{
			name: "tab at start of line",
			xPos: 0,
			minX: 0,
			want: 80, // Full tab width
		},
		{
			name: "tab after 1 character",
			xPos: 10,
			minX: 0,
			want: 70, // Align to next tab stop at 80
		},
		{
			name: "tab after half tab width",
			xPos: 40,
			minX: 0,
			want: 40, // Align to next tab stop at 80
		},
		{
			name: "tab near tab boundary",
			xPos: 75,
			minX: 0,
			want: 5, // Only 5 pixels to next tab stop
		},
		{
			name: "tab exactly at tab boundary",
			xPos: 80,
			minX: 0,
			want: 80, // Full tab width to next stop at 160
		},
		{
			name: "tab with non-zero frame origin",
			xPos: 110, // 10 pixels into frame + 100 pixels of text
			minX: 10,  // Frame starts at x=10
			want: 60,  // Tab stops are at 10, 90, 170... so next is 170, 170-110=60
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			box := Box{Nrune: -1, Bc: '\t'}
			got := tabBoxWidth(&box, tt.xPos, tt.minX, maxtab)
			if got != tt.want {
				t.Errorf("tabBoxWidth(xPos=%d, minX=%d, maxtab=%d) = %d, want %d",
					tt.xPos, tt.minX, maxtab, got, tt.want)
			}
		})
	}
}
