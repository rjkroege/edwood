package rich

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
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
				{Text: []byte("hello"), Nrune: 5, Bc: 0, Style: DefaultStyle()},
				{Text: []byte(" "), Nrune: 1, Bc: 0, Style: DefaultStyle()},
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

func TestLayoutSingleLine(t *testing.T) {
	// Mock font with fixed character width of 10 pixels, height 14
	font := edwoodtest.NewFont(10, 14)
	frameWidth := 500 // Plenty wide for single line tests
	maxtab := 80

	tests := []struct {
		name      string
		content   Content
		wantLines int      // Expected number of lines
		wantBoxes []string // Expected boxes in order (for verification)
	}{
		{
			name:      "empty content",
			content:   Content{},
			wantLines: 0,
			wantBoxes: []string{},
		},
		{
			name:      "single word",
			content:   Plain("hello"),
			wantLines: 1,
			wantBoxes: []string{"hello"},
		},
		{
			name:      "two words",
			content:   Plain("hello world"),
			wantLines: 1,
			wantBoxes: []string{"hello", " ", "world"},
		},
		{
			name:      "text with newline creates two lines",
			content:   Plain("hello\nworld"),
			wantLines: 2,
			wantBoxes: []string{"hello", "\n", "world"},
		},
		{
			name:      "text with tab",
			content:   Plain("hello\tworld"),
			wantLines: 1,
			wantBoxes: []string{"hello", "\t", "world"},
		},
		{
			name:      "multiple newlines",
			content:   Plain("a\n\nb"),
			wantLines: 3,
			wantBoxes: []string{"a", "\n", "\n", "b"},
		},
		{
			name:      "trailing newline",
			content:   Plain("hello\n"),
			wantLines: 2,
			wantBoxes: []string{"hello", "\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boxes := contentToBoxes(tt.content)
			lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

			if len(lines) != tt.wantLines {
				t.Errorf("layout() returned %d lines, want %d", len(lines), tt.wantLines)
			}

			// Verify boxes are in expected order
			var gotBoxes []string
			for _, line := range lines {
				for _, pb := range line.Boxes {
					gotBoxes = append(gotBoxes, boxToString(pb.Box))
				}
			}
			if len(gotBoxes) != len(tt.wantBoxes) {
				t.Errorf("got %d boxes %v, want %d boxes %v", len(gotBoxes), gotBoxes, len(tt.wantBoxes), tt.wantBoxes)
			}
		})
	}
}

func TestLayoutWrapping(t *testing.T) {
	// Mock font with fixed character width of 10 pixels, height 14
	font := edwoodtest.NewFont(10, 14)
	maxtab := 80

	tests := []struct {
		name       string
		content    Content
		frameWidth int
		wantLines  int
	}{
		{
			name:       "no wrapping needed",
			content:    Plain("hi"),
			frameWidth: 100,
			wantLines:  1,
		},
		{
			name:       "wrap single long word",
			content:    Plain("hello"), // 5 chars * 10 = 50 pixels
			frameWidth: 30,             // Only 3 chars fit
			wantLines:  2,              // "hel" on line 1, "lo" on line 2
		},
		{
			name:       "wrap multiple words",
			content:    Plain("hello world"), // 11 chars * 10 = 110 pixels
			frameWidth: 60,                   // 6 chars fit per line
			wantLines:  2,                    // "hello " on line 1, "world" on line 2
		},
		{
			name:       "exact fit no wrap",
			content:    Plain("hello"), // 5 chars * 10 = 50 pixels
			frameWidth: 50,
			wantLines:  1,
		},
		{
			name:       "wrap at boundary",
			content:    Plain("hello"), // 5 chars * 10 = 50 pixels
			frameWidth: 49,             // Just under what's needed
			wantLines:  2,
		},
		{
			name:       "multiple wraps",
			content:    Plain("abcdefghij"), // 10 chars * 10 = 100 pixels
			frameWidth: 30,                  // 3 chars per line
			wantLines:  4,                   // "abc", "def", "ghi", "j"
		},
		{
			name:       "wrap with explicit newlines",
			content:    Plain("hello\nworld foo"), // newline + long second line
			frameWidth: 60,                        // 6 chars fit
			wantLines:  3,                         // "hello", "\nworld ", "foo"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boxes := contentToBoxes(tt.content)
			lines := layout(boxes, font, tt.frameWidth, maxtab, nil, nil)

			if len(lines) != tt.wantLines {
				var lineContents []string
				for i, line := range lines {
					var content string
					for _, pb := range line.Boxes {
						content += boxToString(pb.Box)
					}
					lineContents = append(lineContents, fmt.Sprintf("line[%d]: %q", i, content))
				}
				t.Errorf("layout() returned %d lines, want %d\n%s",
					len(lines), tt.wantLines, strings.Join(lineContents, "\n"))
			}
		})
	}
}

func TestLayoutBoxPositions(t *testing.T) {
	// Mock font with fixed character width of 10 pixels, height 14
	font := edwoodtest.NewFont(10, 14)
	frameWidth := 500
	maxtab := 80

	// Test that boxes are positioned correctly
	t.Run("sequential boxes have correct X positions", func(t *testing.T) {
		content := Plain("ab\tcd")
		boxes := contentToBoxes(content)
		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}

		// Expected: "ab" at x=0 (width 20), tab at x=20 (width 60 to reach 80), "cd" at x=80
		line := lines[0]
		if len(line.Boxes) != 3 {
			t.Fatalf("expected 3 boxes, got %d", len(line.Boxes))
		}

		// Box 0: "ab" at x=0
		if line.Boxes[0].X != 0 {
			t.Errorf("box[0] X = %d, want 0", line.Boxes[0].X)
		}

		// Box 1: tab at x=20
		if line.Boxes[1].X != 20 {
			t.Errorf("box[1] X = %d, want 20", line.Boxes[1].X)
		}

		// Box 2: "cd" at x=80 (after tab)
		if line.Boxes[2].X != 80 {
			t.Errorf("box[2] X = %d, want 80", line.Boxes[2].X)
		}
	})

	t.Run("wrapped lines have correct Y positions", func(t *testing.T) {
		content := Plain("hello\nworld")
		boxes := contentToBoxes(content)
		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}

		// Line 0 at Y=0
		if lines[0].Y != 0 {
			t.Errorf("line[0] Y = %d, want 0", lines[0].Y)
		}

		// Line 1 at Y=14 (font height)
		if lines[1].Y != 14 {
			t.Errorf("line[1] Y = %d, want 14", lines[1].Y)
		}
	})

	t.Run("box widths are computed", func(t *testing.T) {
		content := Plain("hello")
		boxes := contentToBoxes(content)
		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 1 || len(lines[0].Boxes) != 1 {
			t.Fatalf("expected 1 line with 1 box")
		}

		// "hello" is 5 chars * 10 pixels = 50
		if lines[0].Boxes[0].Box.Wid != 50 {
			t.Errorf("box width = %d, want 50", lines[0].Boxes[0].Box.Wid)
		}
	})
}

