// Package wind provides the Window type and related components for edwood.
// This file contains tests for window drawing functionality.
package wind

import (
	"image"
	"testing"
)

// MockDrawContext is a test double for DrawContext.
type MockDrawContext struct {
	rect        image.Rectangle
	previewMode bool
}

// NewMockDrawContext creates a new MockDrawContext with default values.
func NewMockDrawContext() *MockDrawContext {
	return &MockDrawContext{
		rect: image.Rect(0, 0, 800, 600),
	}
}

// Rect implements DrawContext.
func (m *MockDrawContext) Rect() image.Rectangle {
	return m.rect
}

// SetRect sets the drawing rectangle.
func (m *MockDrawContext) SetRect(r image.Rectangle) {
	m.rect = r
}

// IsPreviewMode implements DrawContext.
func (m *MockDrawContext) IsPreviewMode() bool {
	return m.previewMode
}

// SetPreviewMode sets the preview mode state.
func (m *MockDrawContext) SetPreviewMode(mode bool) {
	m.previewMode = mode
}

// TestMockDrawContextNew tests that a new MockDrawContext has default values.
func TestMockDrawContextNew(t *testing.T) {
	dc := NewMockDrawContext()
	if dc == nil {
		t.Fatal("NewMockDrawContext returned nil")
	}

	// Default rectangle should be 800x600
	r := dc.Rect()
	if r.Dx() != 800 || r.Dy() != 600 {
		t.Errorf("default rect should be 800x600; got %dx%d", r.Dx(), r.Dy())
	}

	// Default preview mode should be false
	if dc.IsPreviewMode() {
		t.Error("default preview mode should be false")
	}
}

// TestMockDrawContextRect tests the Rect getter/setter.
func TestMockDrawContextRect(t *testing.T) {
	dc := NewMockDrawContext()

	newRect := image.Rect(100, 100, 500, 400)
	dc.SetRect(newRect)

	r := dc.Rect()
	if !r.Eq(newRect) {
		t.Errorf("rect should be %v; got %v", newRect, r)
	}
}

// TestMockDrawContextPreviewMode tests the PreviewMode getter/setter.
func TestMockDrawContextPreviewMode(t *testing.T) {
	dc := NewMockDrawContext()

	dc.SetPreviewMode(true)
	if !dc.IsPreviewMode() {
		t.Error("preview mode should be true after SetPreviewMode(true)")
	}

	dc.SetPreviewMode(false)
	if dc.IsPreviewMode() {
		t.Error("preview mode should be false after SetPreviewMode(false)")
	}
}

// TestDrawStateNew tests that a new DrawState has correct defaults.
func TestDrawStateNew(t *testing.T) {
	ds := NewDrawState()
	if ds == nil {
		t.Fatal("NewDrawState returned nil")
	}

	if ds.IsDirty() {
		t.Error("new DrawState should not be dirty")
	}

	if ds.IsPreviewMode() {
		t.Error("new DrawState should not be in preview mode")
	}

	if ds.TagLines() != 1 {
		t.Errorf("new DrawState should have 1 tag line; got %d", ds.TagLines())
	}

	if !ds.TagExpand() {
		t.Error("new DrawState should have tagExpand=true")
	}
}

// TestDrawStateDirty tests the dirty flag behavior.
func TestDrawStateDirty(t *testing.T) {
	ds := NewDrawState()

	ds.SetDirty(true)
	if !ds.IsDirty() {
		t.Error("DrawState should be dirty after SetDirty(true)")
	}
	if !ds.NeedsRedraw() {
		t.Error("setting dirty should trigger redraw")
	}

	ds.ClearRedrawFlag()
	ds.SetDirty(false)
	if ds.IsDirty() {
		t.Error("DrawState should not be dirty after SetDirty(false)")
	}
	if !ds.NeedsRedraw() {
		t.Error("clearing dirty should trigger redraw")
	}
}

// TestDrawStateRect tests rectangle management.
func TestDrawStateRect(t *testing.T) {
	ds := NewDrawState()

	r := image.Rect(0, 0, 800, 600)
	ds.SetRect(r)

	if !ds.Rect().Eq(r) {
		t.Errorf("rect should be %v; got %v", r, ds.Rect())
	}

	if !ds.NeedsRedraw() {
		t.Error("changing rect should trigger redraw")
	}
}

// TestDrawStateTagRect tests tag rectangle management.
func TestDrawStateTagRect(t *testing.T) {
	ds := NewDrawState()

	r := image.Rect(0, 0, 800, 20)
	ds.SetTagRect(r)

	if !ds.TagRect().Eq(r) {
		t.Errorf("tag rect should be %v; got %v", r, ds.TagRect())
	}
}

// TestDrawStateBodyRect tests body rectangle management.
func TestDrawStateBodyRect(t *testing.T) {
	ds := NewDrawState()

	r := image.Rect(0, 21, 800, 600)
	ds.SetBodyRect(r)

	if !ds.BodyRect().Eq(r) {
		t.Errorf("body rect should be %v; got %v", r, ds.BodyRect())
	}
}

