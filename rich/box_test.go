package rich

import "testing"

func TestBoxIsNewline(t *testing.T) {
	tests := []struct {
		name string
		box  Box
		want bool
	}{
		{
			name: "newline box",
			box:  Box{Nrune: -1, Bc: '\n'},
			want: true,
		},
		{
			name: "tab box is not newline",
			box:  Box{Nrune: -1, Bc: '\t'},
			want: false,
		},
		{
			name: "text box is not newline",
			box:  Box{Text: []byte("hello"), Nrune: 5, Bc: 0},
			want: false,
		},
		{
			name: "empty text box is not newline",
			box:  Box{Text: []byte{}, Nrune: 0, Bc: 0},
			want: false,
		},
		{
			name: "positive Nrune with newline Bc is not newline",
			box:  Box{Nrune: 1, Bc: '\n'},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.box.IsNewline()
			if got != tt.want {
				t.Errorf("Box{Nrune: %d, Bc: %q}.IsNewline() = %v, want %v",
					tt.box.Nrune, tt.box.Bc, got, tt.want)
			}
		})
	}
}

func TestBoxIsTab(t *testing.T) {
	tests := []struct {
		name string
		box  Box
		want bool
	}{
		{
			name: "tab box",
			box:  Box{Nrune: -1, Bc: '\t'},
			want: true,
		},
		{
			name: "newline box is not tab",
			box:  Box{Nrune: -1, Bc: '\n'},
			want: false,
		},
		{
			name: "text box is not tab",
			box:  Box{Text: []byte("hello"), Nrune: 5, Bc: 0},
			want: false,
		},
		{
			name: "empty text box is not tab",
			box:  Box{Text: []byte{}, Nrune: 0, Bc: 0},
			want: false,
		},
		{
			name: "positive Nrune with tab Bc is not tab",
			box:  Box{Nrune: 1, Bc: '\t'},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.box.IsTab()
			if got != tt.want {
				t.Errorf("Box{Nrune: %d, Bc: %q}.IsTab() = %v, want %v",
					tt.box.Nrune, tt.box.Bc, got, tt.want)
			}
		})
	}
}

// TestBoxIsImage tests the IsImage method for detecting image boxes.
// An image box has Style.Image=true and ImageData set.
func TestBoxIsImage(t *testing.T) {
	// Create a mock CachedImage for testing
	mockCachedImage := &CachedImage{
		Width:  100,
		Height: 50,
		Path:   "test.png",
	}

	tests := []struct {
		name string
		box  Box
		want bool
	}{
		{
			name: "image box with ImageData",
			box: Box{
				Style:     Style{Image: true, ImageURL: "test.png", ImageAlt: "test image", Scale: 1.0},
				ImageData: mockCachedImage,
			},
			want: true,
		},
		{
			name: "image style but no ImageData",
			box: Box{
				Style:     Style{Image: true, ImageURL: "test.png", ImageAlt: "test image", Scale: 1.0},
				ImageData: nil,
			},
			want: false,
		},
		{
			name: "ImageData but no Image style",
			box: Box{
				Style:     Style{Image: false, Scale: 1.0},
				ImageData: mockCachedImage,
			},
			want: false,
		},
		{
			name: "text box is not image",
			box:  Box{Text: []byte("hello"), Nrune: 5, Bc: 0, Style: DefaultStyle()},
			want: false,
		},
		{
			name: "newline box is not image",
			box:  Box{Nrune: -1, Bc: '\n', Style: DefaultStyle()},
			want: false,
		},
		{
			name: "tab box is not image",
			box:  Box{Nrune: -1, Bc: '\t', Style: DefaultStyle()},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.box.IsImage()
			if got != tt.want {
				t.Errorf("Box.IsImage() = %v, want %v", got, tt.want)
			}
		})
	}
}
