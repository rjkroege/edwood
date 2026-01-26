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
			// In markdown, consecutive lines within a paragraph are joined with spaces
			name:     "multiline text",
			input:    "Line one\nLine two\nLine three",
			wantLen:  1,
			wantText: "Line one Line two Line three",
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

func TestParseLink(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSpan []struct {
			text   string
			isLink bool
		}
	}{
		{
			name:  "simple link",
			input: "[click here](https://example.com)",
			wantSpan: []struct {
				text   string
				isLink bool
			}{
				{text: "click here", isLink: true},
			},
		},
		{
			name:  "link in middle of text",
			input: "See [this link](https://example.com) for details.",
			wantSpan: []struct {
				text   string
				isLink bool
			}{
				{text: "See ", isLink: false},
				{text: "this link", isLink: true},
				{text: " for details.", isLink: false},
			},
		},
		{
			name:  "link at end of text",
			input: "Visit [our site](https://example.com)",
			wantSpan: []struct {
				text   string
				isLink bool
			}{
				{text: "Visit ", isLink: false},
				{text: "our site", isLink: true},
			},
		},
		{
			name:  "link at start of text",
			input: "[Home](/) is here",
			wantSpan: []struct {
				text   string
				isLink bool
			}{
				{text: "Home", isLink: true},
				{text: " is here", isLink: false},
			},
		},
		{
			name:  "unclosed bracket treated as plain",
			input: "[unclosed link",
			wantSpan: []struct {
				text   string
				isLink bool
			}{
				{text: "[unclosed link", isLink: false},
			},
		},
		{
			name:  "bracket without url parens treated as plain",
			input: "[no url]",
			wantSpan: []struct {
				text   string
				isLink bool
			}{
				{text: "[no url]", isLink: false},
			},
		},
		{
			name:  "bracket with unclosed parens treated as plain",
			input: "[text](unclosed",
			wantSpan: []struct {
				text   string
				isLink bool
			}{
				{text: "[text](unclosed", isLink: false},
			},
		},
		{
			name:  "empty link text",
			input: "[](https://example.com)",
			wantSpan: []struct {
				text   string
				isLink bool
			}{
				{text: "", isLink: true},
			},
		},
		{
			name:  "link with special characters in url",
			input: "[docs](https://example.com/path?q=1&r=2#section)",
			wantSpan: []struct {
				text   string
				isLink bool
			}{
				{text: "docs", isLink: true},
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
				if got[i].Style.Link != want.isLink {
					t.Errorf("span[%d].Link = %v, want %v (style: %+v)", i, got[i].Style.Link, want.isLink, got[i].Style)
				}
			}
		})
	}
}

func TestParseLinkWithBold(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSpan []struct {
			text   string
			isLink bool
			isBold bool
		}
	}{
		{
			name:  "bold text in link",
			input: "[**bold link**](https://example.com)",
			wantSpan: []struct {
				text   string
				isLink bool
				isBold bool
			}{
				{text: "bold link", isLink: true, isBold: true},
			},
		},
		{
			name:  "italic text in link",
			input: "[*italic link*](https://example.com)",
			wantSpan: []struct {
				text   string
				isLink bool
				isBold bool
			}{
				{text: "italic link", isLink: true, isBold: false},
			},
		},
		{
			name:  "mixed bold and regular in link",
			input: "[click **here**](https://example.com)",
			wantSpan: []struct {
				text   string
				isLink bool
				isBold bool
			}{
				{text: "click ", isLink: true, isBold: false},
				{text: "here", isLink: true, isBold: true},
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
				if got[i].Style.Link != want.isLink {
					t.Errorf("span[%d].Link = %v, want %v", i, got[i].Style.Link, want.isLink)
				}
				if got[i].Style.Bold != want.isBold {
					t.Errorf("span[%d].Bold = %v, want %v", i, got[i].Style.Bold, want.isBold)
				}
			}
		})
	}
}

func TestParseMultipleLinks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSpan []struct {
			text   string
			isLink bool
		}
	}{
		{
			name:  "two links",
			input: "[one](https://one.com) and [two](https://two.com)",
			wantSpan: []struct {
				text   string
				isLink bool
			}{
				{text: "one", isLink: true},
				{text: " and ", isLink: false},
				{text: "two", isLink: true},
			},
		},
		{
			name:  "three links in sequence",
			input: "[a](1)[b](2)[c](3)",
			wantSpan: []struct {
				text   string
				isLink bool
			}{
				{text: "a", isLink: true},
				{text: "b", isLink: true},
				{text: "c", isLink: true},
			},
		},
		{
			name:  "links with other formatting",
			input: "**bold** [link](url) *italic*",
			wantSpan: []struct {
				text   string
				isLink bool
			}{
				{text: "bold", isLink: false},
				{text: " ", isLink: false},
				{text: "link", isLink: true},
				{text: " ", isLink: false},
				{text: "italic", isLink: false},
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
				if got[i].Style.Link != want.isLink {
					t.Errorf("span[%d].Link = %v, want %v (style: %+v)", i, got[i].Style.Link, want.isLink, got[i].Style)
				}
			}
		})
	}
}

func TestLinkHasBlueColor(t *testing.T) {
	// Links should have blue foreground color (LinkBlue)
	got := Parse("[click here](https://example.com)")

	if len(got) != 1 {
		t.Fatalf("got %d spans, want 1 span\n  got: %+v", len(got), got)
	}

	span := got[0]
	if !span.Style.Link {
		t.Fatal("span.Style.Link = false, want true")
	}

	if span.Style.Fg == nil {
		t.Fatal("span.Style.Fg is nil, want LinkBlue color")
	}

	// Check that it's blue (high blue component, low red/green)
	r, g, b, _ := span.Style.Fg.RGBA()
	// Convert from 16-bit to 8-bit for easier comparison
	r8, g8, b8 := r>>8, g>>8, b>>8

	// Blue should be dominant
	if b8 <= r8 || b8 <= g8 {
		t.Errorf("link Fg is not blue enough: R=%d, G=%d, B=%d", r8, g8, b8)
	}

	// Blue component should be substantial (at least 128/255)
	if b8 < 128 {
		t.Errorf("link Fg blue component too low: %d, want >= 128", b8)
	}
}

func TestInlineCodeBackground(t *testing.T) {
	// Inline code spans should have a background color set
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple code span",
			input: "`code`",
		},
		{
			name:  "code span with content",
			input: "`fmt.Println()`",
		},
		{
			name:  "code span with special chars",
			input: "`x := y + z`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)

			if len(got) != 1 {
				t.Fatalf("got %d spans, want 1 span\n  got: %+v", len(got), got)
			}

			span := got[0]
			if !span.Style.Code {
				t.Fatal("span.Style.Code = false, want true")
			}

			if span.Style.Bg == nil {
				t.Fatal("span.Style.Bg is nil, want inline code background color")
			}

			// Check that background is a light gray (high RGB values, roughly equal)
			r, g, b, _ := span.Style.Bg.RGBA()
			// Convert from 16-bit to 8-bit for easier comparison
			r8, g8, b8 := r>>8, g>>8, b>>8

			// Should be light (all components >= 200)
			if r8 < 200 || g8 < 200 || b8 < 200 {
				t.Errorf("inline code Bg is not light enough: R=%d, G=%d, B=%d (want all >= 200)", r8, g8, b8)
			}

			// Should be grayish (components roughly equal, within 20 of each other)
			if abs(int(r8)-int(g8)) > 20 || abs(int(g8)-int(b8)) > 20 || abs(int(r8)-int(b8)) > 20 {
				t.Errorf("inline code Bg is not gray: R=%d, G=%d, B=%d (want components within 20)", r8, g8, b8)
			}
		})
	}
}