// TestDrawStateButtonRect tests button rectangle management.
func TestDrawStateButtonRect(t *testing.T) {
	ds := NewDrawState()

	r := image.Rect(0, 0, 16, 16)
	ds.SetButtonRect(r)

	if !ds.ButtonRect().Eq(r) {
		t.Errorf("button rect should be %v; got %v", r, ds.ButtonRect())
	}
}

// TestDrawStateMaxLines tests maximum lines management.
func TestDrawStateMaxLines(t *testing.T) {
	ds := NewDrawState()

	ds.SetMaxLines(50)
	if ds.MaxLines() != 50 {
		t.Errorf("max lines should be 50; got %d", ds.MaxLines())
	}
}

// TestDrawStatePreviewMode tests preview mode state.
func TestDrawStatePreviewMode(t *testing.T) {
	ds := NewDrawState()

	ds.SetPreviewMode(true)
	if !ds.IsPreviewMode() {
		t.Error("should be in preview mode after SetPreviewMode(true)")
	}
	if !ds.NeedsRedraw() {
		t.Error("changing preview mode should trigger redraw")
	}

	ds.ClearRedrawFlag()
	ds.SetPreviewMode(false)
	if ds.IsPreviewMode() {
		t.Error("should not be in preview mode after SetPreviewMode(false)")
	}
	if !ds.NeedsRedraw() {
		t.Error("changing preview mode should trigger redraw")
	}
}

// TestDrawStateTagLines tests tag lines management.
func TestDrawStateTagLines(t *testing.T) {
	ds := NewDrawState()

	ds.SetTagLines(3)
	if ds.TagLines() != 3 {
		t.Errorf("tag lines should be 3; got %d", ds.TagLines())
	}

	// Setting to 0 should clamp to 1
	ds.SetTagLines(0)
	if ds.TagLines() != 1 {
		t.Errorf("tag lines should be clamped to 1; got %d", ds.TagLines())
	}

	// Setting to negative should clamp to 1
	ds.SetTagLines(-5)
	if ds.TagLines() != 1 {
		t.Errorf("tag lines should be clamped to 1; got %d", ds.TagLines())
	}
}

// TestDrawStateTagExpand tests tag expand behavior.
func TestDrawStateTagExpand(t *testing.T) {
	ds := NewDrawState()

	// Default is true
	if !ds.TagExpand() {
		t.Error("default tag expand should be true")
	}

	ds.SetTagExpand(false)
	if ds.TagExpand() {
		t.Error("tag expand should be false after SetTagExpand(false)")
	}

	ds.SetTagExpand(true)
	if !ds.TagExpand() {
		t.Error("tag expand should be true after SetTagExpand(true)")
	}
}

// TestDrawStateNeedsRedraw tests redraw flag management.
func TestDrawStateNeedsRedraw(t *testing.T) {
	ds := NewDrawState()

	// Initially should not need redraw
	if ds.NeedsRedraw() {
		t.Error("new DrawState should not need redraw")
	}

	// Setting rect should trigger redraw
	ds.SetRect(image.Rect(0, 0, 100, 100))
	if !ds.NeedsRedraw() {
		t.Error("should need redraw after SetRect")
	}

	// Clear and verify
	ds.ClearRedrawFlag()
	if ds.NeedsRedraw() {
		t.Error("should not need redraw after ClearRedrawFlag")
	}

	// Setting dirty should trigger redraw
	ds.SetDirty(true)
	if !ds.NeedsRedraw() {
		t.Error("should need redraw after SetDirty")
	}
}

// TestDrawStateRectNoChangeNoRedraw tests that setting same rect doesn't trigger redraw.
func TestDrawStateRectNoChangeNoRedraw(t *testing.T) {
	ds := NewDrawState()
	r := image.Rect(0, 0, 100, 100)

	ds.SetRect(r)
	ds.ClearRedrawFlag()

	// Setting the same rect should not trigger redraw
	ds.SetRect(r)
	if ds.NeedsRedraw() {
		t.Error("setting same rect should not trigger redraw")
	}
}

// TestDrawStateDirtyNoChangeNoRedraw tests that setting same dirty state doesn't trigger redraw.
func TestDrawStateDirtyNoChangeNoRedraw(t *testing.T) {
	ds := NewDrawState()

	ds.SetDirty(true)
	ds.ClearRedrawFlag()

	// Setting the same dirty state should not trigger redraw
	ds.SetDirty(true)
	if ds.NeedsRedraw() {
		t.Error("setting same dirty state should not trigger redraw")
	}
}

// TestDrawStatePreviewModeNoChangeNoRedraw tests that setting same preview mode doesn't trigger redraw.
func TestDrawStatePreviewModeNoChangeNoRedraw(t *testing.T) {
	ds := NewDrawState()

	ds.SetPreviewMode(true)
	ds.ClearRedrawFlag()

	// Setting the same preview mode should not trigger redraw
	ds.SetPreviewMode(true)
	if ds.NeedsRedraw() {
		t.Error("setting same preview mode should not trigger redraw")
	}
}
