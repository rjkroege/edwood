package edwoodtest

import (
	"image"
	"reflect"
	"testing"

	"github.com/rjkroege/edwood/draw"
)

// TestMockImageImplementsInterface verifies that mockImage implements draw.Image.
func TestMockImageImplementsInterface(t *testing.T) {
	var _ draw.Image = (*mockImage)(nil)
}

// TestMockImageHasLoad verifies that mockImage has a Load method.
// This test will fail until Load is added to both the draw.Image interface
// and the mockImage implementation.
func TestMockImageHasLoad(t *testing.T) {
	mockType := reflect.TypeOf((*mockImage)(nil))
	loadMethod, ok := mockType.MethodByName("Load")
	if !ok {
		t.Fatal("mockImage does not have Load method - needs to be added to implement draw.Image with Load")
	}

	// Verify method signature matches draw.Image.Load
	// Load(r image.Rectangle, data []byte) (int, error)
	methodType := loadMethod.Type

	// Method has receiver as first input, then image.Rectangle, []byte
	if methodType.NumIn() != 3 {
		t.Errorf("Load should have 3 inputs (receiver + 2 params), got %d", methodType.NumIn())
	}

	if methodType.NumOut() != 2 {
		t.Errorf("Load should have 2 outputs, got %d", methodType.NumOut())
	}
}

// TestMockImageLoadBehavior tests the behavior of mockImage.Load.
func TestMockImageLoadBehavior(t *testing.T) {
	display := NewDisplay(image.Rect(0, 0, 100, 100))
	img := NewImage(display, "test", image.Rect(0, 0, 10, 10))

	// Cast to get access to Load method
	mockImg, ok := img.(*mockImage)
	if !ok {
		t.Fatal("NewImage did not return *mockImage")
	}

	// Test Load method
	r := image.Rect(0, 0, 2, 2)
	data := make([]byte, 16) // 2x2 RGBA = 16 bytes
	n, err := mockImg.Load(r, data)
	if err != nil {
		t.Errorf("Load returned error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Load returned %d, want %d", n, len(data))
	}
}