func TestInlineCodeWithSurroundingText(t *testing.T) {
	// When inline code is surrounded by text, only the code span should have a background
	got := Parse("use the `fmt.Println` function")

	if len(got) != 3 {
		t.Fatalf("got %d spans, want 3 spans\n  got: %+v", len(got), got)
	}

	// First span: "use the " - should NOT have background
	if got[0].Text != "use the " {
		t.Errorf("span[0].Text = %q, want %q", got[0].Text, "use the ")
	}
	if got[0].Style.Code {
		t.Error("span[0].Style.Code = true, want false")
	}
	if got[0].Style.Bg != nil {
		t.Errorf("span[0].Style.Bg = %v, want nil (no background for plain text)", got[0].Style.Bg)
	}

	// Second span: "fmt.Println" - should have Code=true and Bg set
	if got[1].Text != "fmt.Println" {
		t.Errorf("span[1].Text = %q, want %q", got[1].Text, "fmt.Println")
	}
	if !got[1].Style.Code {
		t.Error("span[1].Style.Code = false, want true")
	}
	if got[1].Style.Bg == nil {
		t.Error("span[1].Style.Bg is nil, want inline code background color")
	}

	// Third span: " function" - should NOT have background
	if got[2].Text != " function" {
		t.Errorf("span[2].Text = %q, want %q", got[2].Text, " function")
	}
	if got[2].Style.Code {
		t.Error("span[2].Style.Code = true, want false")
	}
	if got[2].Style.Bg != nil {
		t.Errorf("span[2].Style.Bg = %v, want nil (no background for plain text)", got[2].Style.Bg)
	}
}

// abs returns the absolute value of x.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestParseHorizontalRuleHyphens(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
	}{
		{
			name:    "three hyphens",
			input:   "---",
			wantLen: 1,
		},
		{
			name:    "three hyphens with newline",
			input:   "---\n",
			wantLen: 1,
		},
		{
			name:    "four hyphens",
			input:   "----",
			wantLen: 1,
		},
		{
			name:    "many hyphens",
			input:   "----------",
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if len(got) != tt.wantLen {
				t.Fatalf("got %d spans, want %d spans\n  got: %+v", len(got), tt.wantLen, got)
			}

			// The span should contain HRuleRune
			if len(got) > 0 {
				span := got[0]
				if span.Text != string(rich.HRuleRune)+"\n" && span.Text != string(rich.HRuleRune) {
					t.Errorf("span.Text = %q, want HRuleRune marker", span.Text)
				}
			}
		})
	}
}

func TestParseHorizontalRuleAsterisks(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
	}{
		{
			name:    "three asterisks",
			input:   "***",
			wantLen: 1,
		},
		{
			name:    "three asterisks with newline",
			input:   "***\n",
			wantLen: 1,
		},
		{
			name:    "four asterisks",
			input:   "****",
			wantLen: 1,
		},
		{
			name:    "many asterisks",
			input:   "**********",
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if len(got) != tt.wantLen {
				t.Fatalf("got %d spans, want %d spans\n  got: %+v", len(got), tt.wantLen, got)
			}

			// The span should contain HRuleRune
			if len(got) > 0 {
				span := got[0]
				if span.Text != string(rich.HRuleRune)+"\n" && span.Text != string(rich.HRuleRune) {
					t.Errorf("span.Text = %q, want HRuleRune marker", span.Text)
				}
			}
		})
	}
}

func TestParseHorizontalRuleUnderscores(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
	}{
		{
			name:    "three underscores",
			input:   "___",
			wantLen: 1,
		},
		{
			name:    "three underscores with newline",
			input:   "___\n",
			wantLen: 1,
		},
		{
			name:    "four underscores",
			input:   "____",
			wantLen: 1,
		},
		{
			name:    "many underscores",
			input:   "__________",
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if len(got) != tt.wantLen {
				t.Fatalf("got %d spans, want %d spans\n  got: %+v", len(got), tt.wantLen, got)
			}

			// The span should contain HRuleRune
			if len(got) > 0 {
				span := got[0]
				if span.Text != string(rich.HRuleRune)+"\n" && span.Text != string(rich.HRuleRune) {
					t.Errorf("span.Text = %q, want HRuleRune marker", span.Text)
				}
			}
		})
	}
}

func TestParseHorizontalRuleWithSpaces(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
	}{
		{
			name:    "hyphens with spaces",
			input:   "- - -",
			wantLen: 1,
		},
		{
			name:    "hyphens with spaces and newline",
			input:   "- - -\n",
			wantLen: 1,
		},
		{
			name:    "asterisks with spaces",
			input:   "* * *",
			wantLen: 1,
		},
		{
			name:    "underscores with spaces",
			input:   "_ _ _",
			wantLen: 1,
		},
		{
			name:    "hyphens with multiple spaces",
			input:   "-  -  -",
			wantLen: 1,
		},
		{
			name:    "more than three with spaces",
			input:   "- - - -",
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if len(got) != tt.wantLen {
				t.Fatalf("got %d spans, want %d spans\n  got: %+v", len(got), tt.wantLen, got)
			}

			// The span should contain HRuleRune
			if len(got) > 0 {
				span := got[0]
				if span.Text != string(rich.HRuleRune)+"\n" && span.Text != string(rich.HRuleRune) {
					t.Errorf("span.Text = %q, want HRuleRune marker", span.Text)
				}
			}
		})
	}
}

func TestParseNotHorizontalRule(t *testing.T) {
	// These patterns should NOT be parsed as horizontal rules
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "only two hyphens",
			input: "--",
		},
		{
			name:  "only two asterisks",
			input: "**",
		},
		{
			name:  "only two underscores",
			input: "__",
		},
		{
			name:  "mixed characters",
			input: "-*-",
		},
		{
			name:  "hyphens with text",
			input: "---text",
		},
		{
			name:  "text then hyphens",
			input: "text---",
		},
		{
			name:  "hyphens in middle of line",
			input: "a --- b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if len(got) == 0 {
				t.Fatal("Parse returned empty content")
			}

			// None of the spans should contain HRuleRune
			for i, span := range got {
				for _, r := range span.Text {
					if r == rich.HRuleRune {
						t.Errorf("span[%d].Text contains HRuleRune, but should not for input %q", i, tt.input)
					}
				}
			}
		})
	}
}

func TestParseHorizontalRuleBetweenText(t *testing.T) {
	// Horizontal rule between text content
	input := "Above\n---\nBelow"
	got := Parse(input)

	// Should have 3 spans: text before, hrule, text after
	if len(got) < 3 {
		t.Fatalf("got %d spans, want at least 3 spans\n  got: %+v", len(got), got)
	}

	// Find the hrule span
	foundHRule := false
	for _, span := range got {
		for _, r := range span.Text {
			if r == rich.HRuleRune {
				foundHRule = true
				break
			}
		}
	}

	if !foundHRule {
		t.Error("did not find HRuleRune in parsed output")
	}
}

func TestParseFencedCodeBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSpan []struct {
			text   string
			isCode bool
		}
	}{
		{
			name:  "simple fenced code block",
			input: "```\ncode here\n```",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "code here\n", isCode: true},
			},
		},
		{
			name:  "fenced code block with multiple lines",
			input: "```\nline one\nline two\nline three\n```",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "line one\nline two\nline three\n", isCode: true},
			},
		},
		{
			name:  "fenced code block between text",
			input: "Before\n```\ncode\n```\nAfter",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "Before\n", isCode: false},
				{text: "code\n", isCode: true},
				{text: "After", isCode: false},
			},
		},
		{
			name:  "unclosed fenced code block",
			input: "```\nunclosed code",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				// If code block is unclosed, treat remaining content as code
				{text: "unclosed code", isCode: true},
			},
		},
		{
			name:  "empty fenced code block",
			input: "```\n```",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				// Empty code block should produce no output (fence lines are omitted)
			},
		},
		{
			name:  "fenced code block preserves asterisks",
			input: "```\n**not bold**\n```",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "**not bold**\n", isCode: true},
			},
		},
		{
			name:  "fenced code block preserves backticks",
			input: "```\nuse `code` here\n```",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "use `code` here\n", isCode: true},
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

