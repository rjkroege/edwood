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

func TestParseBold(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSpan []struct {
			text   string
			bold   bool
			italic bool
		}
	}{
		{
			name:  "simple bold",
			input: "**bold text**",
			wantSpan: []struct {
				text   string
				bold   bool
				italic bool
			}{
				{text: "bold text", bold: true, italic: false},
			},
		},
		{
			name:  "bold in middle of text",
			input: "some **bold** text",
			wantSpan: []struct {
				text   string
				bold   bool
				italic bool
			}{
				{text: "some ", bold: false, italic: false},
				{text: "bold", bold: true, italic: false},
				{text: " text", bold: false, italic: false},
			},
		},
		{
			name:  "multiple bold sections",
			input: "**one** and **two**",
			wantSpan: []struct {
				text   string
				bold   bool
				italic bool
			}{
				{text: "one", bold: true, italic: false},
				{text: " and ", bold: false, italic: false},
				{text: "two", bold: true, italic: false},
			},
		},
		{
			name:  "unclosed bold treated as plain",
			input: "**unclosed",
			wantSpan: []struct {
				text   string
				bold   bool
				italic bool
			}{
				{text: "**unclosed", bold: false, italic: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if len(got) != len(tt.wantSpan) {
				t.Fatalf("got %d spans, want %d spans\n  got: %+v", len(got), len(tt.wantSpan), got)
			}
			for i, want := range tt.wantSpan {
				if got[i].Text != want.text {
					t.Errorf("span[%d].Text = %q, want %q", i, got[i].Text, want.text)
				}
				if got[i].Style.Bold != want.bold {
					t.Errorf("span[%d].Bold = %v, want %v", i, got[i].Style.Bold, want.bold)
				}
				if got[i].Style.Italic != want.italic {
					t.Errorf("span[%d].Italic = %v, want %v", i, got[i].Style.Italic, want.italic)
				}
			}
		})
	}
}

func TestParseItalic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSpan []struct {
			text   string
			bold   bool
			italic bool
		}
	}{
		{
			name:  "simple italic",
			input: "*italic text*",
			wantSpan: []struct {
				text   string
				bold   bool
				italic bool
			}{
				{text: "italic text", bold: false, italic: true},
			},
		},
		{
			name:  "italic in middle of text",
			input: "some *italic* text",
			wantSpan: []struct {
				text   string
				bold   bool
				italic bool
			}{
				{text: "some ", bold: false, italic: false},
				{text: "italic", bold: false, italic: true},
				{text: " text", bold: false, italic: false},
			},
		},
		{
			name:  "multiple italic sections",
			input: "*one* and *two*",
			wantSpan: []struct {
				text   string
				bold   bool
				italic bool
			}{
				{text: "one", bold: false, italic: true},
				{text: " and ", bold: false, italic: false},
				{text: "two", bold: false, italic: true},
			},
		},
		{
			name:  "unclosed italic treated as plain",
			input: "*unclosed",
			wantSpan: []struct {
				text   string
				bold   bool
				italic bool
			}{
				{text: "*unclosed", bold: false, italic: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if len(got) != len(tt.wantSpan) {
				t.Fatalf("got %d spans, want %d spans\n  got: %+v", len(got), len(tt.wantSpan), got)
			}
			for i, want := range tt.wantSpan {
				if got[i].Text != want.text {
					t.Errorf("span[%d].Text = %q, want %q", i, got[i].Text, want.text)
				}
				if got[i].Style.Bold != want.bold {
					t.Errorf("span[%d].Bold = %v, want %v", i, got[i].Style.Bold, want.bold)
				}
				if got[i].Style.Italic != want.italic {
					t.Errorf("span[%d].Italic = %v, want %v", i, got[i].Style.Italic, want.italic)
				}
			}
		})
	}
}

func TestParseBoldItalic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSpan []struct {
			text   string
			bold   bool
			italic bool
		}
	}{
		{
			name:  "bold and italic combined",
			input: "***bold and italic***",
			wantSpan: []struct {
				text   string
				bold   bool
				italic bool
			}{
				{text: "bold and italic", bold: true, italic: true},
			},
		},
		{
			name:  "bold italic in middle",
			input: "some ***bold italic*** text",
			wantSpan: []struct {
				text   string
				bold   bool
				italic bool
			}{
				{text: "some ", bold: false, italic: false},
				{text: "bold italic", bold: true, italic: true},
				{text: " text", bold: false, italic: false},
			},
		},
		{
			name:  "bold and italic separately",
			input: "**bold** and *italic*",
			wantSpan: []struct {
				text   string
				bold   bool
				italic bool
			}{
				{text: "bold", bold: true, italic: false},
				{text: " and ", bold: false, italic: false},
				{text: "italic", bold: false, italic: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if len(got) != len(tt.wantSpan) {
				t.Fatalf("got %d spans, want %d spans\n  got: %+v", len(got), len(tt.wantSpan), got)
			}
			for i, want := range tt.wantSpan {
				if got[i].Text != want.text {
					t.Errorf("span[%d].Text = %q, want %q", i, got[i].Text, want.text)
				}
				if got[i].Style.Bold != want.bold {
					t.Errorf("span[%d].Bold = %v, want %v", i, got[i].Style.Bold, want.bold)
				}
				if got[i].Style.Italic != want.italic {
					t.Errorf("span[%d].Italic = %v, want %v", i, got[i].Style.Italic, want.italic)
				}
			}
		})
	}
}

func TestParseInlineCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSpan []struct {
			text   string
			isCode bool
		}
	}{
		{
			name:  "simple inline code",
			input: "`code`",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "code", isCode: true},
			},
		},
		{
			name:  "code in middle of text",
			input: "use the `fmt.Println` function",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "use the ", isCode: false},
				{text: "fmt.Println", isCode: true},
				{text: " function", isCode: false},
			},
		},
		{
			name:  "multiple code spans",
			input: "`one` and `two`",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "one", isCode: true},
				{text: " and ", isCode: false},
				{text: "two", isCode: true},
			},
		},
		{
			name:  "unclosed backtick treated as plain",
			input: "`unclosed",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "`unclosed", isCode: false},
			},
		},
		{
			name:  "empty code span",
			input: "``",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "", isCode: true},
			},
		},
		{
			name:  "code with special characters",
			input: "`x := y + z`",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "x := y + z", isCode: true},
			},
		},
		{
			name:  "code span preserves asterisks",
			input: "`**not bold**`",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "**not bold**", isCode: true},
			},
		},
		{
			name:  "code and bold mixed",
			input: "**bold** and `code`",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "bold", isCode: false},   // bold, not code
				{text: " and ", isCode: false},  // plain
				{text: "code", isCode: true},    // code
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if len(got) != len(tt.wantSpan) {
				t.Fatalf("got %d spans, want %d spans\n  got: %+v", len(got), len(tt.wantSpan), got)
			}
			for i, want := range tt.wantSpan {
				if got[i].Text != want.text {
					t.Errorf("span[%d].Text = %q, want %q", i, got[i].Text, want.text)
				}
				if got[i].Style.Code != want.isCode {
					t.Errorf("span[%d].Code = %v, want %v (style: %+v)", i, got[i].Style.Code, want.isCode, got[i].Style)
				}
			}
		})
	}
}
