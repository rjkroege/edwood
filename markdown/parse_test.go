package markdown

import (
	"testing"

	"github.com/rjkroege/edwood/rich"
)

func TestParsePlainText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantText string
	}{
		{
			name:     "empty string",
			input:    "",
			wantLen:  0,
			wantText: "",
		},
		{
			name:     "simple text",
			input:    "Hello, World!",
			wantLen:  1,
			wantText: "Hello, World!",
		},
		{
			name:     "multiline text",
			input:    "Line one\nLine two\nLine three",
			wantLen:  1,
			wantText: "Line one\nLine two\nLine three",
		},
		{
			name:     "text with spaces",
			input:    "  some   spaced   text  ",
			wantLen:  1,
			wantText: "  some   spaced   text  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)

			if len(got) != tt.wantLen {
				t.Errorf("Parse(%q) returned %d spans, want %d", tt.input, len(got), tt.wantLen)
				return
			}

			if tt.wantLen == 0 {
				return
			}

			// For plain text, should be default style
			if got[0].Style != rich.DefaultStyle() {
				t.Errorf("Parse(%q) style = %+v, want DefaultStyle()", tt.input, got[0].Style)
			}

			if got[0].Text != tt.wantText {
				t.Errorf("Parse(%q) text = %q, want %q", tt.input, got[0].Text, tt.wantText)
			}
		})
	}
}

func TestParseH1(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantText  string
		wantScale float64
		wantBold  bool
	}{
		{
			name:      "simple h1",
			input:     "# Heading",
			wantText:  "Heading",
			wantScale: 2.0,
			wantBold:  true,
		},
		{
			name:      "h1 with extra spaces after hash",
			input:     "#  Heading",
			wantText:  "Heading",
			wantScale: 2.0,
			wantBold:  true,
		},
		{
			name:      "h1 with trailing newline",
			input:     "# Heading\n",
			wantText:  "Heading\n",
			wantScale: 2.0,
			wantBold:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if len(got) == 0 {
				t.Fatal("Parse returned empty content")
			}
			if got[0].Text != tt.wantText {
				t.Errorf("text = %q, want %q", got[0].Text, tt.wantText)
			}
			if got[0].Style.Scale != tt.wantScale {
				t.Errorf("scale = %v, want %v", got[0].Style.Scale, tt.wantScale)
			}
			if got[0].Style.Bold != tt.wantBold {
				t.Errorf("bold = %v, want %v", got[0].Style.Bold, tt.wantBold)
			}
		})
	}
}

func TestParseH2(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantText  string
		wantScale float64
		wantBold  bool
	}{
		{
			name:      "simple h2",
			input:     "## Heading",
			wantText:  "Heading",
			wantScale: 1.5,
			wantBold:  true,
		},
		{
			name:      "h2 with extra spaces",
			input:     "##  Heading",
			wantText:  "Heading",
			wantScale: 1.5,
			wantBold:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if len(got) == 0 {
				t.Fatal("Parse returned empty content")
			}
			if got[0].Text != tt.wantText {
				t.Errorf("text = %q, want %q", got[0].Text, tt.wantText)
			}
			if got[0].Style.Scale != tt.wantScale {
				t.Errorf("scale = %v, want %v", got[0].Style.Scale, tt.wantScale)
			}
			if got[0].Style.Bold != tt.wantBold {
				t.Errorf("bold = %v, want %v", got[0].Style.Bold, tt.wantBold)
			}
		})
	}
}

func TestParseH3(t *testing.T) {
	got := Parse("### Heading")
	if len(got) == 0 {
		t.Fatal("Parse returned empty content")
	}
	if got[0].Text != "Heading" {
		t.Errorf("text = %q, want %q", got[0].Text, "Heading")
	}
	if got[0].Style.Scale != 1.25 {
		t.Errorf("scale = %v, want %v", got[0].Style.Scale, 1.25)
	}
	if !got[0].Style.Bold {
		t.Error("bold = false, want true")
	}
}

func TestParseH4(t *testing.T) {
	got := Parse("#### Heading")
	if len(got) == 0 {
		t.Fatal("Parse returned empty content")
	}
	if got[0].Text != "Heading" {
		t.Errorf("text = %q, want %q", got[0].Text, "Heading")
	}
	if got[0].Style.Scale != 1.125 {
		t.Errorf("scale = %v, want %v", got[0].Style.Scale, 1.125)
	}
	if !got[0].Style.Bold {
		t.Error("bold = false, want true")
	}
}

func TestParseH5(t *testing.T) {
	got := Parse("##### Heading")
	if len(got) == 0 {
		t.Fatal("Parse returned empty content")
	}
	if got[0].Text != "Heading" {
		t.Errorf("text = %q, want %q", got[0].Text, "Heading")
	}
	if got[0].Style.Scale != 1.0 {
		t.Errorf("scale = %v, want %v", got[0].Style.Scale, 1.0)
	}
	if !got[0].Style.Bold {
		t.Error("bold = false, want true")
	}
}

func TestParseH6(t *testing.T) {
	got := Parse("###### Heading")
	if len(got) == 0 {
		t.Fatal("Parse returned empty content")
	}
	if got[0].Text != "Heading" {
		t.Errorf("text = %q, want %q", got[0].Text, "Heading")
	}
	if got[0].Style.Scale != 0.875 {
		t.Errorf("scale = %v, want %v", got[0].Style.Scale, 0.875)
	}
	if !got[0].Style.Bold {
		t.Error("bold = false, want true")
	}
}

func TestParseHeadingMixed(t *testing.T) {
	// Test document with heading followed by plain text
	input := "# Title\n\nSome body text."
	got := Parse(input)

	// Should have at least 2 spans: heading and body
	if len(got) < 2 {
		t.Fatalf("expected at least 2 spans, got %d", len(got))
	}

	// First span should be heading
	if got[0].Style.Scale != 2.0 {
		t.Errorf("heading scale = %v, want 2.0", got[0].Style.Scale)
	}

	// Later span should be body text with default scale
	foundBody := false
	for _, span := range got[1:] {
		if span.Style.Scale == 1.0 && !span.Style.Bold {
			foundBody = true
			break
		}
	}
	if !foundBody {
		t.Error("expected body text with default style")
	}
}

func TestParseHeadingNotAtLineStart(t *testing.T) {
	// Hash in middle of line should not be a heading
	input := "Some text # not a heading"
	got := Parse(input)
	if len(got) == 0 {
		t.Fatal("Parse returned empty content")
	}
	// Should be plain text, not heading style
	if got[0].Style.Scale != 1.0 {
		t.Errorf("scale = %v, want 1.0 (plain text)", got[0].Style.Scale)
	}
	if got[0].Style.Bold {
		t.Error("should not be bold for non-heading")
	}
}
