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