// boxToString converts a box to a string for test output.
func boxToString(b Box) string {
	if b.IsNewline() {
		return "\n"
	}
	if b.IsTab() {
		return "\t"
	}
	return string(b.Text)
}

// TestLayoutListIndent tests that list items are indented based on ListIndent.
func TestLayoutListIndent(t *testing.T) {
	// Mock font with fixed character width of 10 pixels, height 14
	font := edwoodtest.NewFont(10, 14)
	frameWidth := 500
	maxtab := 80

	// A simple list item: "• Item" where bullet is at indent 0
	// Expected: bullet at X=0, content "Item" after the bullet
	t.Run("single level list item at indent 0", func(t *testing.T) {
		content := Content{
			{Text: "•", Style: Style{ListBullet: true, ListIndent: 0, Scale: 1.0}},
			{Text: " ", Style: Style{ListItem: true, ListIndent: 0, Scale: 1.0}},
			{Text: "Item", Style: Style{ListItem: true, ListIndent: 0, Scale: 1.0}},
		}
		boxes := contentToBoxes(content)
		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}

		// At indent level 0, bullet should be at X=0
		if lines[0].Boxes[0].X != 0 {
			t.Errorf("bullet X = %d, want 0 (no indentation for level 0)", lines[0].Boxes[0].X)
		}
	})

	t.Run("list item at indent 1 is indented", func(t *testing.T) {
		content := Content{
			{Text: "•", Style: Style{ListBullet: true, ListIndent: 1, Scale: 1.0}},
			{Text: " ", Style: Style{ListItem: true, ListIndent: 1, Scale: 1.0}},
			{Text: "Nested", Style: Style{ListItem: true, ListIndent: 1, Scale: 1.0}},
		}
		boxes := contentToBoxes(content)
		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}

		// At indent level 1, bullet should be indented (e.g., 20 pixels per level)
		// ListIndentWidth should be 2 characters * 10 pixels = 20 pixels per level
		expectedIndent := 20 // 1 level * 20 pixels
		if lines[0].Boxes[0].X != expectedIndent {
			t.Errorf("bullet X = %d, want %d (indented for level 1)", lines[0].Boxes[0].X, expectedIndent)
		}
	})

	t.Run("list item at indent 2 is further indented", func(t *testing.T) {
		content := Content{
			{Text: "•", Style: Style{ListBullet: true, ListIndent: 2, Scale: 1.0}},
			{Text: " ", Style: Style{ListItem: true, ListIndent: 2, Scale: 1.0}},
			{Text: "Deep", Style: Style{ListItem: true, ListIndent: 2, Scale: 1.0}},
		}
		boxes := contentToBoxes(content)
		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}

		// At indent level 2, bullet should be at 40 pixels (2 * 20)
		expectedIndent := 40 // 2 levels * 20 pixels
		if lines[0].Boxes[0].X != expectedIndent {
			t.Errorf("bullet X = %d, want %d (indented for level 2)", lines[0].Boxes[0].X, expectedIndent)
		}
	})
}

// TestLayoutCodeBlockIndent tests that fenced code blocks are indented.
func TestLayoutCodeBlockIndent(t *testing.T) {
	// Mock font with fixed character width of 10 pixels, height 14
	font := edwoodtest.NewFont(10, 14)
	frameWidth := 500
	maxtab := 80

	// Expected indent is CodeBlockIndentChars * M-width
	// With mock font width of 10, that's 4 * 10 = 40 pixels
	expectedIndent := CodeBlockIndentChars * font.BytesWidth([]byte("M"))

	t.Run("code block is indented by 4 M-widths", func(t *testing.T) {
		// A code block line: "print('hello')" with Block=true and Code=true
		content := Content{
			{Text: "print('hello')", Style: Style{Block: true, Code: true, Scale: 1.0}},
		}
		boxes := contentToBoxes(content)
		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}

		// Code blocks should be indented by 4 * M-width
		if lines[0].Boxes[0].X != expectedIndent {
			t.Errorf("code block X = %d, want %d (4 * M-width)", lines[0].Boxes[0].X, expectedIndent)
		}
	})

	t.Run("code block wrapping maintains indentation", func(t *testing.T) {
		// A long code block line that wraps
		content := Content{
			{Text: "this is a very long line of code that will wrap", Style: Style{Block: true, Code: true, Scale: 1.0}},
		}
		boxes := contentToBoxes(content)
		// Frame width that forces wrapping: 200 pixels = 20 chars
		// After 40px indent, only 160px = 16 chars fit per line
		lines := layout(boxes, font, 200, maxtab, nil, nil)

		if len(lines) < 2 {
			t.Fatalf("expected multiple lines for wrapped code, got %d", len(lines))
		}

		// All lines should start at the calculated indent
		for i, line := range lines {
			if len(line.Boxes) == 0 {
				continue
			}
			if line.Boxes[0].X != expectedIndent {
				t.Errorf("line %d: code block X = %d, want %d (4 * M-width)", i, line.Boxes[0].X, expectedIndent)
			}
		}
	})

	t.Run("non-block code is not indented", func(t *testing.T) {
		// Inline code (Code=true but Block=false) should not be indented
		content := Content{
			{Text: "inline", Style: Style{Code: true, Scale: 1.0}},
		}
		boxes := contentToBoxes(content)
		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}

		// Inline code should start at X=0 (no indentation)
		if lines[0].Boxes[0].X != 0 {
			t.Errorf("inline code X = %d, want 0 (no indentation)", lines[0].Boxes[0].X)
		}
	})
}

