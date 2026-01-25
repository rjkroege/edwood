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

// WithBackground is an Option that sets the background image for the frame.
func WithBackground(b draw.Image) Option {
	return func(f *frameImpl) {
		f.background = b
	}
}

// WithFont is an Option that sets the font for the frame.
func WithFont(f draw.Font) Option {
	return func(fi *frameImpl) {
		fi.font = f
	}
}

func TestFrameWithFont(t *testing.T) {
	rect := image.Rect(0, 0, 400, 300)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	f := NewFrame()
	fi := f.(*frameImpl)

	// Initialize with display and font
	f.Init(rect, WithDisplay(display), WithFont(font))

	// Verify font is stored
	if fi.font == nil {
		t.Error("font not stored in frame")
	}
	if fi.font != font {
		t.Errorf("font = %v, want %v", fi.font, font)
	}

	// Verify font properties are accessible
	if fi.font.Height() != 14 {
		t.Errorf("font.Height() = %d, want 14", fi.font.Height())
	}
}

func TestFrameRedrawFillsBackground(t *testing.T) {
	rect := image.Rect(10, 20, 200, 300)
	display := edwoodtest.NewDisplay(rect)

	// Allocate a distinct background color image (use Medblue as a visually distinct color)
	bgColor := draw.Medblue
	bgImage, err := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, bgColor)
	if err != nil {
		t.Fatalf("AllocImage failed: %v", err)
	}

	f := NewFrame()
	f.Init(rect, WithDisplay(display), WithBackground(bgImage))

	// Clear any draw ops from init
	display.(edwoodtest.GettableDrawOps).Clear()

	// Call Redraw
	f.Redraw()

	// Verify that a fill operation occurred for the frame's rectangle
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	if len(ops) == 0 {
		t.Fatal("Redraw() did not produce any draw operations")
	}

	// Look for a fill operation covering the frame rectangle
	found := false
	expectedFill := "fill " + rect.String()
	for _, op := range ops {
		if len(op) >= len(expectedFill) && op[:len(expectedFill)] == expectedFill {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Redraw() did not fill the background rectangle %v\ngot ops: %v", rect, ops)
	}
}