func TestParseFencedCodeBlockWithLanguage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSpan []struct {
			text   string
			isCode bool
		}
	}{
		{
			name:  "go code block",
			input: "```go\nfunc main() {\n}\n```",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "func main() {\n}\n", isCode: true},
			},
		},
		{
			name:  "python code block",
			input: "```python\ndef hello():\n    print('hi')\n```",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "def hello():\n    print('hi')\n", isCode: true},
			},
		},
		{
			name:  "javascript code block",
			input: "```js\nconst x = 1;\n```",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "const x = 1;\n", isCode: true},
			},
		},
		{
			name:  "language with trailing space",
			input: "```go \ncode\n```",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				{text: "code\n", isCode: true},
			},
		},
		{
			name:  "language identifier is stripped",
			input: "```rust\nfn main() {}\n```",
			wantSpan: []struct {
				text   string
				isCode bool
			}{
				// The language "rust" should not appear in output
				{text: "fn main() {}\n", isCode: true},
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

func TestParseFencedCodeBlockPreservesWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantText string
	}{
		{
			name:     "preserves leading spaces",
			input:    "```\n    indented\n```",
			wantText: "    indented\n",
		},
		{
			name:     "preserves tabs",
			input:    "```\n\tindented with tab\n```",
			wantText: "\tindented with tab\n",
		},
		{
			name:     "preserves multiple indent levels",
			input:    "```\nif x {\n    if y {\n        deep\n    }\n}\n```",
			wantText: "if x {\n    if y {\n        deep\n    }\n}\n",
		},
		{
			name:     "preserves blank lines",
			input:    "```\nline one\n\nline three\n```",
			wantText: "line one\n\nline three\n",
		},
		{
			name:     "preserves trailing spaces",
			input:    "```\nwith trailing   \n```",
			wantText: "with trailing   \n",
		},
		{
			name:     "preserves mixed whitespace",
			input:    "```\n  \t  mixed\n```",
			wantText: "  \t  mixed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if len(got) == 0 {
				t.Fatalf("Parse returned empty content for input %q", tt.input)
			}

			// The code block content should be in a code-styled span
			var codeSpan *rich.Span
			for i := range got {
				if got[i].Style.Code {
					codeSpan = &got[i]
					break
				}
			}

			if codeSpan == nil {
				t.Fatalf("no code span found in output: %+v", got)
			}

			if codeSpan.Text != tt.wantText {
				t.Errorf("code span text = %q, want %q", codeSpan.Text, tt.wantText)
			}
		})
	}
}

func TestParseFencedCodeBlockHasBackground(t *testing.T) {
	// Fenced code blocks should have a background color
	got := Parse("```\ncode\n```")

	if len(got) != 1 {
		t.Fatalf("got %d spans, want 1 span\n  got: %+v", len(got), got)
	}

	span := got[0]
	if !span.Style.Code {
		t.Fatal("span.Style.Code = false, want true")
	}

	if !span.Style.Block {
		t.Fatal("span.Style.Block = false, want true for fenced code blocks")
	}

	if span.Style.Bg == nil {
		t.Fatal("span.Style.Bg is nil, want code block background color")
	}

	// Check that background is a light gray (high RGB values, roughly equal)
	r, g, b, _ := span.Style.Bg.RGBA()
	// Convert from 16-bit to 8-bit for easier comparison
	r8, g8, b8 := r>>8, g>>8, b>>8

	// Should be light (all components >= 200)
	if r8 < 200 || g8 < 200 || b8 < 200 {
		t.Errorf("code block Bg is not light enough: R=%d, G=%d, B=%d (want all >= 200)", r8, g8, b8)
	}

	// Should be grayish (components roughly equal, within 20 of each other)
	if abs(int(r8)-int(g8)) > 20 || abs(int(g8)-int(b8)) > 20 || abs(int(r8)-int(b8)) > 20 {
		t.Errorf("code block Bg is not gray: R=%d, G=%d, B=%d (want components within 20)", r8, g8, b8)
	}
}

// Tests for Phase 15A: Lists

func TestIsUnorderedListItem(t *testing.T) {
	tests := []struct {
		name             string
		line             string
		wantIsListItem   bool
		wantIndentLevel  int
		wantContentStart int
	}{
		{
			name:             "hyphen marker",
			line:             "- Item",
			wantIsListItem:   true,
			wantIndentLevel:  0,
			wantContentStart: 2,
		},
		{
			name:             "asterisk marker",
			line:             "* Item",
			wantIsListItem:   true,
			wantIndentLevel:  0,
			wantContentStart: 2,
		},
		{
			name:             "plus marker",
			line:             "+ Item",
			wantIsListItem:   true,
			wantIndentLevel:  0,
			wantContentStart: 2,
		},
		{
			name:             "hyphen with trailing newline",
			line:             "- Item\n",
			wantIsListItem:   true,
			wantIndentLevel:  0,
			wantContentStart: 2,
		},
		{
			name:             "just marker and space",
			line:             "- ",
			wantIsListItem:   true,
			wantIndentLevel:  0,
			wantContentStart: 2,
		},
		{
			name:             "no space after marker",
			line:             "-Item",
			wantIsListItem:   false,
			wantIndentLevel:  0,
			wantContentStart: 0,
		},
		{
			name:             "just marker no space",
			line:             "-",
			wantIsListItem:   false,
			wantIndentLevel:  0,
			wantContentStart: 0,
		},
		{
			name:             "empty line",
			line:             "",
			wantIsListItem:   false,
			wantIndentLevel:  0,
			wantContentStart: 0,
		},
		{
			name:             "plain text",
			line:             "Hello world",
			wantIsListItem:   false,
			wantIndentLevel:  0,
			wantContentStart: 0,
		},
		{
			name:             "hyphen in middle of text",
			line:             "some - text",
			wantIsListItem:   false,
			wantIndentLevel:  0,
			wantContentStart: 0,
		},
		{
			name:             "double hyphen (not list)",
			line:             "-- Not a list",
			wantIsListItem:   false,
			wantIndentLevel:  0,
			wantContentStart: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isListItem, indentLevel, contentStart := isUnorderedListItem(tt.line)
			if isListItem != tt.wantIsListItem {
				t.Errorf("isUnorderedListItem(%q) isListItem = %v, want %v", tt.line, isListItem, tt.wantIsListItem)
			}
			if indentLevel != tt.wantIndentLevel {
				t.Errorf("isUnorderedListItem(%q) indentLevel = %d, want %d", tt.line, indentLevel, tt.wantIndentLevel)
			}
			if contentStart != tt.wantContentStart {
				t.Errorf("isUnorderedListItem(%q) contentStart = %d, want %d", tt.line, contentStart, tt.wantContentStart)
			}
		})
	}
}