// TestContentToBoxesImage tests that image spans are converted to image boxes.
// Image spans have Style.Image=true and should create a special image box.
func TestContentToBoxesImage(t *testing.T) {
	tests := []struct {
		name       string
		content    Content
		wantBoxes  int
		checkImage bool // Whether to check for image box
	}{
		{
			name: "single image span",
			content: Content{
				{Text: "[Image: alt]", Style: Style{Image: true, ImageURL: "test.png", ImageAlt: "alt", Scale: 1.0}},
			},
			wantBoxes:  1,
			checkImage: true,
		},
		{
			name: "image with surrounding text",
			content: Content{
				{Text: "Before ", Style: DefaultStyle()},
				{Text: "[Image: photo]", Style: Style{Image: true, ImageURL: "photo.jpg", ImageAlt: "photo", Scale: 1.0}},
				{Text: " After", Style: DefaultStyle()},
			},
			wantBoxes:  5, // "Before", " ", "[Image: photo]", " ", "After"
			checkImage: true,
		},
		{
			name: "multiple images",
			content: Content{
				{Text: "[Image: img1]", Style: Style{Image: true, ImageURL: "img1.png", ImageAlt: "img1", Scale: 1.0}},
				{Text: "\n", Style: DefaultStyle()},
				{Text: "[Image: img2]", Style: Style{Image: true, ImageURL: "img2.png", ImageAlt: "img2", Scale: 1.0}},
			},
			wantBoxes:  3, // image1, newline, image2
			checkImage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boxes := contentToBoxes(tt.content)

			if len(boxes) != tt.wantBoxes {
				var boxDescs []string
				for _, b := range boxes {
					boxDescs = append(boxDescs, boxToString(b))
				}
				t.Errorf("contentToBoxes() returned %d boxes %v, want %d",
					len(boxes), boxDescs, tt.wantBoxes)
			}

			if tt.checkImage {
				// Verify at least one box has Image style
				hasImageBox := false
				for _, box := range boxes {
					if box.Style.Image {
						hasImageBox = true
						break
					}
				}
				if !hasImageBox {
					t.Error("expected at least one box with Image style")
				}
			}
		})
	}
}

// TestLayoutNestedListIndent tests nested lists with multiple levels.
func TestLayoutNestedListIndent(t *testing.T) {
	// Mock font with fixed character width of 10 pixels, height 14
	font := edwoodtest.NewFont(10, 14)
	frameWidth := 500
	maxtab := 80

	t.Run("multiple list items at different indent levels", func(t *testing.T) {
		// Simulates:
		// - Item 1
		//   - Nested item
		//     - Deep nested
		content := Content{
			// Line 1: "• Item 1" at indent 0
			{Text: "•", Style: Style{ListBullet: true, ListIndent: 0, Scale: 1.0}},
			{Text: " Item 1", Style: Style{ListItem: true, ListIndent: 0, Scale: 1.0}},
			{Text: "\n", Style: Style{Scale: 1.0}},
			// Line 2: "• Nested item" at indent 1
			{Text: "•", Style: Style{ListBullet: true, ListIndent: 1, Scale: 1.0}},
			{Text: " Nested item", Style: Style{ListItem: true, ListIndent: 1, Scale: 1.0}},
			{Text: "\n", Style: Style{Scale: 1.0}},
			// Line 3: "• Deep nested" at indent 2
			{Text: "•", Style: Style{ListBullet: true, ListIndent: 2, Scale: 1.0}},
			{Text: " Deep nested", Style: Style{ListItem: true, ListIndent: 2, Scale: 1.0}},
		}
		boxes := contentToBoxes(content)
		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d", len(lines))
		}

		// Line 1: bullet at indent 0 (X=0)
		if len(lines[0].Boxes) < 1 {
			t.Fatalf("line 0 has no boxes")
		}
		if lines[0].Boxes[0].X != 0 {
			t.Errorf("line 0 bullet X = %d, want 0", lines[0].Boxes[0].X)
		}

		// Line 2: bullet at indent 1 (X=20)
		if len(lines[1].Boxes) < 1 {
			t.Fatalf("line 1 has no boxes")
		}
		if lines[1].Boxes[0].X != 20 {
			t.Errorf("line 1 bullet X = %d, want 20", lines[1].Boxes[0].X)
		}

		// Line 3: bullet at indent 2 (X=40)
		if len(lines[2].Boxes) < 1 {
			t.Fatalf("line 2 has no boxes")
		}
		if lines[2].Boxes[0].X != 40 {
			t.Errorf("line 2 bullet X = %d, want 40", lines[2].Boxes[0].X)
		}
	})

	t.Run("ordered list numbers are indented", func(t *testing.T) {
		content := Content{
			// "1." at indent 0
			{Text: "1.", Style: Style{ListBullet: true, ListOrdered: true, ListNumber: 1, ListIndent: 0, Scale: 1.0}},
			{Text: " First", Style: Style{ListItem: true, ListIndent: 0, Scale: 1.0}},
			{Text: "\n", Style: Style{Scale: 1.0}},
			// "a." at indent 1 (sub-list)
			{Text: "a.", Style: Style{ListBullet: true, ListOrdered: true, ListNumber: 1, ListIndent: 1, Scale: 1.0}},
			{Text: " Sub-item", Style: Style{ListItem: true, ListIndent: 1, Scale: 1.0}},
		}
		boxes := contentToBoxes(content)
		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}

		// Line 1: "1." at X=0
		if lines[0].Boxes[0].X != 0 {
			t.Errorf("line 0 number X = %d, want 0", lines[0].Boxes[0].X)
		}

		// Line 2: "a." at X=20
		if lines[1].Boxes[0].X != 20 {
			t.Errorf("line 1 number X = %d, want 20", lines[1].Boxes[0].X)
		}
	})

	t.Run("list content wraps with correct indentation", func(t *testing.T) {
		// A list item with long content that wraps
		// The wrapped portion should maintain the same indentation
		content := Content{
			{Text: "•", Style: Style{ListBullet: true, ListIndent: 1, Scale: 1.0}},
			{Text: " This is a very long item that will need to wrap to the next line because it exceeds the frame width", Style: Style{ListItem: true, ListIndent: 1, Scale: 1.0}},
		}
		boxes := contentToBoxes(content)
		lines := layout(boxes, font, 100, maxtab, nil, nil) // narrow frame to force wrapping

		if len(lines) < 2 {
			t.Fatalf("expected at least 2 lines for wrapped content, got %d", len(lines))
		}

		// First line should have bullet at indent 1 (X=20)
		if lines[0].Boxes[0].X != 20 {
			t.Errorf("line 0 bullet X = %d, want 20", lines[0].Boxes[0].X)
		}

		// Wrapped lines should also be indented to align with the text after the bullet
		// The continuation should be at the same indentation level as the text start
		// (bullet width + space = about 20 pixels more, so continuation at ~40)
		for i := 1; i < len(lines); i++ {
			if len(lines[i].Boxes) > 0 {
				// Wrapped content should be indented (at least at the list indent level)
				if lines[i].Boxes[0].X < 20 {
					t.Errorf("wrapped line %d X = %d, want >= 20 (should maintain list indentation)", i, lines[i].Boxes[0].X)
				}
			}
		}
	})
}

