package rich

import (
	"image"
	"testing"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/edwoodtest"
)

func TestNewFrame(t *testing.T) {
	f := NewFrame()
	if f == nil {
		t.Fatal("NewFrame() returned nil")
	}
}

func TestFrameInit(t *testing.T) {
	// Create a mock display
	rect := image.Rect(10, 20, 200, 300)
	display := edwoodtest.NewDisplay(rect)

	f := NewFrame()
	fi := f.(*frameImpl)

	// Initialize with rect and display
	f.Init(rect, WithDisplay(display))

	// Verify rect is stored
	if got := f.Rect(); got != rect {
		t.Errorf("Rect() = %v, want %v", got, rect)
	}

	// Verify display is stored
	if fi.display != display {
		t.Errorf("display not stored correctly")
	}
}

func TestFrameInitWithOptions(t *testing.T) {
	rect := image.Rect(0, 0, 100, 100)
	display := edwoodtest.NewDisplay(rect)

	f := NewFrame()
	fi := f.(*frameImpl)

	// Test that multiple options can be applied
	f.Init(rect, WithDisplay(display))

	if fi.display == nil {
		t.Error("WithDisplay option not applied")
	}
	if f.Rect() != rect {
		t.Errorf("Rect() = %v, want %v", f.Rect(), rect)
	}
}

// WithDisplay is an Option that sets the display for the frame.
func WithDisplay(d draw.Display) Option {
	return func(f *frameImpl) {
		f.display = d
	}
}