func TestIsUnorderedListItemNested(t *testing.T) {
	tests := []struct {
		name             string
		line             string
		wantIsListItem   bool
		wantIndentLevel  int
		wantContentStart int
	}{
		{
			name:             "one level indent with 2 spaces",
			line:             "  - Nested item",
			wantIsListItem:   true,
			wantIndentLevel:  1,
			wantContentStart: 4,
		},
		{
			name:             "two levels indent with 4 spaces",
			line:             "    - Deep nested",
			wantIsListItem:   true,
			wantIndentLevel:  2,
			wantContentStart: 6,
		},
		{
			name:             "three levels indent with 6 spaces",
			line:             "      - Very deep",
			wantIsListItem:   true,
			wantIndentLevel:  3,
			wantContentStart: 8,
		},
		{
			name:             "one level indent with tab",
			line:             "\t- Tab nested",
			wantIsListItem:   true,
			wantIndentLevel:  1,
			wantContentStart: 3,
		},
		{
			name:             "two levels indent with tabs",
			line:             "\t\t- Double tab",
			wantIsListItem:   true,
			wantIndentLevel:  2,
			wantContentStart: 4,
		},
		{
			name:             "mixed indent (tab + 2 spaces)",
			line:             "\t  - Mixed indent",
			wantIsListItem:   true,
			wantIndentLevel:  2,
			wantContentStart: 5,
		},
		{
			name:             "nested asterisk",
			line:             "  * Nested asterisk",
			wantIsListItem:   true,
			wantIndentLevel:  1,
			wantContentStart: 4,
		},
		{
			name:             "nested plus",
			line:             "  + Nested plus",
			wantIsListItem:   true,
			wantIndentLevel:  1,
			wantContentStart: 4,
		},
		{
			name:             "odd number of spaces (1 space)",
			line:             " - One space indent",
			wantIsListItem:   true,
			wantIndentLevel:  0, // 1 space alone doesn't make a full indent level
			wantContentStart: 3,
		},
		{
			name:             "odd number of spaces (3 spaces)",
			line:             "   - Three space indent",
			wantIsListItem:   true,
			wantIndentLevel:  1, // 3 spaces = 1 full indent level
			wantContentStart: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isListItem, indentLevel, contentStart := isUnorderedListItem(tt.line)
			if isListItem != tt.wantIsListItem {
				t.Errorf("isUnorderedListItem(%q) isListItem = %v, want %v", tt.line, isListItem, tt.wantIsListItem)
			}
			if indentLevel != tt.wantIndentLevel {
				t.Errorf("isUnorderedListItem(%q) indentLevel = %d, want %d", tt.line, indentLevel, tt.wantIndentLevel)
			}
			if contentStart != tt.wantContentStart {
				t.Errorf("isUnorderedListItem(%q) contentStart = %d, want %d", tt.line, contentStart, tt.wantContentStart)
			}
		})
	}
}

func TestIsOrderedListItem(t *testing.T) {
	tests := []struct {
		name             string
		line             string
		wantIsListItem   bool
		wantIndentLevel  int
		wantContentStart int
		wantItemNumber   int
	}{
		{
			name:             "simple number with period",
			line:             "1. Item",
			wantIsListItem:   true,
			wantIndentLevel:  0,
			wantContentStart: 3,
			wantItemNumber:   1,
		},
		{
			name:             "number 2 with period",
			line:             "2. Second item",
			wantIsListItem:   true,
			wantIndentLevel:  0,
			wantContentStart: 3,
			wantItemNumber:   2,
		},
		{
			name:             "number 10 with period",
			line:             "10. Tenth item",
			wantIsListItem:   true,
			wantIndentLevel:  0,
			wantContentStart: 4,
			wantItemNumber:   10,
		},
		{
			name:             "large number with period",
			line:             "999. Large number",
			wantIsListItem:   true,
			wantIndentLevel:  0,
			wantContentStart: 5,
			wantItemNumber:   999,
		},
		{
			name:             "number with paren",
			line:             "1) Item with paren",
			wantIsListItem:   true,
			wantIndentLevel:  0,
			wantContentStart: 3,
			wantItemNumber:   1,
		},
		{
			name:             "number 5 with paren",
			line:             "5) Fifth item",
			wantIsListItem:   true,
			wantIndentLevel:  0,
			wantContentStart: 3,
			wantItemNumber:   5,
		},
		{
			name:             "with trailing newline",
			line:             "1. Item\n",
			wantIsListItem:   true,
			wantIndentLevel:  0,
			wantContentStart: 3,
			wantItemNumber:   1,
		},
		{
			name:             "just number period space",
			line:             "1. ",
			wantIsListItem:   true,
			wantIndentLevel:  0,
			wantContentStart: 3,
			wantItemNumber:   1,
		},
		{
			name:             "no space after period",
			line:             "1.Item",
			wantIsListItem:   false,
			wantIndentLevel:  0,
			wantContentStart: 0,
			wantItemNumber:   0,
		},
		{
			name:             "no space after paren",
			line:             "1)Item",
			wantIsListItem:   false,
			wantIndentLevel:  0,
			wantContentStart: 0,
			wantItemNumber:   0,
		},
		{
			name:             "just number no delimiter",
			line:             "1",
			wantIsListItem:   false,
			wantIndentLevel:  0,
			wantContentStart: 0,
			wantItemNumber:   0,
		},
		{
			name:             "empty line",
			line:             "",
			wantIsListItem:   false,
			wantIndentLevel:  0,
			wantContentStart: 0,
			wantItemNumber:   0,
		},
		{
			name:             "plain text",
			line:             "Hello world",
			wantIsListItem:   false,
			wantIndentLevel:  0,
			wantContentStart: 0,
			wantItemNumber:   0,
		},
		{
			name:             "number in middle of text",
			line:             "some 1. text",
			wantIsListItem:   false,
			wantIndentLevel:  0,
			wantContentStart: 0,
			wantItemNumber:   0,
		},
		{
			name:             "zero as number",
			line:             "0. Zero item",
			wantIsListItem:   true,
			wantIndentLevel:  0,
			wantContentStart: 3,
			wantItemNumber:   0,
		},
		{
			name:             "leading zero in number",
			line:             "01. Padded number",
			wantIsListItem:   true,
			wantIndentLevel:  0,
			wantContentStart: 4,
			wantItemNumber:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isListItem, indentLevel, contentStart, itemNumber := isOrderedListItem(tt.line)
			if isListItem != tt.wantIsListItem {
				t.Errorf("isOrderedListItem(%q) isListItem = %v, want %v", tt.line, isListItem, tt.wantIsListItem)
			}
			if indentLevel != tt.wantIndentLevel {
				t.Errorf("isOrderedListItem(%q) indentLevel = %d, want %d", tt.line, indentLevel, tt.wantIndentLevel)
			}
			if contentStart != tt.wantContentStart {
				t.Errorf("isOrderedListItem(%q) contentStart = %d, want %d", tt.line, contentStart, tt.wantContentStart)
			}
			if itemNumber != tt.wantItemNumber {
				t.Errorf("isOrderedListItem(%q) itemNumber = %d, want %d", tt.line, itemNumber, tt.wantItemNumber)
			}
		})
	}
}

func TestIsOrderedListItemNested(t *testing.T) {
	tests := []struct {
		name             string
		line             string
		wantIsListItem   bool
		wantIndentLevel  int
		wantContentStart int
		wantItemNumber   int
	}{
		{
			name:             "one level indent with 2 spaces",
			line:             "  1. Nested item",
			wantIsListItem:   true,
			wantIndentLevel:  1,
			wantContentStart: 5,
			wantItemNumber:   1,
		},
		{
			name:             "two levels indent with 4 spaces",
			line:             "    1. Deep nested",
			wantIsListItem:   true,
			wantIndentLevel:  2,
			wantContentStart: 7,
			wantItemNumber:   1,
		},
		{
			name:             "three levels indent with 6 spaces",
			line:             "      1. Very deep",
			wantIsListItem:   true,
			wantIndentLevel:  3,
			wantContentStart: 9,
			wantItemNumber:   1,
		},
		{
			name:             "one level indent with tab",
			line:             "\t1. Tab nested",
			wantIsListItem:   true,
			wantIndentLevel:  1,
			wantContentStart: 4,
			wantItemNumber:   1,
		},
		{
			name:             "two levels indent with tabs",
			line:             "\t\t1. Double tab",
			wantIsListItem:   true,
			wantIndentLevel:  2,
			wantContentStart: 5,
			wantItemNumber:   1,
		},
		{
			name:             "mixed indent (tab + 2 spaces)",
			line:             "\t  1. Mixed indent",
			wantIsListItem:   true,
			wantIndentLevel:  2,
			wantContentStart: 6,
			wantItemNumber:   1,
		},
		{
			name:             "nested with paren delimiter",
			line:             "  1) Nested paren",
			wantIsListItem:   true,
			wantIndentLevel:  1,
			wantContentStart: 5,
			wantItemNumber:   1,
		},
		{
			name:             "nested with multi-digit number",
			line:             "  10. Multi-digit nested",
			wantIsListItem:   true,
			wantIndentLevel:  1,
			wantContentStart: 6,
			wantItemNumber:   10,
		},
		{
			name:             "odd number of spaces (1 space)",
			line:             " 1. One space indent",
			wantIsListItem:   true,
			wantIndentLevel:  0, // 1 space alone doesn't make a full indent level
			wantContentStart: 4,
			wantItemNumber:   1,
		},
		{
			name:             "odd number of spaces (3 spaces)",
			line:             "   1. Three space indent",
			wantIsListItem:   true,
			wantIndentLevel:  1, // 3 spaces = 1 full indent level
			wantContentStart: 6,
			wantItemNumber:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isListItem, indentLevel, contentStart, itemNumber := isOrderedListItem(tt.line)
			if isListItem != tt.wantIsListItem {
				t.Errorf("isOrderedListItem(%q) isListItem = %v, want %v", tt.line, isListItem, tt.wantIsListItem)
			}
			if indentLevel != tt.wantIndentLevel {
				t.Errorf("isOrderedListItem(%q) indentLevel = %d, want %d", tt.line, indentLevel, tt.wantIndentLevel)
			}
			if contentStart != tt.wantContentStart {
				t.Errorf("isOrderedListItem(%q) contentStart = %d, want %d", tt.line, contentStart, tt.wantContentStart)
			}
			if itemNumber != tt.wantItemNumber {
				t.Errorf("isOrderedListItem(%q) itemNumber = %d, want %d", tt.line, itemNumber, tt.wantItemNumber)
			}
		})
	}
}