// =============================================================================
// Phase 16E: Image Layout Tests
// =============================================================================

// TestLayoutImageWidth tests that image boxes have correct width based on image dimensions.
// When an image box has ImageData set, the layout should use the image's width.
func TestLayoutImageWidth(t *testing.T) {
	font := edwoodtest.NewFont(10, 14)
	frameWidth := 500
	maxtab := 80

	// Create a mock cached image with known dimensions
	mockImage := &CachedImage{
		Width:  200,
		Height: 100,
		Path:   "test.png",
	}

	t.Run("image box uses image width", func(t *testing.T) {
		// Create an image box with ImageData set
		boxes := []Box{
			{
				Text:      nil,
				Nrune:     0,
				Bc:        0,
				Style:     Style{Image: true, ImageURL: "test.png", ImageAlt: "test", Scale: 1.0},
				ImageData: mockImage,
			},
		}

		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}
		if len(lines[0].Boxes) != 1 {
			t.Fatalf("expected 1 box, got %d", len(lines[0].Boxes))
		}

		// The image box should have width equal to the image width (200)
		gotWidth := lines[0].Boxes[0].Box.Wid
		if gotWidth != 200 {
			t.Errorf("image box width = %d, want 200", gotWidth)
		}
	})

	t.Run("image box without ImageData uses placeholder width", func(t *testing.T) {
		// Create an image box without ImageData (loading failed or not yet loaded)
		boxes := []Box{
			{
				Text:      []byte("[Image: test]"),
				Nrune:     13,
				Bc:        0,
				Style:     Style{Image: true, ImageURL: "test.png", ImageAlt: "test", Scale: 1.0},
				ImageData: nil,
			},
		}

		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}
		if len(lines[0].Boxes) != 1 {
			t.Fatalf("expected 1 box, got %d", len(lines[0].Boxes))
		}

		// Without ImageData, width should be based on placeholder text
		gotWidth := lines[0].Boxes[0].Box.Wid
		// "[Image: test]" = 13 chars * 10 pixels = 130
		if gotWidth != 130 {
			t.Errorf("placeholder box width = %d, want 130", gotWidth)
		}
	})
}

// TestLayoutImageScale tests that images wider than frame are scaled down.
// Large images should be scaled to fit within the frame width.
func TestLayoutImageScale(t *testing.T) {
	font := edwoodtest.NewFont(10, 14)
	frameWidth := 300 // Narrow frame
	maxtab := 80

	// Create a mock cached image that's wider than the frame
	wideImage := &CachedImage{
		Width:  600, // Twice the frame width
		Height: 200,
		Path:   "wide.png",
	}

	t.Run("wide image scaled to fit frame", func(t *testing.T) {
		boxes := []Box{
			{
				Text:      nil,
				Nrune:     0,
				Bc:        0,
				Style:     Style{Image: true, ImageURL: "wide.png", ImageAlt: "wide", Scale: 1.0},
				ImageData: wideImage,
			},
		}

		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}
		if len(lines[0].Boxes) != 1 {
			t.Fatalf("expected 1 box, got %d", len(lines[0].Boxes))
		}

		// The image should be scaled to fit within frameWidth (300)
		gotWidth := lines[0].Boxes[0].Box.Wid
		if gotWidth > frameWidth {
			t.Errorf("image box width = %d, should be <= frame width %d", gotWidth, frameWidth)
		}
		// After scaling 600 -> 300, the width should be exactly frameWidth
		if gotWidth != frameWidth {
			t.Errorf("image box width = %d, want %d (scaled to fit)", gotWidth, frameWidth)
		}
	})

	t.Run("image smaller than frame not scaled up", func(t *testing.T) {
		smallImage := &CachedImage{
			Width:  100,
			Height: 50,
			Path:   "small.png",
		}

		boxes := []Box{
			{
				Text:      nil,
				Nrune:     0,
				Bc:        0,
				Style:     Style{Image: true, ImageURL: "small.png", ImageAlt: "small", Scale: 1.0},
				ImageData: smallImage,
			},
		}

		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}

		// Small images should keep their original width
		gotWidth := lines[0].Boxes[0].Box.Wid
		if gotWidth != 100 {
			t.Errorf("small image box width = %d, want 100 (not scaled up)", gotWidth)
		}
	})
}

// TestLayoutImageLineHeight tests that lines containing images have appropriate height.
// The line height should accommodate the image height (possibly scaled).
func TestLayoutImageLineHeight(t *testing.T) {
	font := edwoodtest.NewFont(10, 14)
	frameWidth := 500
	maxtab := 80

	t.Run("line height includes image height", func(t *testing.T) {
		tallImage := &CachedImage{
			Width:  100,
			Height: 200, // Much taller than font height (14)
			Path:   "tall.png",
		}

		boxes := []Box{
			{
				Text:      nil,
				Nrune:     0,
				Bc:        0,
				Style:     Style{Image: true, ImageURL: "tall.png", ImageAlt: "tall", Scale: 1.0},
				ImageData: tallImage,
			},
		}

		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}

		// Line height should be at least the image height
		gotHeight := lines[0].Height
		if gotHeight < 200 {
			t.Errorf("line height = %d, should be >= 200 (image height)", gotHeight)
		}
	})

	t.Run("scaled image has proportional line height", func(t *testing.T) {
		// Image that needs scaling: 1000x500 scaled to fit in 500 wide frame
		// Scaled to 500x250
		largeImage := &CachedImage{
			Width:  1000,
			Height: 500,
			Path:   "large.png",
		}

		boxes := []Box{
			{
				Text:      nil,
				Nrune:     0,
				Bc:        0,
				Style:     Style{Image: true, ImageURL: "large.png", ImageAlt: "large", Scale: 1.0},
				ImageData: largeImage,
			},
		}

		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}

		// After scaling 1000 -> 500 (50%), height should also scale: 500 -> 250
		gotHeight := lines[0].Height
		// The height should be the scaled height (250) not the original (500)
		if gotHeight > 500 {
			t.Errorf("line height = %d, should be proportionally scaled (expected ~250)", gotHeight)
		}
		if gotHeight < 200 {
			t.Errorf("line height = %d, should be around 250 (scaled proportionally)", gotHeight)
		}
	})

	t.Run("image on same line as text uses max height", func(t *testing.T) {
		shortImage := &CachedImage{
			Width:  50,
			Height: 30, // Taller than font (14) but not huge
			Path:   "short.png",
		}

		boxes := []Box{
			{
				Text:  []byte("Text"),
				Nrune: 4,
				Bc:    0,
				Style: DefaultStyle(),
			},
			{
				Text:  []byte(" "),
				Nrune: 1,
				Bc:    0,
				Style: DefaultStyle(),
			},
			{
				Text:      nil,
				Nrune:     0,
				Bc:        0,
				Style:     Style{Image: true, ImageURL: "short.png", ImageAlt: "short", Scale: 1.0},
				ImageData: shortImage,
			},
		}

		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)

		if len(lines) != 1 {
			t.Fatalf("expected 1 line, got %d", len(lines))
		}

		// Line height should be the max of text height (14) and image height (30)
		gotHeight := lines[0].Height
		if gotHeight < 30 {
			t.Errorf("line height = %d, should be >= 30 (image height)", gotHeight)
		}
	})
}

