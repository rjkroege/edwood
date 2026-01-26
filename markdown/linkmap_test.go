package markdown

import (
	"testing"
)

func TestLinkMapEmpty(t *testing.T) {
	lm := NewLinkMap()

	// Looking up any position in an empty LinkMap should return empty string
	tests := []int{0, 1, 10, 100, -1}
	for _, pos := range tests {
		url := lm.URLAt(pos)
		if url != "" {
			t.Errorf("URLAt(%d) on empty LinkMap = %q, want empty string", pos, url)
		}
	}
}

func TestLinkMapLookup(t *testing.T) {
	lm := NewLinkMap()

	// Add a link from position 5 to 10 with URL "https://example.com"
	lm.Add(5, 10, "https://example.com")

	tests := []struct {
		name    string
		pos     int
		wantURL string
	}{
		{
			name:    "before link",
			pos:     4,
			wantURL: "",
		},
		{
			name:    "at link start",
			pos:     5,
			wantURL: "https://example.com",
		},
		{
			name:    "inside link",
			pos:     7,
			wantURL: "https://example.com",
		},
		{
			name:    "at link end (exclusive)",
			pos:     10,
			wantURL: "",
		},
		{
			name:    "after link",
			pos:     15,
			wantURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lm.URLAt(tt.pos)
			if got != tt.wantURL {
				t.Errorf("URLAt(%d) = %q, want %q", tt.pos, got, tt.wantURL)
			}
		})
	}
}

func TestLinkMapMultipleLinks(t *testing.T) {
	lm := NewLinkMap()

	// Add multiple links
	// "Click [here](url1) or [there](url2) for info"
	// Positions: "Click " = 0-6, "here" = 6-10, " or " = 10-14, "there" = 14-19, " for info" = 19-28
	lm.Add(6, 10, "https://url1.com")
	lm.Add(14, 19, "https://url2.com")

	tests := []struct {
		name    string
		pos     int
		wantURL string
	}{
		{
			name:    "before first link",
			pos:     3,
			wantURL: "",
		},
		{
			name:    "in first link",
			pos:     7,
			wantURL: "https://url1.com",
		},
		{
			name:    "between links",
			pos:     12,
			wantURL: "",
		},
		{
			name:    "in second link",
			pos:     16,
			wantURL: "https://url2.com",
		},
		{
			name:    "after second link",
			pos:     25,
			wantURL: "",
		},
		{
			name:    "at first link start",
			pos:     6,
			wantURL: "https://url1.com",
		},
		{
			name:    "at second link start",
			pos:     14,
			wantURL: "https://url2.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lm.URLAt(tt.pos)
			if got != tt.wantURL {
				t.Errorf("URLAt(%d) = %q, want %q", tt.pos, got, tt.wantURL)
			}
		})
	}
}

func TestLinkMapAdjacentLinks(t *testing.T) {
	lm := NewLinkMap()

	// Two links right next to each other
	lm.Add(0, 5, "https://first.com")
	lm.Add(5, 10, "https://second.com")

	tests := []struct {
		name    string
		pos     int
		wantURL string
	}{
		{
			name:    "in first link",
			pos:     2,
			wantURL: "https://first.com",
		},
		{
			name:    "at boundary (second link)",
			pos:     5,
			wantURL: "https://second.com",
		},
		{
			name:    "in second link",
			pos:     7,
			wantURL: "https://second.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lm.URLAt(tt.pos)
			if got != tt.wantURL {
				t.Errorf("URLAt(%d) = %q, want %q", tt.pos, got, tt.wantURL)
			}
		})
	}
}
