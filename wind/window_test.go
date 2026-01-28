// Package wind provides the Window type and related components for edwood.
// This package contains window state management, preview mode functionality,
// drawing methods, and event handling.
package wind

import (
	"testing"
)

// TestWindowStateNew tests that a new WindowState is properly initialized.
func TestWindowStateNew(t *testing.T) {
	ws := NewWindowState()
	if ws == nil {
		t.Fatal("NewWindowState returned nil")
	}

	// A new WindowState should not be dirty
	if ws.IsDirty() {
		t.Error("new WindowState should not be dirty")
	}
}

// TestWindowStateSetDirty tests the SetDirty method.
func TestWindowStateSetDirty(t *testing.T) {
	ws := NewWindowState()
	ws.SetDirty(true)

	if !ws.IsDirty() {
		t.Error("WindowState should be dirty after SetDirty(true)")
	}

	ws.SetDirty(false)
	if ws.IsDirty() {
		t.Error("WindowState should not be dirty after SetDirty(false)")
	}
}

// TestWindowStateAddr tests the address management methods.
func TestWindowStateAddr(t *testing.T) {
	ws := NewWindowState()

	// Default address should be zero
	addr := ws.Addr()
	if addr.Start != 0 || addr.End != 0 {
		t.Errorf("default address should be (0, 0); got (%d, %d)", addr.Start, addr.End)
	}

	// Set a new address
	ws.SetAddr(Range{Start: 10, End: 20})
	addr = ws.Addr()
	if addr.Start != 10 || addr.End != 20 {
		t.Errorf("address should be (10, 20); got (%d, %d)", addr.Start, addr.End)
	}
}

// TestWindowStateLimit tests the limit management methods.
func TestWindowStateLimit(t *testing.T) {
	ws := NewWindowState()

	// Default limit should be zero
	limit := ws.Limit()
	if limit.Start != 0 || limit.End != 0 {
		t.Errorf("default limit should be (0, 0); got (%d, %d)", limit.Start, limit.End)
	}

	// Set a new limit
	ws.SetLimit(Range{Start: 100, End: 200})
	limit = ws.Limit()
	if limit.Start != 100 || limit.End != 200 {
		t.Errorf("limit should be (100, 200); got (%d, %d)", limit.Start, limit.End)
	}
}

// TestWindowStateNomark tests the nomark flag.
func TestWindowStateNomark(t *testing.T) {
	ws := NewWindowState()

	if ws.Nomark() {
		t.Error("new WindowState should have nomark=false")
	}

	ws.SetNomark(true)
	if !ws.Nomark() {
		t.Error("WindowState should have nomark=true after SetNomark(true)")
	}
}

// TestPreviewStateNew tests that a new PreviewState is properly initialized.
func TestPreviewStateNew(t *testing.T) {
	ps := NewPreviewState()
	if ps == nil {
		t.Fatal("NewPreviewState returned nil")
	}

	// A new PreviewState should not be in preview mode
	if ps.IsPreviewMode() {
		t.Error("new PreviewState should not be in preview mode")
	}
}

// TestPreviewStateSetPreviewMode tests entering and exiting preview mode.
func TestPreviewStateSetPreviewMode(t *testing.T) {
	ps := NewPreviewState()

	ps.SetPreviewMode(true)
	if !ps.IsPreviewMode() {
		t.Error("PreviewState should be in preview mode after SetPreviewMode(true)")
	}

	ps.SetPreviewMode(false)
	if ps.IsPreviewMode() {
		t.Error("PreviewState should not be in preview mode after SetPreviewMode(false)")
	}
}

// TestPreviewStateSourceMap tests source map management.
func TestPreviewStateSourceMap(t *testing.T) {
	ps := NewPreviewState()

	// Default should be nil
	if ps.SourceMap() != nil {
		t.Error("new PreviewState should have nil SourceMap")
	}

	// This test will be expanded when SourceMap type is integrated
}

// TestPreviewStateLinkMap tests link map management.
func TestPreviewStateLinkMap(t *testing.T) {
	ps := NewPreviewState()

	// Default should be nil
	if ps.LinkMap() != nil {
		t.Error("new PreviewState should have nil LinkMap")
	}

	// This test will be expanded when LinkMap type is integrated
}

// TestPreviewStateImageCache tests image cache management.
func TestPreviewStateImageCache(t *testing.T) {
	ps := NewPreviewState()

	// Default should be nil
	if ps.ImageCache() != nil {
		t.Error("new PreviewState should have nil ImageCache")
	}

	// This test will be expanded when ImageCache type is integrated
}

// TestPreviewStateClearCache tests clearing the preview state cache.
func TestPreviewStateClearCache(t *testing.T) {
	ps := NewPreviewState()
	ps.SetPreviewMode(true)

	// ClearCache should not panic on empty state
	ps.ClearCache()

	// After clear, should still be in preview mode (only cache is cleared)
	if !ps.IsPreviewMode() {
		t.Error("ClearCache should not affect preview mode state")
	}
}

// TestPreviewStateDoubleClick tests double-click state tracking.
func TestPreviewStateDoubleClick(t *testing.T) {
	ps := NewPreviewState()

	// Default click position should be 0
	pos, msec := ps.ClickState()
	if pos != 0 || msec != 0 {
		t.Errorf("default click state should be (0, 0); got (%d, %d)", pos, msec)
	}

	// Set click state
	ps.SetClickState(100, 12345)
	pos, msec = ps.ClickState()
	if pos != 100 || msec != 12345 {
		t.Errorf("click state should be (100, 12345); got (%d, %d)", pos, msec)
	}
}