func TestParseUnorderedList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSpan []struct {
			text       string
			listBullet bool
			listItem   bool
			listIndent int
		}
	}{
		{
			name:  "simple unordered list item",
			input: "- Item one",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Item one", listBullet: false, listItem: true, listIndent: 0},
			},
		},
		{
			name:  "unordered list with asterisk marker",
			input: "* Item with asterisk",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Item with asterisk", listBullet: false, listItem: true, listIndent: 0},
			},
		},
		{
			name:  "unordered list with plus marker",
			input: "+ Item with plus",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Item with plus", listBullet: false, listItem: true, listIndent: 0},
			},
		},
		{
			name:  "multiple unordered list items",
			input: "- First\n- Second\n- Third",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "First\n", listBullet: false, listItem: true, listIndent: 0},
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Second\n", listBullet: false, listItem: true, listIndent: 0},
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Third", listBullet: false, listItem: true, listIndent: 0},
			},
		},
		{
			name:  "unordered list with bold text",
			input: "- **Bold** item",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Bold", listBullet: false, listItem: true, listIndent: 0},   // bold
				{text: " item", listBullet: false, listItem: true, listIndent: 0}, // plain
			},
		},
		{
			name:  "unordered list with code span",
			input: "- Use `code` here",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Use ", listBullet: false, listItem: true, listIndent: 0},
				{text: "code", listBullet: false, listItem: true, listIndent: 0}, // code span
				{text: " here", listBullet: false, listItem: true, listIndent: 0},
			},
		},
		{
			name:  "empty unordered list item",
			input: "- ",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
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
				if got[i].Style.ListBullet != want.listBullet {
					t.Errorf("span[%d].Style.ListBullet = %v, want %v", i, got[i].Style.ListBullet, want.listBullet)
				}
				if got[i].Style.ListItem != want.listItem {
					t.Errorf("span[%d].Style.ListItem = %v, want %v", i, got[i].Style.ListItem, want.listItem)
				}
				if got[i].Style.ListIndent != want.listIndent {
					t.Errorf("span[%d].Style.ListIndent = %d, want %d", i, got[i].Style.ListIndent, want.listIndent)
				}
			}
		})
	}
}

func TestParseNestedList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSpan []struct {
			text       string
			listBullet bool
			listItem   bool
			listIndent int
		}
	}{
		{
			name:  "simple nested unordered list",
			input: "- Parent\n  - Child",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Parent\n", listBullet: false, listItem: true, listIndent: 0},
				{text: "•", listBullet: true, listItem: false, listIndent: 1},
				{text: " ", listBullet: false, listItem: true, listIndent: 1},
				{text: "Child", listBullet: false, listItem: true, listIndent: 1},
			},
		},
		{
			name:  "nested list with multiple children",
			input: "- Parent\n  - Child 1\n  - Child 2",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Parent\n", listBullet: false, listItem: true, listIndent: 0},
				{text: "•", listBullet: true, listItem: false, listIndent: 1},
				{text: " ", listBullet: false, listItem: true, listIndent: 1},
				{text: "Child 1\n", listBullet: false, listItem: true, listIndent: 1},
				{text: "•", listBullet: true, listItem: false, listIndent: 1},
				{text: " ", listBullet: false, listItem: true, listIndent: 1},
				{text: "Child 2", listBullet: false, listItem: true, listIndent: 1},
			},
		},
		{
			name:  "nested list back to parent level",
			input: "- Parent 1\n  - Child\n- Parent 2",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Parent 1\n", listBullet: false, listItem: true, listIndent: 0},
				{text: "•", listBullet: true, listItem: false, listIndent: 1},
				{text: " ", listBullet: false, listItem: true, listIndent: 1},
				{text: "Child\n", listBullet: false, listItem: true, listIndent: 1},
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Parent 2", listBullet: false, listItem: true, listIndent: 0},
			},
		},
		{
			name:  "nested ordered list",
			input: "1. Parent\n   1. Child",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "1.", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Parent\n", listBullet: false, listItem: true, listIndent: 0},
				{text: "1.", listBullet: true, listItem: false, listIndent: 1},
				{text: " ", listBullet: false, listItem: true, listIndent: 1},
				{text: "Child", listBullet: false, listItem: true, listIndent: 1},
			},
		},
		{
			name:  "mixed nested lists",
			input: "- Unordered parent\n  1. Ordered child",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Unordered parent\n", listBullet: false, listItem: true, listIndent: 0},
				{text: "1.", listBullet: true, listItem: false, listIndent: 1},
				{text: " ", listBullet: false, listItem: true, listIndent: 1},
				{text: "Ordered child", listBullet: false, listItem: true, listIndent: 1},
			},
		},
		{
			name:  "nested list with tab indent",
			input: "- Parent\n\t- Child",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Parent\n", listBullet: false, listItem: true, listIndent: 0},
				{text: "•", listBullet: true, listItem: false, listIndent: 1},
				{text: " ", listBullet: false, listItem: true, listIndent: 1},
				{text: "Child", listBullet: false, listItem: true, listIndent: 1},
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
				if got[i].Style.ListBullet != want.listBullet {
					t.Errorf("span[%d].Style.ListBullet = %v, want %v", i, got[i].Style.ListBullet, want.listBullet)
				}
				if got[i].Style.ListItem != want.listItem {
					t.Errorf("span[%d].Style.ListItem = %v, want %v", i, got[i].Style.ListItem, want.listItem)
				}
				if got[i].Style.ListIndent != want.listIndent {
					t.Errorf("span[%d].Style.ListIndent = %d, want %d", i, got[i].Style.ListIndent, want.listIndent)
				}
			}
		})
	}
}

