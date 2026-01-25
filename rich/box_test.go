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
