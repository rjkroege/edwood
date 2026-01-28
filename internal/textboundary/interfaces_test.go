package textboundary

import (
	"image"
	"testing"
)

// TestMouseSnapshotButtons tests the Buttons method.
func TestMouseSnapshotButtons(t *testing.T) {
	tests := []struct {
		name    string
		state   int
		want    int
	}{
		{"no buttons", 0, 0},
		{"left button", 1, 1},
		{"right button", 4, 4},
		{"multiple buttons", 5, 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snap := MouseSnapshot{ButtonState: tc.state}
			if got := snap.Buttons(); got != tc.want {
				t.Errorf("Buttons() = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestMouseSnapshotPoint tests the Point method.
func TestMouseSnapshotPoint(t *testing.T) {
	tests := []struct {
		name string
		pos  image.Point
	}{
		{"origin", image.Point{0, 0}},
		{"positive", image.Point{100, 200}},
		{"mixed", image.Point{-50, 150}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snap := MouseSnapshot{Position: tc.pos}
			if got := snap.Point(); got != tc.pos {
				t.Errorf("Point() = %v, want %v", got, tc.pos)
			}
		})
	}
}

// TestMouseSnapshotMsec tests the Msec method.
func TestMouseSnapshotMsec(t *testing.T) {
	tests := []struct {
		name      string
		timestamp uint32
	}{
		{"zero", 0},
		{"typical", 1234567890},
		{"max", ^uint32(0)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snap := MouseSnapshot{Timestamp: tc.timestamp}
			if got := snap.Msec(); got != tc.timestamp {
				t.Errorf("Msec() = %d, want %d", got, tc.timestamp)
			}
		})
	}
}

// TestMouseSnapshotHasMoved tests the HasMoved method.
func TestMouseSnapshotHasMoved(t *testing.T) {
	tests := []struct {
		name      string
		snapPos   image.Point
		otherPos  image.Point
		threshold int
		want      bool
	}{
		{
			name:      "no movement",
			snapPos:   image.Point{100, 100},
			otherPos:  image.Point{100, 100},
			threshold: 3,
			want:      false,
		},
		{
			name:      "x movement below threshold",
			snapPos:   image.Point{100, 100},
			otherPos:  image.Point{102, 100},
			threshold: 3,
			want:      false,
		},
		{
			name:      "x movement at threshold",
			snapPos:   image.Point{100, 100},
			otherPos:  image.Point{103, 100},
			threshold: 3,
			want:      true,
		},
		{
			name:      "y movement below threshold",
			snapPos:   image.Point{100, 100},
			otherPos:  image.Point{100, 102},
			threshold: 3,
			want:      false,
		},
		{
			name:      "y movement at threshold",
			snapPos:   image.Point{100, 100},
			otherPos:  image.Point{100, 103},
			threshold: 3,
			want:      true,
		},
		{
			name:      "negative x movement at threshold",
			snapPos:   image.Point{100, 100},
			otherPos:  image.Point{97, 100},
			threshold: 3,
			want:      true,
		},
		{
			name:      "negative y movement at threshold",
			snapPos:   image.Point{100, 100},
			otherPos:  image.Point{100, 97},
			threshold: 3,
			want:      true,
		},
		{
			name:      "diagonal movement below threshold",
			snapPos:   image.Point{100, 100},
			otherPos:  image.Point{101, 101},
			threshold: 3,
			want:      false,
		},
		{
			name:      "zero threshold same position",
			snapPos:   image.Point{100, 100},
			otherPos:  image.Point{100, 100},
			threshold: 0,
			want:      true, // 0 >= 0 is true
		},
		{
			name:      "single pixel with zero threshold",
			snapPos:   image.Point{100, 100},
			otherPos:  image.Point{101, 100},
			threshold: 0,
			want:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snap := MouseSnapshot{Position: tc.snapPos}
			if got := snap.HasMoved(tc.otherPos, tc.threshold); got != tc.want {
				t.Errorf("HasMoved(%v, %d) = %v, want %v", tc.otherPos, tc.threshold, got, tc.want)
			}
		})
	}
}

// TestMouseSnapshotButtonsChanged tests the ButtonsChanged method.
func TestMouseSnapshotButtonsChanged(t *testing.T) {
	tests := []struct {
		name       string
		snapState  int
		checkState int
		want       bool
	}{
		{"same no buttons", 0, 0, false},
		{"same left button", 1, 1, false},
		{"released button", 1, 0, true},
		{"pressed button", 0, 1, true},
		{"different buttons", 1, 4, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snap := MouseSnapshot{ButtonState: tc.snapState}
			if got := snap.ButtonsChanged(tc.checkState); got != tc.want {
				t.Errorf("ButtonsChanged(%d) = %v, want %v", tc.checkState, got, tc.want)
			}
		})
	}
}

// TestMouseSnapshotMouseStateInterface tests that MouseSnapshot implements MouseState.
func TestMouseSnapshotMouseStateInterface(t *testing.T) {
	var _ MouseState = MouseSnapshot{}

	snap := MouseSnapshot{
		ButtonState: 1,
		Position:    image.Point{100, 200},
		Timestamp:   12345,
	}

	// Test via interface
	var state MouseState = snap
	if got := state.Buttons(); got != 1 {
		t.Errorf("Buttons() via interface = %d, want 1", got)
	}
	if got := state.Point(); got != (image.Point{100, 200}) {
		t.Errorf("Point() via interface = %v, want {100, 200}", got)
	}
	if got := state.Msec(); got != 12345 {
		t.Errorf("Msec() via interface = %d, want 12345", got)
	}
}

// TestNilDirectoryResolver tests the NilDirectoryResolver.
func TestNilDirectoryResolver(t *testing.T) {
	var resolver DirectoryResolver = NilDirectoryResolver{}
	if got := resolver.ResolveDir(); got != "" {
		t.Errorf("NilDirectoryResolver.ResolveDir() = %q, want empty", got)
	}
}

// TestStaticDirectoryResolver tests the StaticDirectoryResolver.
func TestStaticDirectoryResolver(t *testing.T) {
	tests := []struct {
		name string
		dir  string
	}{
		{"empty", ""},
		{"absolute path", "/home/user/project"},
		{"relative path", "src/pkg"},
		{"current dir", "."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolver := StaticDirectoryResolver{Dir: tc.dir}
			if got := resolver.ResolveDir(); got != tc.dir {
				t.Errorf("ResolveDir() = %q, want %q", got, tc.dir)
			}
		})
	}
}