func TestParseDeepNestedList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSpan []struct {
			text       string
			listBullet bool
			listItem   bool
			listIndent int
		}
	}{
		{
			name:  "three level nested unordered list",
			input: "- Level 0\n  - Level 1\n    - Level 2",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Level 0\n", listBullet: false, listItem: true, listIndent: 0},
				{text: "•", listBullet: true, listItem: false, listIndent: 1},
				{text: " ", listBullet: false, listItem: true, listIndent: 1},
				{text: "Level 1\n", listBullet: false, listItem: true, listIndent: 1},
				{text: "•", listBullet: true, listItem: false, listIndent: 2},
				{text: " ", listBullet: false, listItem: true, listIndent: 2},
				{text: "Level 2", listBullet: false, listItem: true, listIndent: 2},
			},
		},
		{
			name:  "four level nested list",
			input: "- L0\n  - L1\n    - L2\n      - L3",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "L0\n", listBullet: false, listItem: true, listIndent: 0},
				{text: "•", listBullet: true, listItem: false, listIndent: 1},
				{text: " ", listBullet: false, listItem: true, listIndent: 1},
				{text: "L1\n", listBullet: false, listItem: true, listIndent: 1},
				{text: "•", listBullet: true, listItem: false, listIndent: 2},
				{text: " ", listBullet: false, listItem: true, listIndent: 2},
				{text: "L2\n", listBullet: false, listItem: true, listIndent: 2},
				{text: "•", listBullet: true, listItem: false, listIndent: 3},
				{text: " ", listBullet: false, listItem: true, listIndent: 3},
				{text: "L3", listBullet: false, listItem: true, listIndent: 3},
			},
		},
		{
			name:  "deep nested then return to shallow",
			input: "- L0\n  - L1\n    - L2\n  - L1 again\n- L0 again",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "L0\n", listBullet: false, listItem: true, listIndent: 0},
				{text: "•", listBullet: true, listItem: false, listIndent: 1},
				{text: " ", listBullet: false, listItem: true, listIndent: 1},
				{text: "L1\n", listBullet: false, listItem: true, listIndent: 1},
				{text: "•", listBullet: true, listItem: false, listIndent: 2},
				{text: " ", listBullet: false, listItem: true, listIndent: 2},
				{text: "L2\n", listBullet: false, listItem: true, listIndent: 2},
				{text: "•", listBullet: true, listItem: false, listIndent: 1},
				{text: " ", listBullet: false, listItem: true, listIndent: 1},
				{text: "L1 again\n", listBullet: false, listItem: true, listIndent: 1},
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "L0 again", listBullet: false, listItem: true, listIndent: 0},
			},
		},
		{
			name:  "three level nested ordered list",
			input: "1. Level 0\n  1. Level 1\n    1. Level 2", // Use 2-space indentation (consistent with unordered lists)
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "1.", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Level 0\n", listBullet: false, listItem: true, listIndent: 0},
				{text: "1.", listBullet: true, listItem: false, listIndent: 1},
				{text: " ", listBullet: false, listItem: true, listIndent: 1},
				{text: "Level 1\n", listBullet: false, listItem: true, listIndent: 1},
				{text: "1.", listBullet: true, listItem: false, listIndent: 2},
				{text: " ", listBullet: false, listItem: true, listIndent: 2},
				{text: "Level 2", listBullet: false, listItem: true, listIndent: 2},
			},
		},
		{
			name:  "mixed deep nested lists",
			input: "- Unordered L0\n  1. Ordered L1\n    - Unordered L2",
			wantSpan: []struct {
				text       string
				listBullet bool
				listItem   bool
				listIndent int
			}{
				{text: "•", listBullet: true, listItem: false, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listIndent: 0},
				{text: "Unordered L0\n", listBullet: false, listItem: true, listIndent: 0},
				{text: "1.", listBullet: true, listItem: false, listIndent: 1},
				{text: " ", listBullet: false, listItem: true, listIndent: 1},
				{text: "Ordered L1\n", listBullet: false, listItem: true, listIndent: 1},
				{text: "•", listBullet: true, listItem: false, listIndent: 2},
				{text: " ", listBullet: false, listItem: true, listIndent: 2},
				{text: "Unordered L2", listBullet: false, listItem: true, listIndent: 2},
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
				if got[i].Style.ListBullet != want.listBullet {
					t.Errorf("span[%d].Style.ListBullet = %v, want %v", i, got[i].Style.ListBullet, want.listBullet)
				}
				if got[i].Style.ListItem != want.listItem {
					t.Errorf("span[%d].Style.ListItem = %v, want %v", i, got[i].Style.ListItem, want.listItem)
				}
				if got[i].Style.ListIndent != want.listIndent {
					t.Errorf("span[%d].Style.ListIndent = %d, want %d", i, got[i].Style.ListIndent, want.listIndent)
				}
			}
		})
	}
}

func TestParseOrderedList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSpan []struct {
			text        string
			listBullet  bool
			listItem    bool
			listOrdered bool
			listNumber  int
			listIndent  int
		}
	}{
		{
			name:  "simple ordered list item",
			input: "1. First item",
			wantSpan: []struct {
				text        string
				listBullet  bool
				listItem    bool
				listOrdered bool
				listNumber  int
				listIndent  int
			}{
				{text: "1.", listBullet: true, listItem: false, listOrdered: true, listNumber: 1, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listOrdered: true, listNumber: 1, listIndent: 0},
				{text: "First item", listBullet: false, listItem: true, listOrdered: true, listNumber: 1, listIndent: 0},
			},
		},
		{
			name:  "ordered list item with paren",
			input: "1) First item",
			wantSpan: []struct {
				text        string
				listBullet  bool
				listItem    bool
				listOrdered bool
				listNumber  int
				listIndent  int
			}{
				{text: "1.", listBullet: true, listItem: false, listOrdered: true, listNumber: 1, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listOrdered: true, listNumber: 1, listIndent: 0},
				{text: "First item", listBullet: false, listItem: true, listOrdered: true, listNumber: 1, listIndent: 0},
			},
		},
		{
			name:  "multiple ordered list items",
			input: "1. First\n2. Second\n3. Third",
			wantSpan: []struct {
				text        string
				listBullet  bool
				listItem    bool
				listOrdered bool
				listNumber  int
				listIndent  int
			}{
				{text: "1.", listBullet: true, listItem: false, listOrdered: true, listNumber: 1, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listOrdered: true, listNumber: 1, listIndent: 0},
				{text: "First\n", listBullet: false, listItem: true, listOrdered: true, listNumber: 1, listIndent: 0},
				{text: "2.", listBullet: true, listItem: false, listOrdered: true, listNumber: 2, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listOrdered: true, listNumber: 2, listIndent: 0},
				{text: "Second\n", listBullet: false, listItem: true, listOrdered: true, listNumber: 2, listIndent: 0},
				{text: "3.", listBullet: true, listItem: false, listOrdered: true, listNumber: 3, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listOrdered: true, listNumber: 3, listIndent: 0},
				{text: "Third", listBullet: false, listItem: true, listOrdered: true, listNumber: 3, listIndent: 0},
			},
		},
		{
			name:  "ordered list with multi-digit number",
			input: "10. Tenth item",
			wantSpan: []struct {
				text        string
				listBullet  bool
				listItem    bool
				listOrdered bool
				listNumber  int
				listIndent  int
			}{
				{text: "10.", listBullet: true, listItem: false, listOrdered: true, listNumber: 10, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listOrdered: true, listNumber: 10, listIndent: 0},
				{text: "Tenth item", listBullet: false, listItem: true, listOrdered: true, listNumber: 10, listIndent: 0},
			},
		},
		{
			name:  "ordered list with bold text",
			input: "1. **Bold** item",
			wantSpan: []struct {
				text        string
				listBullet  bool
				listItem    bool
				listOrdered bool
				listNumber  int
				listIndent  int
			}{
				{text: "1.", listBullet: true, listItem: false, listOrdered: true, listNumber: 1, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listOrdered: true, listNumber: 1, listIndent: 0},
				{text: "Bold", listBullet: false, listItem: true, listOrdered: true, listNumber: 1, listIndent: 0},   // bold
				{text: " item", listBullet: false, listItem: true, listOrdered: true, listNumber: 1, listIndent: 0}, // plain
			},
		},
		{
			name:  "empty ordered list item",
			input: "1. ",
			wantSpan: []struct {
				text        string
				listBullet  bool
				listItem    bool
				listOrdered bool
				listNumber  int
				listIndent  int
			}{
				{text: "1.", listBullet: true, listItem: false, listOrdered: true, listNumber: 1, listIndent: 0},
				{text: " ", listBullet: false, listItem: true, listOrdered: true, listNumber: 1, listIndent: 0},
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
				if got[i].Style.ListBullet != want.listBullet {
					t.Errorf("span[%d].Style.ListBullet = %v, want %v", i, got[i].Style.ListBullet, want.listBullet)
				}
				if got[i].Style.ListItem != want.listItem {
					t.Errorf("span[%d].Style.ListItem = %v, want %v", i, got[i].Style.ListItem, want.listItem)
				}
				if got[i].Style.ListOrdered != want.listOrdered {
					t.Errorf("span[%d].Style.ListOrdered = %v, want %v", i, got[i].Style.ListOrdered, want.listOrdered)
				}
				if got[i].Style.ListNumber != want.listNumber {
					t.Errorf("span[%d].Style.ListNumber = %d, want %d", i, got[i].Style.ListNumber, want.listNumber)
				}
				if got[i].Style.ListIndent != want.listIndent {
					t.Errorf("span[%d].Style.ListIndent = %d, want %d", i, got[i].Style.ListIndent, want.listIndent)
				}
			}
		})
	}
}