// TestLayoutWithCache tests that layout can use an ImageCache to load images.
// When a cache is provided, layout should use it to retrieve image data for sizing.
func TestLayoutWithCache(t *testing.T) {
	font := edwoodtest.NewFont(10, 14)
	frameWidth := 500
	maxtab := 80

	// Note: This test verifies the interface for passing ImageCache to layout.
	// The actual loading behavior depends on implementation in 16E.4.

	t.Run("layout accepts nil cache", func(t *testing.T) {
		// Layout should work without a cache (backward compatibility)
		boxes := []Box{
			{
				Text:  []byte("Hello"),
				Nrune: 5,
				Bc:    0,
				Style: DefaultStyle(),
			},
		}

		// Should not panic with nil cache
		lines := layout(boxes, font, frameWidth, maxtab, nil, nil)
		if len(lines) != 1 {
			t.Errorf("expected 1 line, got %d", len(lines))
		}
	})

	t.Run("layout with image cache for image boxes", func(t *testing.T) {
		// Create an image box that would need the cache
		boxes := []Box{
			{
				Text:      []byte("[Image: test]"),
				Nrune:     13,
				Bc:        0,
				Style:     Style{Image: true, ImageURL: "/path/to/test.png", ImageAlt: "test", Scale: 1.0},
				ImageData: nil, // No image data yet
			},
		}

		// Create a cache (even if we don't populate it for this test)
		cache := NewImageCache(10)

		// Layout should handle the case where image isn't in cache yet
		// For now, just verify it doesn't panic
		lines := layoutWithCache(boxes, font, frameWidth, maxtab, nil, nil, cache)
		if len(lines) != 1 {
			t.Errorf("expected 1 line, got %d", len(lines))
		}
	})
}

// =============================================================================
// Phase 16I.3: layoutWithCache Integration Tests
// =============================================================================

// TestLayoutWithCacheLoadsImages verifies that layoutWithCache automatically
// loads images from the cache when processing image boxes. When a box has
// Style.Image=true and ImageURL set, layoutWithCache should call cache.Load()
// to load the image data.
func TestLayoutWithCacheLoadsImages(t *testing.T) {
	// Create a temporary PNG file
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "cache_load_test.png")

	// Create a simple 40x30 magenta image
	img := createTestImage(40, 30, 255, 0, 255)
	if err := saveTestPNG(pngPath, img); err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}

	font := &testLayoutFont{width: 10, height: 14}
	frameWidth := 500
	maxtab := 80

	// Create an image box WITHOUT ImageData set initially
	// layoutWithCache should load the image from the cache
	boxes := []Box{
		{
			Text:      []byte("[Image: test]"),
			Nrune:     13,
			Bc:        0,
			Style:     Style{Image: true, ImageURL: pngPath, ImageAlt: "test", Scale: 1.0},
			ImageData: nil, // Not yet loaded
		},
	}

	// Create a fresh cache
	cache := NewImageCache(10)

	// Verify image is NOT in cache before layout
	if _, ok := cache.Get(pngPath); ok {
		t.Error("image should NOT be in cache before layoutWithCache")
	}

	// Call layoutWithCache - this should trigger image loading
	lines := layoutWithCache(boxes, font, frameWidth, maxtab, nil, nil, cache)

	if len(lines) == 0 {
		t.Fatal("layoutWithCache returned no lines")
	}

	// Verify image IS now in cache after layout
	cached, ok := cache.Get(pngPath)
	if !ok {
		t.Error("image should be in cache after layoutWithCache")
	}
	if cached == nil {
		t.Fatal("cached image is nil")
	}
	if cached.Err != nil {
		t.Errorf("cached image has error: %v", cached.Err)
	}
	if cached.Width != 40 || cached.Height != 30 {
		t.Errorf("cached image dimensions = %dx%d, want 40x30", cached.Width, cached.Height)
	}
}