// TestFuncDirectoryResolver tests the FuncDirectoryResolver adapter.
func TestFuncDirectoryResolver(t *testing.T) {
	callCount := 0
	resolver := FuncDirectoryResolver(func() string {
		callCount++
		return "/computed/path"
	})

	if got := resolver.ResolveDir(); got != "/computed/path" {
		t.Errorf("ResolveDir() = %q, want %q", got, "/computed/path")
	}
	if callCount != 1 {
		t.Errorf("function called %d times, want 1", callCount)
	}

	// Call again to verify function is called each time
	resolver.ResolveDir()
	if callCount != 2 {
		t.Errorf("function called %d times after second call, want 2", callCount)
	}
}

// TestDirectoryResolverInterface tests that all resolvers implement DirectoryResolver.
func TestDirectoryResolverInterface(t *testing.T) {
	var _ DirectoryResolver = NilDirectoryResolver{}
	var _ DirectoryResolver = StaticDirectoryResolver{}
	var _ DirectoryResolver = FuncDirectoryResolver(nil)

	// All implementations should be usable interchangeably
	resolvers := []DirectoryResolver{
		NilDirectoryResolver{},
		StaticDirectoryResolver{Dir: "/test"},
		FuncDirectoryResolver(func() string { return "/func" }),
	}

	expected := []string{"", "/test", "/func"}
	for i, r := range resolvers {
		if got := r.ResolveDir(); got != expected[i] {
			t.Errorf("resolver[%d].ResolveDir() = %q, want %q", i, got, expected[i])
		}
	}
}

// TestMouseWaiterFuncAdapter tests the MouseWaiterFunc adapter.
func TestMouseWaiterFuncAdapter(t *testing.T) {
	callCount := 0
	var lastButton int
	var lastPos image.Point
	var lastThreshold int

	waiter := MouseWaiterFunc(func(button int, pos image.Point, threshold int) bool {
		callCount++
		lastButton = button
		lastPos = pos
		lastThreshold = threshold
		return button == 1 // return true for left button
	})

	// Test with left button
	if got := waiter.WaitForChange(1, image.Point{10, 20}, 3); !got {
		t.Error("WaitForChange(1, ...) = false, want true")
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
	if lastButton != 1 || lastPos != (image.Point{10, 20}) || lastThreshold != 3 {
		t.Errorf("got button=%d pos=%v threshold=%d, want button=1 pos={10,20} threshold=3",
			lastButton, lastPos, lastThreshold)
	}

	// Test with no button
	if got := waiter.WaitForChange(0, image.Point{30, 40}, 5); got {
		t.Error("WaitForChange(0, ...) = true, want false")
	}
}