// =============================================================================
// Table Tests (Phase 15B)
// =============================================================================

// Alignment is imported from rich package for use in tests
type Alignment = rich.Alignment

const (
	AlignLeft   = rich.AlignLeft
	AlignCenter = rich.AlignCenter
	AlignRight  = rich.AlignRight
)

func TestIsTableRow(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantIs    bool
		wantCells int // expected number of cells if it is a table row
	}{
		{
			name:      "simple table row",
			input:     "| A | B |",
			wantIs:    true,
			wantCells: 2,
		},
		{
			name:      "table row with more cells",
			input:     "| A | B | C | D |",
			wantIs:    true,
			wantCells: 4,
		},
		{
			name:      "table row without leading pipe",
			input:     "A | B |",
			wantIs:    false,
			wantCells: 0,
		},
		{
			name:      "table row without trailing pipe",
			input:     "| A | B",
			wantIs:    true, // Common markdown parsers accept this
			wantCells: 2,
		},
		{
			name:      "plain text with pipe",
			input:     "This is not | a table",
			wantIs:    false,
			wantCells: 0,
		},
		{
			name:      "empty line",
			input:     "",
			wantIs:    false,
			wantCells: 0,
		},
		{
			name:      "only pipes",
			input:     "|||",
			wantIs:    true,
			wantCells: 2, // Two empty cells
		},
		{
			name:      "table row with trailing newline",
			input:     "| A | B |\n",
			wantIs:    true,
			wantCells: 2,
		},
		{
			name:      "table row with spaces in cells",
			input:     "| Header 1 | Header 2 |",
			wantIs:    true,
			wantCells: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIs, gotCells := isTableRow(tt.input)
			if gotIs != tt.wantIs {
				t.Errorf("isTableRow(%q) = %v, want %v", tt.input, gotIs, tt.wantIs)
			}
			if gotIs && len(gotCells) != tt.wantCells {
				t.Errorf("isTableRow(%q) cells = %d, want %d", tt.input, len(gotCells), tt.wantCells)
			}
		})
	}
}

func TestIsTableRowMultipleCells(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCells []string
	}{
		{
			name:      "two cells",
			input:     "| A | B |",
			wantCells: []string{"A", "B"},
		},
		{
			name:      "three cells with content",
			input:     "| Name | Age | City |",
			wantCells: []string{"Name", "Age", "City"},
		},
		{
			name:      "cells with extra whitespace",
			input:     "|  A  |  B  |",
			wantCells: []string{"A", "B"}, // Whitespace should be trimmed
		},
		{
			name:      "empty cells",
			input:     "| | |",
			wantCells: []string{"", ""},
		},
		{
			name:      "single cell",
			input:     "| A |",
			wantCells: []string{"A"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isRow, cells := isTableRow(tt.input)
			if !isRow {
				t.Fatalf("isTableRow(%q) = false, want true", tt.input)
			}
			if len(cells) != len(tt.wantCells) {
				t.Errorf("cell count = %d, want %d\n  got: %v", len(cells), len(tt.wantCells), cells)
				return
			}
			for i, want := range tt.wantCells {
				if cells[i] != want {
					t.Errorf("cell[%d] = %q, want %q", i, cells[i], want)
				}
			}
		})
	}
}

func TestIsTableSeparator(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantIs bool
	}{
		{
			name:   "simple separator",
			input:  "|---|---|",
			wantIs: true,
		},
		{
			name:   "separator with spaces",
			input:  "| --- | --- |",
			wantIs: true,
		},
		{
			name:   "separator with more dashes",
			input:  "|-----|-----|",
			wantIs: true,
		},
		{
			name:   "not enough dashes",
			input:  "|--|--|",
			wantIs: false, // Need at least 3 dashes
		},
		{
			name:   "header row not separator",
			input:  "| A | B |",
			wantIs: false,
		},
		{
			name:   "mixed content",
			input:  "|---| A |",
			wantIs: false, // All cells must be separator cells
		},
		{
			name:   "empty line",
			input:  "",
			wantIs: false,
		},
		{
			name:   "only pipes",
			input:  "|||",
			wantIs: false,
		},
		{
			name:   "single separator cell",
			input:  "|---|",
			wantIs: true,
		},
		{
			name:   "many separator cells",
			input:  "|---|---|---|---|",
			wantIs: true,
		},
		{
			name:   "separator with trailing newline",
			input:  "|---|---|\n",
			wantIs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTableSeparatorRow(tt.input)
			if got != tt.wantIs {
				t.Errorf("isTableSeparatorRow(%q) = %v, want %v", tt.input, got, tt.wantIs)
			}
		})
	}
}

func TestIsTableSeparatorWithAlignment(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantIs     bool
		wantAligns []Alignment
	}{
		{
			name:       "left aligned",
			input:      "|:---|:---|",
			wantIs:     true,
			wantAligns: []Alignment{AlignLeft, AlignLeft},
		},
		{
			name:       "right aligned",
			input:      "|---:|---:|",
			wantIs:     true,
			wantAligns: []Alignment{AlignRight, AlignRight},
		},
		{
			name:       "center aligned",
			input:      "|:---:|:---:|",
			wantIs:     true,
			wantAligns: []Alignment{AlignCenter, AlignCenter},
		},
		{
			name:       "mixed alignment",
			input:      "|:---|:---:|---:|",
			wantIs:     true,
			wantAligns: []Alignment{AlignLeft, AlignCenter, AlignRight},
		},
		{
			name:       "default alignment (no colons)",
			input:      "|---|---|",
			wantIs:     true,
			wantAligns: []Alignment{AlignLeft, AlignLeft}, // Default is left
		},
		{
			name:       "with spaces",
			input:      "| :--- | :---: | ---: |",
			wantIs:     true,
			wantAligns: []Alignment{AlignLeft, AlignCenter, AlignRight},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIs, gotAligns := parseTableSeparator(tt.input)
			if gotIs != tt.wantIs {
				t.Errorf("parseTableSeparator(%q) = %v, want %v", tt.input, gotIs, tt.wantIs)
				return
			}
			if !gotIs {
				return
			}
			if len(gotAligns) != len(tt.wantAligns) {
				t.Errorf("alignment count = %d, want %d\n  got: %v", len(gotAligns), len(tt.wantAligns), gotAligns)
				return
			}
			for i, want := range tt.wantAligns {
				if gotAligns[i] != want {
					t.Errorf("align[%d] = %d, want %d", i, gotAligns[i], want)
				}
			}
		})
	}
}