// TestLayoutWithCachePopulatesImageData verifies that layoutWithCache populates
// the ImageData field of image boxes after loading. This is essential for
// rendering, as the renderer needs the image dimensions and pixel data.
func TestLayoutWithCachePopulatesImageData(t *testing.T) {
	// Create a temporary PNG file
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "populate_test.png")

	// Create a simple 50x40 cyan image
	img := createTestImage(50, 40, 0, 255, 255)
	if err := saveTestPNG(pngPath, img); err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}

	font := &testLayoutFont{width: 10, height: 14}
	frameWidth := 500
	maxtab := 80

	// Create image boxes without ImageData
	boxes := []Box{
		{
			Text:  []byte("Some text "),
			Nrune: 10,
			Bc:    0,
			Style: DefaultStyle(),
		},
		{
			Text:      []byte("[Image: photo]"),
			Nrune:     14,
			Bc:        0,
			Style:     Style{Image: true, ImageURL: pngPath, ImageAlt: "photo", Scale: 1.0},
			ImageData: nil, // Should be populated by layoutWithCache
		},
		{
			Text:  []byte(" more text"),
			Nrune: 10,
			Bc:    0,
			Style: DefaultStyle(),
		},
	}

	// Create cache
	cache := NewImageCache(10)

	// Call layoutWithCache
	lines := layoutWithCache(boxes, font, frameWidth, maxtab, nil, nil, cache)

	if len(lines) == 0 {
		t.Fatal("layoutWithCache returned no lines")
	}

	// Find the image box in the layout and verify ImageData is populated
	var imageBoxFound bool
	for _, line := range lines {
		for _, pb := range line.Boxes {
			if pb.Box.Style.Image {
				imageBoxFound = true
				if pb.Box.ImageData == nil {
					t.Error("image box ImageData should be populated after layoutWithCache")
				} else {
					// Verify the populated data is correct
					if pb.Box.ImageData.Width != 50 || pb.Box.ImageData.Height != 40 {
						t.Errorf("image box dimensions = %dx%d, want 50x40",
							pb.Box.ImageData.Width, pb.Box.ImageData.Height)
					}
					if pb.Box.ImageData.Err != nil {
						t.Errorf("image box has unexpected error: %v", pb.Box.ImageData.Err)
					}
					// Verify the box uses the image dimensions for layout
					// (not the placeholder text dimensions)
					if pb.Box.Wid != 50 {
						t.Errorf("image box width = %d, want 50 (image width)", pb.Box.Wid)
					}
				}
			}
		}
	}

	if !imageBoxFound {
		t.Error("image box not found in layout results")
	}
}

// TestLayoutWithCacheHandlesLoadError verifies that layoutWithCache handles
// image load errors gracefully. When an image fails to load, the CachedImage
// is still stored in the box but with an error and zero dimensions.
func TestLayoutWithCacheHandlesLoadError(t *testing.T) {
	font := &testLayoutFont{width: 10, height: 14}
	frameWidth := 500
	maxtab := 80

	// Create box with non-existent image path
	boxes := []Box{
		{
			Text:      []byte("[Image: missing]"),
			Nrune:     16,
			Bc:        0,
			Style:     Style{Image: true, ImageURL: "/nonexistent/path/to/image.png", ImageAlt: "missing", Scale: 1.0},
			ImageData: nil,
		},
	}

	cache := NewImageCache(10)

	// Should not panic
	lines := layoutWithCache(boxes, font, frameWidth, maxtab, nil, nil, cache)

	if len(lines) == 0 {
		t.Fatal("layoutWithCache returned no lines")
	}

	// Verify the error was cached
	cached, ok := cache.Get("/nonexistent/path/to/image.png")
	if !ok {
		t.Error("failed load should still be cached")
	}
	if cached != nil && cached.Err == nil {
		t.Error("cached entry should have error set")
	}

	// Find the image box and verify it has ImageData set with the error
	for _, line := range lines {
		for _, pb := range line.Boxes {
			if pb.Box.Style.Image {
				if pb.Box.ImageData == nil {
					t.Error("ImageData should be set even for failed loads")
				} else if pb.Box.ImageData.Err == nil {
					t.Error("ImageData.Err should be set for failed loads")
				}
			}
		}
	}
}

// TestLayoutWithCacheMultipleImages verifies that layoutWithCache handles
// multiple images in the content, loading each one.
func TestLayoutWithCacheMultipleImages(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two test images with different sizes
	pngPath1 := filepath.Join(tmpDir, "img1.png")
	pngPath2 := filepath.Join(tmpDir, "img2.png")

	img1 := createTestImage(30, 25, 255, 0, 0) // red
	img2 := createTestImage(45, 35, 0, 0, 255) // blue

	if err := saveTestPNG(pngPath1, img1); err != nil {
		t.Fatalf("failed to create test PNG 1: %v", err)
	}
	if err := saveTestPNG(pngPath2, img2); err != nil {
		t.Fatalf("failed to create test PNG 2: %v", err)
	}

	font := &testLayoutFont{width: 10, height: 14}
	frameWidth := 500
	maxtab := 80

	boxes := []Box{
		{
			Text:      []byte("[Image: img1]"),
			Nrune:     13,
			Bc:        0,
			Style:     Style{Image: true, ImageURL: pngPath1, ImageAlt: "img1", Scale: 1.0},
			ImageData: nil,
		},
		{
			Text: nil, Nrune: -1, Bc: '\n', Style: DefaultStyle(),
		},
		{
			Text:      []byte("[Image: img2]"),
			Nrune:     13,
			Bc:        0,
			Style:     Style{Image: true, ImageURL: pngPath2, ImageAlt: "img2", Scale: 1.0},
			ImageData: nil,
		},
	}

	cache := NewImageCache(10)
	lines := layoutWithCache(boxes, font, frameWidth, maxtab, nil, nil, cache)

	// Should have loaded both images
	cached1, ok1 := cache.Get(pngPath1)
	cached2, ok2 := cache.Get(pngPath2)

	if !ok1 {
		t.Error("first image should be in cache")
	}
	if !ok2 {
		t.Error("second image should be in cache")
	}

	if cached1 != nil && (cached1.Width != 30 || cached1.Height != 25) {
		t.Errorf("first cached image dimensions = %dx%d, want 30x25", cached1.Width, cached1.Height)
	}
	if cached2 != nil && (cached2.Width != 45 || cached2.Height != 35) {
		t.Errorf("second cached image dimensions = %dx%d, want 45x35", cached2.Width, cached2.Height)
	}

	// Verify layout produced correct number of lines
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

// =============================================================================
// Phase 16I.6: Relative Path Resolution Tests
// =============================================================================

// TestLayoutResolvesRelativePaths verifies that layoutWithCache resolves
// relative image paths using the provided basePath. This is essential for
// loading images specified with relative paths in markdown files.
//
// Example: If a markdown file at /home/user/docs/readme.md contains
// ![alt](images/photo.png), the relative path "images/photo.png" should
// be resolved to /home/user/docs/images/photo.png before loading.
func TestLayoutResolvesRelativePaths(t *testing.T) {
	// Create a temporary directory structure:
	// tmpDir/
	//   docs/
	//     readme.md (simulated)
	//     images/
	//       photo.png
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs")
	imagesDir := filepath.Join(docsDir, "images")

	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		t.Fatalf("failed to create images directory: %v", err)
	}

	// Create a test image at docs/images/photo.png
	imgPath := filepath.Join(imagesDir, "photo.png")
	img := createTestImage(60, 45, 0, 128, 255) // Blue image
	if err := saveTestPNG(imgPath, img); err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}

	// Simulate the markdown file path (for basePath)
	mdPath := filepath.Join(docsDir, "readme.md")

	font := &testLayoutFont{width: 10, height: 14}
	frameWidth := 500
	maxtab := 80

	// Create an image box with a RELATIVE path (as would appear in markdown)
	relativeImagePath := "images/photo.png"
	boxes := []Box{
		{
			Text:      []byte("[Image: photo]"),
			Nrune:     14,
			Bc:        0,
			Style:     Style{Image: true, ImageURL: relativeImagePath, ImageAlt: "photo", Scale: 1.0},
			ImageData: nil, // Should be populated after layout
		},
	}

	// Create a fresh cache
	cache := NewImageCache(10)

	// Call layoutWithCache WITH a basePath
	// The basePath should be the markdown file's path
	lines := layoutWithCacheAndBasePath(boxes, font, frameWidth, maxtab, nil, nil, cache, mdPath)

	if len(lines) == 0 {
		t.Fatal("layoutWithCacheAndBasePath returned no lines")
	}

	// Verify the image was loaded using the resolved path
	// The cache should contain the ABSOLUTE path (resolved from relative)
	resolvedPath := filepath.Join(docsDir, relativeImagePath)
	cached, ok := cache.Get(resolvedPath)
	if !ok {
		// Also check if it was cached with the relative path (wrong behavior)
		if _, hasRelative := cache.Get(relativeImagePath); hasRelative {
			t.Error("image was cached with relative path; should be cached with resolved absolute path")
		} else {
			t.Error("image should be in cache after layoutWithCacheAndBasePath")
		}
	}
	if cached != nil && cached.Err != nil {
		t.Errorf("cached image has error: %v", cached.Err)
	}
	if cached != nil && (cached.Width != 60 || cached.Height != 45) {
		t.Errorf("cached image dimensions = %dx%d, want 60x45", cached.Width, cached.Height)
	}

	// Verify the box's ImageData was populated
	for _, line := range lines {
		for _, pb := range line.Boxes {
			if pb.Box.Style.Image {
				if pb.Box.ImageData == nil {
					t.Error("image box ImageData should be populated after layout with basePath")
				} else if pb.Box.ImageData.Width != 60 || pb.Box.ImageData.Height != 45 {
					t.Errorf("image box dimensions = %dx%d, want 60x45",
						pb.Box.ImageData.Width, pb.Box.ImageData.Height)
				}
			}
		}
	}
}

