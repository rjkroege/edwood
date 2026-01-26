package draw

import (
	"reflect"
	"testing"
)

// TestImageLoad tests the Load method of the Image interface.
// This method is needed for loading pixel data into Plan 9 images
// for image rendering in Markdeep mode.
//
// The Load method allows loading raw pixel data into a Plan 9 image,
// which is required for rendering inline images in Markdeep preview mode.
func TestImageLoad(t *testing.T) {
	t.Run("interface_has_load_method", func(t *testing.T) {
		// Use reflection to verify that Image interface includes Load method
		// This will fail until Load is added to the interface
		imageType := reflect.TypeOf((*Image)(nil)).Elem()
		loadMethod, ok := imageType.MethodByName("Load")
		if !ok {
			t.Fatal("Image interface does not have Load method")
		}

		// Verify method signature: Load(r image.Rectangle, data []byte) (int, error)
		methodType := loadMethod.Type

		// Should have 2 inputs (receiver excluded for interface types):
		// image.Rectangle and []byte
		if methodType.NumIn() != 2 {
			t.Errorf("Load should have 2 parameters, got %d", methodType.NumIn())
		}

		// Should have 2 outputs: int and error
		if methodType.NumOut() != 2 {
			t.Errorf("Load should have 2 return values, got %d", methodType.NumOut())
		}

		// Check first parameter is image.Rectangle
		if methodType.NumIn() >= 1 {
			param1 := methodType.In(0)
			if param1.String() != "image.Rectangle" {
				t.Errorf("Load first param should be image.Rectangle, got %s", param1.String())
			}
		}

		// Check second parameter is []byte
		if methodType.NumIn() >= 2 {
			param2 := methodType.In(1)
			if param2.Kind() != reflect.Slice || param2.Elem().Kind() != reflect.Uint8 {
				t.Errorf("Load second param should be []byte, got %s", param2.String())
			}
		}

		// Check first return is int
		if methodType.NumOut() >= 1 {
			ret1 := methodType.Out(0)
			if ret1.Kind() != reflect.Int {
				t.Errorf("Load first return should be int, got %s", ret1.String())
			}
		}

		// Check second return is error
		if methodType.NumOut() >= 2 {
			ret2 := methodType.Out(1)
			errorType := reflect.TypeOf((*error)(nil)).Elem()
			if !ret2.Implements(errorType) {
				t.Errorf("Load second return should be error, got %s", ret2.String())
			}
		}
	})
}

// TestImageLoadImplementation tests that imageImpl correctly implements Load.
// This tests the wrapper around the underlying 9fans.net/go/draw.Image.Load.
func TestImageLoadImplementation(t *testing.T) {
	// This test verifies that imageImpl.Load properly delegates to the
	// underlying drawImage.Load method. Since we can't create a real display
	// in unit tests, we verify the method exists and has the correct signature.
	//
	// Once implemented, imageImpl.Load should look like:
	//   func (dst *imageImpl) Load(r image.Rectangle, data []byte) (int, error) {
	//       return dst.drawImage.Load(r, data)
	//   }

	imageImplType := reflect.TypeOf((*imageImpl)(nil))
	loadMethod, ok := imageImplType.MethodByName("Load")
	if !ok {
		t.Fatal("imageImpl does not implement Load method")
	}

	// Verify it returns correct types
	methodType := loadMethod.Type
	// Method on concrete type has receiver as first input
	// So inputs are: receiver, image.Rectangle, []byte
	if methodType.NumIn() != 3 {
		t.Errorf("imageImpl.Load should have 3 inputs (including receiver), got %d", methodType.NumIn())
	}

	if methodType.NumOut() != 2 {
		t.Errorf("imageImpl.Load should have 2 outputs, got %d", methodType.NumOut())
	}
}

// TestPixConstantsExist verifies that the required Pix constants are exported.
// These constants are needed for specifying pixel formats when allocating
// images for image rendering.
//
// Required constants:
// - RGBA32: Red, Green, Blue, Alpha - 8 bits each (32 bits total)
// - RGB24: Red, Green, Blue - 8 bits each (24 bits total, no alpha)
// - ARGB32: Alpha, Red, Green, Blue - 8 bits each (alpha first)
// - XRGB32: Ignored, Red, Green, Blue - 8 bits each (no alpha, padding byte)
func TestPixConstantsExist(t *testing.T) {
	// These tests verify that Pix constants are exported from the draw package.
	// The constants will be added by exporting them from the underlying
	// 9fans.net/go/draw or duitdraw packages.

	tests := []string{"RGBA32", "RGB24", "ARGB32", "XRGB32"}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			// Check if constant exists by looking for it in the package
			// This uses runtime checks since we can't reflect on package constants
			pix, ok := getPixConstant(name)
			if !ok {
				t.Fatalf("Pix constant %s is not exported from draw package", name)
			}
			// Verify it's a valid Pix value (non-zero)
			if pix == 0 {
				t.Errorf("Pix constant %s has zero value", name)
			}
		})
	}
}

// getPixConstant returns the Pix constant with the given name if it exists.
// This function is updated as constants are added to the package.
func getPixConstant(name string) (Pix, bool) {
	// Map of known Pix constants
	constants := map[string]Pix{
		"RGBA32": RGBA32,
		"RGB24":  RGB24,
		"ARGB32": ARGB32,
		"XRGB32": XRGB32,
	}
	pix, ok := constants[name]
	return pix, ok
}

// TestPixConstantsDistinct verifies that Pix constants have distinct values.
// This test will pass once constants are added to getPixConstant.
func TestPixConstantsDistinct(t *testing.T) {
	constants := []string{"RGBA32", "RGB24", "ARGB32", "XRGB32"}
	seen := make(map[Pix]string)

	for _, name := range constants {
		pix, ok := getPixConstant(name)
		if !ok {
			t.Skipf("Constant %s not yet exported", name)
		}
		if existing, dup := seen[pix]; dup {
			t.Errorf("Constants %s and %s have the same value %v", existing, name, pix)
		}
		seen[pix] = name
	}

	if len(seen) == 0 {
		t.Skip("No Pix constants exported yet")
	}
}
