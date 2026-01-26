package rich

import (
	"fmt"
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