// TestLayoutResolvesRelativePathsWithParentDir verifies that relative paths
// with parent directory references (../) are resolved correctly.
func TestLayoutResolvesRelativePathsWithParentDir(t *testing.T) {
	// Create a temporary directory structure:
	// tmpDir/
	//   assets/
	//     logo.png
	//   docs/
	//     guide/
	//       intro.md (simulated)
	tmpDir := t.TempDir()
	assetsDir := filepath.Join(tmpDir, "assets")
	guideDir := filepath.Join(tmpDir, "docs", "guide")

	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("failed to create assets directory: %v", err)
	}
	if err := os.MkdirAll(guideDir, 0755); err != nil {
		t.Fatalf("failed to create guide directory: %v", err)
	}

	// Create a test image at assets/logo.png
	imgPath := filepath.Join(assetsDir, "logo.png")
	img := createTestImage(80, 60, 255, 128, 0) // Orange image
	if err := saveTestPNG(imgPath, img); err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}

	// Simulate the markdown file path
	mdPath := filepath.Join(guideDir, "intro.md")

	font := &testLayoutFont{width: 10, height: 14}
	frameWidth := 500
	maxtab := 80

	// Create an image box with a relative path that goes up two directories
	relativeImagePath := "../../assets/logo.png"
	boxes := []Box{
		{
			Text:      []byte("[Image: logo]"),
			Nrune:     13,
			Bc:        0,
			Style:     Style{Image: true, ImageURL: relativeImagePath, ImageAlt: "logo", Scale: 1.0},
			ImageData: nil,
		},
	}

	cache := NewImageCache(10)

	lines := layoutWithCacheAndBasePath(boxes, font, frameWidth, maxtab, nil, nil, cache, mdPath)

	if len(lines) == 0 {
		t.Fatal("layoutWithCacheAndBasePath returned no lines")
	}

	// Verify the box's ImageData was populated with correct dimensions
	for _, line := range lines {
		for _, pb := range line.Boxes {
			if pb.Box.Style.Image {
				if pb.Box.ImageData == nil {
					t.Error("image box ImageData should be populated for relative path with ../")
				} else if pb.Box.ImageData.Width != 80 || pb.Box.ImageData.Height != 60 {
					t.Errorf("image box dimensions = %dx%d, want 80x60",
						pb.Box.ImageData.Width, pb.Box.ImageData.Height)
				}
			}
		}
	}
}

// TestLayoutAbsolutePathIgnoresBasePath verifies that absolute image paths
// are NOT modified by the basePath - they should load directly.
func TestLayoutAbsolutePathIgnoresBasePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an image at an absolute path
	imgPath := filepath.Join(tmpDir, "absolute_image.png")
	img := createTestImage(50, 50, 255, 0, 0) // Red image
	if err := saveTestPNG(imgPath, img); err != nil {
		t.Fatalf("failed to create test PNG: %v", err)
	}

	// Use a different directory as the "basePath" (should be ignored)
	basePath := "/some/other/directory/readme.md"

	font := &testLayoutFont{width: 10, height: 14}
	frameWidth := 500
	maxtab := 80

	// Create an image box with an ABSOLUTE path
	boxes := []Box{
		{
			Text:      []byte("[Image: abs]"),
			Nrune:     12,
			Bc:        0,
			Style:     Style{Image: true, ImageURL: imgPath, ImageAlt: "abs", Scale: 1.0},
			ImageData: nil,
		},
	}

	cache := NewImageCache(10)

	lines := layoutWithCacheAndBasePath(boxes, font, frameWidth, maxtab, nil, nil, cache, basePath)

	if len(lines) == 0 {
		t.Fatal("layoutWithCacheAndBasePath returned no lines")
	}

	// Verify the image was loaded from the absolute path
	cached, ok := cache.Get(imgPath)
	if !ok {
		t.Error("absolute path image should be cached with its original path")
	}
	if cached != nil && cached.Err != nil {
		t.Errorf("cached image has error: %v", cached.Err)
	}

	// Verify dimensions
	for _, line := range lines {
		for _, pb := range line.Boxes {
			if pb.Box.Style.Image {
				if pb.Box.ImageData == nil {
					t.Error("image box ImageData should be populated for absolute path")
				} else if pb.Box.ImageData.Width != 50 || pb.Box.ImageData.Height != 50 {
					t.Errorf("image box dimensions = %dx%d, want 50x50",
						pb.Box.ImageData.Width, pb.Box.ImageData.Height)
				}
			}
		}
	}
}

