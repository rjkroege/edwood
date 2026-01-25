package rich

import "testing"

func TestSpanLen(t *testing.T) {
	tests := []struct {
		name string
		span Span
		want int
	}{
		{
			name: "empty span",
			span: Span{Text: "", Style: DefaultStyle()},
			want: 0,
		},
		{
			name: "ascii text",
			span: Span{Text: "hello", Style: DefaultStyle()},
			want: 5,
		},
		{
			name: "unicode text",
			span: Span{Text: "hello\u4e16\u754c", Style: DefaultStyle()}, // "hello" + 2 CJK chars
			want: 7,
		},
		{
			name: "emoji",
			span: Span{Text: "\U0001F600", Style: DefaultStyle()}, // single emoji
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := len([]rune(tt.span.Text))
			if got != tt.want {
				t.Errorf("len([]rune(%q)) = %d, want %d", tt.span.Text, got, tt.want)
			}
		})
	}
}

func TestContentLen(t *testing.T) {
	tests := []struct {
		name    string
		content Content
		want    int
	}{
		{
			name:    "empty content",
			content: Content{},
			want:    0,
		},
		{
			name: "single span",
			content: Content{
				{Text: "hello", Style: DefaultStyle()},
			},
			want: 5,
		},
		{
			name: "multiple spans",
			content: Content{
				{Text: "hello ", Style: DefaultStyle()},
				{Text: "world", Style: StyleBold},
			},
			want: 11,
		},
		{
			name: "unicode across spans",
			content: Content{
				{Text: "hello", Style: DefaultStyle()},
				{Text: "\u4e16\u754c", Style: StyleBold}, // 2 CJK chars
			},
			want: 7,
		},
		{
			name: "empty spans mixed",
			content: Content{
				{Text: "", Style: DefaultStyle()},
				{Text: "abc", Style: StyleBold},
				{Text: "", Style: StyleItalic},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.content.Len()
			if got != tt.want {
				t.Errorf("Content.Len() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPlainContent(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{
			name: "empty text",
			text: "",
		},
		{
			name: "simple text",
			text: "hello world",
		},
		{
			name: "text with newline",
			text: "line1\nline2",
		},
		{
			name: "unicode text",
			text: "hello\u4e16\u754c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := Plain(tt.text)

			// Plain should return exactly one span
			if len(content) != 1 {
				t.Fatalf("Plain(%q) returned %d spans, want 1", tt.text, len(content))
			}

			// The span should contain the original text
			if content[0].Text != tt.text {
				t.Errorf("Plain(%q)[0].Text = %q, want %q", tt.text, content[0].Text, tt.text)
			}

			// The span should use default style
			if !stylesEqual(content[0].Style, DefaultStyle()) {
				t.Errorf("Plain(%q)[0].Style = %v, want DefaultStyle()", tt.text, content[0].Style)
			}

			// Content.Len should equal rune count
			wantLen := len([]rune(tt.text))
			if content.Len() != wantLen {
				t.Errorf("Plain(%q).Len() = %d, want %d", tt.text, content.Len(), wantLen)
			}
		})
	}
}