func TestParseSimpleTable(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSpan []struct {
			text        string
			table       bool
			tableHeader bool
			code        bool
			block       bool
		}
	}{
		{
			name: "simple 2x2 table",
			input: `| A | B |
|---|---|
| 1 | 2 |`,
			wantSpan: []struct {
				text        string
				table       bool
				tableHeader bool
				code        bool
				block       bool
			}{
				// Header row
				{text: "| A | B |\n", table: true, tableHeader: true, code: true, block: true},
				// Separator row
				{text: "|---|---|\n", table: true, tableHeader: false, code: true, block: true},
				// Data row
				{text: "| 1 | 2 |", table: true, tableHeader: false, code: true, block: true},
			},
		},
		{
			name: "table with multiple data rows",
			input: `| Name | Value |
|------|-------|
| foo  | 1     |
| bar  | 2     |`,
			wantSpan: []struct {
				text        string
				table       bool
				tableHeader bool
				code        bool
				block       bool
			}{
				{text: "| Name | Value |\n", table: true, tableHeader: true, code: true, block: true},
				{text: "|------|-------|\n", table: true, tableHeader: false, code: true, block: true},
				{text: "| foo  | 1     |\n", table: true, tableHeader: false, code: true, block: true},
				{text: "| bar  | 2     |", table: true, tableHeader: false, code: true, block: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if len(got) != len(tt.wantSpan) {
				t.Fatalf("got %d spans, want %d spans\n  input: %q\n  got: %+v", len(got), len(tt.wantSpan), tt.input, got)
			}
			for i, want := range tt.wantSpan {
				if got[i].Text != want.text {
					t.Errorf("span[%d].Text = %q, want %q", i, got[i].Text, want.text)
				}
				if got[i].Style.Table != want.table {
					t.Errorf("span[%d].Style.Table = %v, want %v", i, got[i].Style.Table, want.table)
				}
				if got[i].Style.TableHeader != want.tableHeader {
					t.Errorf("span[%d].Style.TableHeader = %v, want %v", i, got[i].Style.TableHeader, want.tableHeader)
				}
				if got[i].Style.Code != want.code {
					t.Errorf("span[%d].Style.Code = %v, want %v", i, got[i].Style.Code, want.code)
				}
				if got[i].Style.Block != want.block {
					t.Errorf("span[%d].Style.Block = %v, want %v", i, got[i].Style.Block, want.block)
				}
			}
		})
	}
}

func TestParseTableWithAlignment(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantAligns []Alignment // Alignment for each column
	}{
		{
			name: "left aligned columns",
			input: `| A | B |
|:--|:--|
| 1 | 2 |`,
			wantAligns: []Alignment{AlignLeft, AlignLeft},
		},
		{
			name: "center aligned columns",
			input: `| A | B |
|:--:|:--:|
| 1 | 2 |`,
			wantAligns: []Alignment{AlignCenter, AlignCenter},
		},
		{
			name: "right aligned columns",
			input: `| A | B |
|--:|--:|
| 1 | 2 |`,
			wantAligns: []Alignment{AlignRight, AlignRight},
		},
		{
			name: "mixed alignment",
			input: `| Left | Center | Right |
|:-----|:------:|------:|
| L    |   C    |     R |`,
			wantAligns: []Alignment{AlignLeft, AlignCenter, AlignRight},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse and check that alignments are captured correctly
			// The alignment should be stored in the table spans
			got := Parse(tt.input)

			// Find a data cell span to check alignment
			foundDataCell := false
			for _, span := range got {
				if span.Style.Table && !span.Style.TableHeader {
					foundDataCell = true
					// For now, we just verify the table is parsed
					// The full alignment check would require checking per-cell alignment
					break
				}
			}

			if !foundDataCell {
				t.Error("no data cell found in parsed table")
			}
		})
	}
}

func TestCalculateColumnWidths(t *testing.T) {
	tests := []struct {
		name       string
		rows       [][]string
		wantWidths []int
	}{
		{
			name: "uniform widths",
			rows: [][]string{
				{"A", "B"},
				{"1", "2"},
			},
			wantWidths: []int{1, 1},
		},
		{
			name: "varying widths",
			rows: [][]string{
				{"Name", "Value"},
				{"foo", "1"},
				{"barbaz", "12345"},
			},
			wantWidths: []int{6, 5}, // max of each column
		},
		{
			name: "empty cells",
			rows: [][]string{
				{"A", "B", "C"},
				{"", "xx", ""},
			},
			wantWidths: []int{1, 2, 1},
		},
		{
			name: "single row",
			rows: [][]string{
				{"Header1", "Header2", "Header3"},
			},
			wantWidths: []int{7, 7, 7},
		},
		{
			name:       "empty table",
			rows:       [][]string{},
			wantWidths: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateColumnWidths(tt.rows)
			if len(got) != len(tt.wantWidths) {
				t.Errorf("width count = %d, want %d\n  got: %v", len(got), len(tt.wantWidths), got)
				return
			}
			for i, want := range tt.wantWidths {
				if got[i] != want {
					t.Errorf("width[%d] = %d, want %d", i, got[i], want)
				}
			}
		})
	}
}

func TestEmitAlignedTable(t *testing.T) {
	// Test that table cells are padded to column widths
	input := `| A | BB |
|---|---|
| 1 | 2  |`

	got := Parse(input)

	// The table should be rendered with aligned columns
	// We just check that it parses without error and produces table spans
	foundTable := false
	for _, span := range got {
		if span.Style.Table {
			foundTable = true
			break
		}
	}

	if !foundTable {
		t.Error("no table spans found in parsed output")
	}
}

func TestEmitTableWithWrap(t *testing.T) {
	// Test table with longer cell content
	input := `| Column A | Column B |
|----------|----------|
| Short    | This is a longer cell |`

	got := Parse(input)

	// The table should be rendered (for now, we don't wrap cells)
	// Just verify it parses as a table
	foundTable := false
	for _, span := range got {
		if span.Style.Table {
			foundTable = true
			break
		}
	}

	if !foundTable {
		t.Error("no table spans found in parsed output")
	}
}

func TestTableSourceMap(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "simple table source mapping",
			input: `| A | B |
|---|---|
| 1 | 2 |`,
		},
		{
			name: "table in document",
			input: `# Header

| A | B |
|---|---|
| 1 | 2 |

Some text after.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, sourceMap, _ := ParseWithSourceMap(tt.input)

			// Verify content was parsed
			if len(content) == 0 {
				t.Error("no content parsed")
				return
			}

			// Verify source map exists and can map positions
			// ToSource should return valid positions for rendered content
			totalLen := 0
			for _, span := range content {
				totalLen += len([]rune(span.Text))
			}

			if totalLen > 0 {
				// Map from start of rendered to source
				srcStart, srcEnd := sourceMap.ToSource(0, 1)
				if srcStart < 0 || srcEnd < 0 {
					t.Errorf("invalid source mapping: srcStart=%d, srcEnd=%d", srcStart, srcEnd)
				}
			}
		})
	}
}

func TestTableInDocument(t *testing.T) {
	// Test table surrounded by other content
	input := `# Title

Some paragraph text here.

| Header 1 | Header 2 |
|----------|----------|
| Data 1   | Data 2   |

More text after the table.`

	got := Parse(input)

	// Should have heading, paragraph, table, and trailing paragraph
	foundHeading := false
	foundTable := false
	foundParagraph := false

	for _, span := range got {
		if span.Style.Bold && span.Style.Scale > 1.0 {
			foundHeading = true
		}
		if span.Style.Table {
			foundTable = true
		}
		if !span.Style.Bold && !span.Style.Table && span.Style.Scale == 1.0 && span.Text != "\n" {
			foundParagraph = true
		}
	}

	if !foundHeading {
		t.Error("no heading found")
	}
	if !foundTable {
		t.Error("no table found")
	}
	if !foundParagraph {
		t.Error("no paragraph found")
	}
}

func TestTableNotTable(t *testing.T) {
	// Test that certain patterns are NOT parsed as tables
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "pipe in regular text",
			input: "This is | not a table",
		},
		{
			name:  "pipe at start but no separator",
			input: "| This looks like a header\nBut has no separator row",
		},
		{
			name:  "code block with pipe",
			input: "```\n| A | B |\n|---|---|\n```",
		},
		{
			name:  "single pipe row",
			input: "| Just one row |",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			for _, span := range got {
				if span.Style.Table {
					t.Errorf("unexpected table span found in %q: %+v", tt.input, span)
				}
			}
		})
	}
}