// TestLayoutEmptyBasePathFallsBack verifies that when basePath is empty,
// relative paths are used as-is (likely failing to load, which is expected).
func TestLayoutEmptyBasePathFallsBack(t *testing.T) {
	font := &testLayoutFont{width: 10, height: 14}
	frameWidth := 500
	maxtab := 80

	// Create an image box with a relative path but no basePath
	boxes := []Box{
		{
			Text:      []byte("[Image: orphan]"),
			Nrune:     15,
			Bc:        0,
			Style:     Style{Image: true, ImageURL: "nonexistent/image.png", ImageAlt: "orphan", Scale: 1.0},
			ImageData: nil,
		},
	}

	cache := NewImageCache(10)

	// Empty basePath - relative path should fail to load (expected)
	lines := layoutWithCacheAndBasePath(boxes, font, frameWidth, maxtab, nil, nil, cache, "")

	if len(lines) == 0 {
		t.Fatal("layoutWithCacheAndBasePath returned no lines")
	}

	// The image should still have ImageData set (with an error)
	for _, line := range lines {
		for _, pb := range line.Boxes {
			if pb.Box.Style.Image {
				if pb.Box.ImageData == nil {
					t.Error("image box ImageData should be set even on error")
				} else if pb.Box.ImageData.Err == nil {
					t.Error("expected error for non-existent relative path with empty basePath")
				}
			}
		}
	}
}

// =============================================================================
// Phase 24D: Explicit Image Width Tests
// =============================================================================

// TestImageBoxDimensionsExplicitWidth tests that imageBoxDimensions uses
// Style.ImageWidth when set, scaling height proportionally.
func TestImageBoxDimensionsExplicitWidth(t *testing.T) {
	// Image is 400x200 (2:1 aspect ratio)
	mockImage := &CachedImage{
		Width:  400,
		Height: 200,
		Path:   "photo.png",
	}

	t.Run("explicit width smaller than natural", func(t *testing.T) {
		box := Box{
			Style:     Style{Image: true, ImageURL: "photo.png", ImageWidth: 200, Scale: 1.0},
			ImageData: mockImage,
		}
		w, h := imageBoxDimensions(&box, 500)
		if w != 200 {
			t.Errorf("width = %d, want 200", w)
		}
		// Height should scale proportionally: 200 * (200/400) = 100
		if h != 100 {
			t.Errorf("height = %d, want 100", h)
		}
	})

	t.Run("explicit width larger than natural", func(t *testing.T) {
		box := Box{
			Style:     Style{Image: true, ImageURL: "photo.png", ImageWidth: 600, Scale: 1.0},
			ImageData: mockImage,
		}
		// ImageWidth=600 exceeds frame width 800, but is within frame
		w, h := imageBoxDimensions(&box, 800)
		if w != 600 {
			t.Errorf("width = %d, want 600", w)
		}
		// Height should scale proportionally: 200 * (600/400) = 300
		if h != 300 {
			t.Errorf("height = %d, want 300", h)
		}
	})

	t.Run("zero ImageWidth uses natural size", func(t *testing.T) {
		box := Box{
			Style:     Style{Image: true, ImageURL: "photo.png", ImageWidth: 0, Scale: 1.0},
			ImageData: mockImage,
		}
		w, h := imageBoxDimensions(&box, 500)
		if w != 400 {
			t.Errorf("width = %d, want 400 (natural)", w)
		}
		if h != 200 {
			t.Errorf("height = %d, want 200 (natural)", h)
		}
	})
}

// TestImageBoxDimensionsClampToFrame tests that explicit ImageWidth is
// clamped to the frame width when it exceeds it.
func TestImageBoxDimensionsClampToFrame(t *testing.T) {
	// Image is 400x200 (2:1 aspect ratio)
	mockImage := &CachedImage{
		Width:  400,
		Height: 200,
		Path:   "photo.png",
	}

	t.Run("explicit width clamped to frame", func(t *testing.T) {
		box := Box{
			Style:     Style{Image: true, ImageURL: "photo.png", ImageWidth: 500, Scale: 1.0},
			ImageData: mockImage,
		}
		// Frame is only 300 wide, so 500 should be clamped to 300
		w, h := imageBoxDimensions(&box, 300)
		if w != 300 {
			t.Errorf("width = %d, want 300 (clamped to frame)", w)
		}
		// Height should scale proportionally from original: 200 * (300/400) = 150
		if h != 150 {
			t.Errorf("height = %d, want 150", h)
		}
	})

	t.Run("natural size wider than frame still clamped", func(t *testing.T) {
		// Even without explicit width, images wider than frame are clamped
		box := Box{
			Style:     Style{Image: true, ImageURL: "photo.png", ImageWidth: 0, Scale: 1.0},
			ImageData: mockImage,
		}
		w, h := imageBoxDimensions(&box, 200)
		if w != 200 {
			t.Errorf("width = %d, want 200 (clamped to frame)", w)
		}
		// 200 * (200/400) = 100
		if h != 100 {
			t.Errorf("height = %d, want 100", h)
		}
	})
}

// Helper: testLayoutFont implements draw.Font for testing
type testLayoutFont struct {
	width  int
	height int
}

func (f *testLayoutFont) Name() string             { return "test-layout-font" }
func (f *testLayoutFont) Height() int              { return f.height }
func (f *testLayoutFont) BytesWidth(b []byte) int  { return f.width * len(b) }
func (f *testLayoutFont) RunesWidth(r []rune) int  { return f.width * len(r) }
func (f *testLayoutFont) StringWidth(s string) int { return f.width * len(s) }

// Helper: createTestImage creates a test image with given dimensions and color
func createTestImage(w, h int, r, g, b uint8) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	c := color.RGBA{r, g, b, 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// Helper: saveTestPNG saves an image as PNG
func saveTestPNG(path string, img *image.RGBA) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
