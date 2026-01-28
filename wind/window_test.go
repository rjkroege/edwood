// Package wind provides the Window type and related components for edwood.
// This package contains window state management, preview mode functionality,
// drawing methods, and event handling.
package wind

import (
	"image"
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

// =============================================================================
// Tests for Window type (Phase 5F)
// These tests validate the core Window functionality once it is moved to this
// package. Until then, they test the supporting types and interfaces.
// =============================================================================

// TestWindowStateIntegration tests that WindowState can be used as part of a
// larger window structure. This validates the interface that Window will use.
func TestWindowStateIntegration(t *testing.T) {
	ws := NewWindowState()

	// Simulate a window becoming dirty
	ws.SetDirty(true)
	if !ws.IsDirty() {
		t.Error("window state should be dirty")
	}

	// Set an address range (like selecting text)
	ws.SetAddr(Range{Start: 10, End: 50})
	addr := ws.Addr()
	if addr.Start != 10 || addr.End != 50 {
		t.Errorf("address range should be (10, 50); got (%d, %d)", addr.Start, addr.End)
	}

	// Set a limit range (for restricted operations)
	ws.SetLimit(Range{Start: 0, End: 100})
	limit := ws.Limit()
	if limit.Start != 0 || limit.End != 100 {
		t.Errorf("limit range should be (0, 100); got (%d, %d)", limit.Start, limit.End)
	}

	// Test nomark flag
	ws.SetNomark(true)
	if !ws.Nomark() {
		t.Error("nomark should be true")
	}
}

// TestDrawStateIntegration tests that DrawState can be used as part of a
// larger window structure for drawing operations.
func TestDrawStateIntegration(t *testing.T) {
	ds := NewDrawState()

	// Simulate a resize operation
	ds.SetRect(image.Rect(0, 0, 800, 600))
	ds.SetTagRect(image.Rect(0, 0, 800, 20))
	ds.SetBodyRect(image.Rect(0, 21, 800, 600))
	ds.SetButtonRect(image.Rect(0, 0, 16, 16))
	ds.SetMaxLines(25)

	// Check that drawing state is correctly set
	if !ds.Rect().Eq(image.Rect(0, 0, 800, 600)) {
		t.Errorf("rect should be (0,0)-(800,600); got %v", ds.Rect())
	}
	if !ds.TagRect().Eq(image.Rect(0, 0, 800, 20)) {
		t.Errorf("tag rect should be (0,0)-(800,20); got %v", ds.TagRect())
	}
	if !ds.BodyRect().Eq(image.Rect(0, 21, 800, 600)) {
		t.Errorf("body rect should be (0,21)-(800,600); got %v", ds.BodyRect())
	}
	if ds.MaxLines() != 25 {
		t.Errorf("max lines should be 25; got %d", ds.MaxLines())
	}

	// Changing state should trigger redraw
	if !ds.NeedsRedraw() {
		t.Error("should need redraw after setting rect")
	}
	ds.ClearRedrawFlag()
	if ds.NeedsRedraw() {
		t.Error("should not need redraw after clearing flag")
	}
}

// TestEventStateIntegration tests that EventState can be used for window
// event processing.
func TestEventStateIntegration(t *testing.T) {
	es := NewEventState()

	// Simulate mouse movement into body area
	tagRect := image.Rect(0, 0, 800, 20)
	bodyRect := image.Rect(0, 21, 800, 600)
	scrollRect := image.Rect(0, 21, 20, 600)

	// Mouse in body (not scrollbar)
	es.UpdateMouseRegion(image.Pt(400, 300), tagRect, bodyRect, scrollRect)
	if !es.IsMouseInBody() {
		t.Error("mouse should be in body")
	}
	if es.IsMouseInTag() {
		t.Error("mouse should not be in tag")
	}
	if es.IsMouseInScrollbar() {
		t.Error("mouse should not be in scrollbar")
	}

	// Simulate a selection operation
	es.SetSelectionActive(true)
	es.SetSelection(100, 200)
	start, end := es.Selection()
	if start != 100 || end != 200 {
		t.Errorf("selection should be (100, 200); got (%d, %d)", start, end)
	}

	// Simulate a double-click
	es.RecordClick(150, 1000)
	es.RecordClick(150, 1300) // within 500ms threshold
	_, _, count := es.ClickState()
	if count != 2 {
		t.Errorf("click count should be 2; got %d", count)
	}
}

// TestPreviewStateIntegration tests that PreviewState can be used for
// markdown preview functionality.
func TestPreviewStateIntegration(t *testing.T) {
	ps := NewPreviewState()

	// Enter preview mode
	ps.SetPreviewMode(true)
	if !ps.IsPreviewMode() {
		t.Error("should be in preview mode")
	}

	// Set double-click state for preview
	ps.SetClickState(50, 2000)
	pos, msec := ps.ClickState()
	if pos != 50 || msec != 2000 {
		t.Errorf("click state should be (50, 2000); got (%d, %d)", pos, msec)
	}

	// Clear cache (should not affect preview mode)
	ps.ClearCache()
	if !ps.IsPreviewMode() {
		t.Error("clearing cache should not exit preview mode")
	}

	// Exit preview mode
	ps.SetPreviewMode(false)
	if ps.IsPreviewMode() {
		t.Error("should not be in preview mode after exit")
	}
}

// TestRangeType tests the Range type used for addresses and limits.
func TestRangeType(t *testing.T) {
	// Zero value
	var r Range
	if r.Start != 0 || r.End != 0 {
		t.Errorf("zero Range should be (0, 0); got (%d, %d)", r.Start, r.End)
	}

	// Initialized value
	r = Range{Start: 10, End: 20}
	if r.Start != 10 || r.End != 20 {
		t.Errorf("Range should be (10, 20); got (%d, %d)", r.Start, r.End)
	}

	// Range can represent empty selection (start == end)
	r = Range{Start: 15, End: 15}
	if r.Start != r.End {
		t.Error("empty range should have start == end")
	}

	// Range can represent backwards selection (start > end is valid)
	r = Range{Start: 30, End: 10}
	if r.Start != 30 || r.End != 10 {
		t.Errorf("backwards Range should be preserved; got (%d, %d)", r.Start, r.End)
	}
}

// TestAllStateTypesIndependent verifies that state types can be used independently.
// This is important for the Window extraction to ensure clean separation.
func TestAllStateTypesIndependent(t *testing.T) {
	// Create all state types independently
	ws := NewWindowState()
	ds := NewDrawState()
	es := NewEventState()
	ps := NewPreviewState()

	// Each should be non-nil and independently usable
	if ws == nil || ds == nil || es == nil || ps == nil {
		t.Fatal("all state types should be created successfully")
	}

	// Modify one without affecting others
	ws.SetDirty(true)
	ds.SetPreviewMode(true)
	es.SetMouseInTag(true)
	ps.SetPreviewMode(true)

	// Verify independence
	if !ws.IsDirty() {
		t.Error("WindowState dirty flag should be set")
	}
	if !ds.IsPreviewMode() {
		t.Error("DrawState preview mode should be set")
	}
	if !es.IsMouseInTag() {
		t.Error("EventState mouseInTag should be set")
	}
	if !ps.IsPreviewMode() {
		t.Error("PreviewState preview mode should be set")
	}
}

// TestWindowStateAddrClamp tests address clamping behavior that Window.ClampAddr
// will need. This test validates the Range type for this use case.
func TestWindowStateAddrClamp(t *testing.T) {
	ws := NewWindowState()

	testCases := []struct {
		name     string
		input    Range
		maxLen   int
		expected Range
	}{
		{"negative start", Range{Start: -5, End: 10}, 100, Range{Start: 0, End: 10}},
		{"negative end", Range{Start: 0, End: -5}, 100, Range{Start: 0, End: 0}},
		{"beyond max", Range{Start: 50, End: 150}, 100, Range{Start: 50, End: 100}},
		{"both beyond", Range{Start: 200, End: 300}, 100, Range{Start: 100, End: 100}},
		{"valid range", Range{Start: 10, End: 50}, 100, Range{Start: 10, End: 50}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set the address
			ws.SetAddr(tc.input)

			// Simulate clamping (this logic will be in Window.ClampAddr)
			addr := ws.Addr()
			if addr.Start < 0 {
				addr.Start = 0
			}
			if addr.End < 0 {
				addr.End = 0
			}
			if addr.Start > tc.maxLen {
				addr.Start = tc.maxLen
			}
			if addr.End > tc.maxLen {
				addr.End = tc.maxLen
			}
			ws.SetAddr(addr)

			result := ws.Addr()
			if result.Start != tc.expected.Start || result.End != tc.expected.End {
				t.Errorf("clamped addr should be (%d, %d); got (%d, %d)",
					tc.expected.Start, tc.expected.End, result.Start, result.End)
			}
		})
	}
}

// =============================================================================
// Tests for WindowBase (Phase 5F)
// These tests validate the WindowBase struct that provides portable window state.
// =============================================================================

// TestWindowBaseNew tests that NewWindowBase creates a properly initialized struct.
func TestWindowBaseNew(t *testing.T) {
	wb := NewWindowBase()
	if wb == nil {
		t.Fatal("NewWindowBase returned nil")
	}

	// All state components should be initialized
	if wb.State == nil {
		t.Error("WindowBase.State should not be nil")
	}
	if wb.Draw == nil {
		t.Error("WindowBase.Draw should not be nil")
	}
	if wb.Events == nil {
		t.Error("WindowBase.Events should not be nil")
	}
	if wb.Preview == nil {
		t.Error("WindowBase.Preview should not be nil")
	}
}

// TestWindowBaseImplementsWindow verifies WindowBase implements the Window interface.
func TestWindowBaseImplementsWindow(t *testing.T) {
	var _ Window = (*WindowBase)(nil)
	var _ Window = NewWindowBase()
}

// TestWindowBaseID tests the ID getter and setter.
func TestWindowBaseID(t *testing.T) {
	wb := NewWindowBase()

	// Default ID should be 0
	if wb.ID() != 0 {
		t.Errorf("default ID should be 0; got %d", wb.ID())
	}

	// Set ID
	wb.SetID(42)
	if wb.ID() != 42 {
		t.Errorf("ID should be 42; got %d", wb.ID())
	}
}

// TestWindowBaseRect tests the Rect getter and setter.
func TestWindowBaseRect(t *testing.T) {
	wb := NewWindowBase()

	// Default rect should be zero
	if !wb.Rect().Empty() {
		t.Errorf("default rect should be empty; got %v", wb.Rect())
	}

	// Set rect
	r := image.Rect(0, 0, 800, 600)
	wb.SetRect(r)
	if !wb.Rect().Eq(r) {
		t.Errorf("rect should be %v; got %v", r, wb.Rect())
	}

	// SetRect should also update Draw.Rect
	if !wb.Draw.Rect().Eq(r) {
		t.Errorf("Draw.Rect should be %v; got %v", r, wb.Draw.Rect())
	}
}

// TestWindowBasePreviewMode tests preview mode toggling.
func TestWindowBasePreviewMode(t *testing.T) {
	wb := NewWindowBase()

	// Default should not be in preview mode
	if wb.IsPreviewMode() {
		t.Error("default should not be in preview mode")
	}

	// Enable preview mode
	wb.SetPreviewMode(true)
	if !wb.IsPreviewMode() {
		t.Error("should be in preview mode after SetPreviewMode(true)")
	}

	// SetPreviewMode should also update Preview and Draw state
	if !wb.Preview.IsPreviewMode() {
		t.Error("Preview.IsPreviewMode should be true")
	}
	if !wb.Draw.IsPreviewMode() {
		t.Error("Draw.IsPreviewMode should be true")
	}

	// Disable preview mode
	wb.SetPreviewMode(false)
	if wb.IsPreviewMode() {
		t.Error("should not be in preview mode after SetPreviewMode(false)")
	}
}

// TestWindowBaseDirty tests dirty flag management.
func TestWindowBaseDirty(t *testing.T) {
	wb := NewWindowBase()

	// Default should not be dirty
	if wb.IsDirty() {
		t.Error("default should not be dirty")
	}

	// Set dirty
	wb.SetDirty(true)
	if !wb.IsDirty() {
		t.Error("should be dirty after SetDirty(true)")
	}

	// SetDirty should also update State and Draw
	if !wb.State.IsDirty() {
		t.Error("State.IsDirty should be true")
	}
	if !wb.Draw.IsDirty() {
		t.Error("Draw.IsDirty should be true")
	}

	// Clear dirty
	wb.SetDirty(false)
	if wb.IsDirty() {
		t.Error("should not be dirty after SetDirty(false)")
	}
}

// TestWindowBaseAddr tests address range management.
func TestWindowBaseAddr(t *testing.T) {
	wb := NewWindowBase()

	// Default address should be zero
	addr := wb.Addr()
	if addr.Start != 0 || addr.End != 0 {
		t.Errorf("default addr should be (0, 0); got (%d, %d)", addr.Start, addr.End)
	}

	// Set address
	wb.SetAddr(Range{Start: 10, End: 50})
	addr = wb.Addr()
	if addr.Start != 10 || addr.End != 50 {
		t.Errorf("addr should be (10, 50); got (%d, %d)", addr.Start, addr.End)
	}

	// Should be delegating to State
	stateAddr := wb.State.Addr()
	if stateAddr.Start != 10 || stateAddr.End != 50 {
		t.Errorf("State.Addr should be (10, 50); got (%d, %d)", stateAddr.Start, stateAddr.End)
	}
}

// TestWindowBaseLimit tests limit range management.
func TestWindowBaseLimit(t *testing.T) {
	wb := NewWindowBase()

	// Default limit should be zero
	limit := wb.Limit()
	if limit.Start != 0 || limit.End != 0 {
		t.Errorf("default limit should be (0, 0); got (%d, %d)", limit.Start, limit.End)
	}

	// Set limit
	wb.SetLimit(Range{Start: 0, End: 100})
	limit = wb.Limit()
	if limit.Start != 0 || limit.End != 100 {
		t.Errorf("limit should be (0, 100); got (%d, %d)", limit.Start, limit.End)
	}
}

// TestWindowBaseNomark tests nomark flag management.
func TestWindowBaseNomark(t *testing.T) {
	wb := NewWindowBase()

	// Default should be false
	if wb.Nomark() {
		t.Error("default nomark should be false")
	}

	// Set nomark
	wb.SetNomark(true)
	if !wb.Nomark() {
		t.Error("nomark should be true after SetNomark(true)")
	}
}

// TestWindowBaseTagLines tests tag line management.
func TestWindowBaseTagLines(t *testing.T) {
	wb := NewWindowBase()

	// Default should be 1
	if wb.TagLines() != 1 {
		t.Errorf("default tag lines should be 1; got %d", wb.TagLines())
	}

	// Set tag lines
	wb.SetTagLines(3)
	if wb.TagLines() != 3 {
		t.Errorf("tag lines should be 3; got %d", wb.TagLines())
	}

	// Setting to 0 should clamp to 1
	wb.SetTagLines(0)
	if wb.TagLines() != 1 {
		t.Errorf("tag lines should clamp to 1; got %d", wb.TagLines())
	}
}

// TestWindowBaseTagExpand tests tag expand management.
func TestWindowBaseTagExpand(t *testing.T) {
	wb := NewWindowBase()

	// Default should be true
	if !wb.TagExpand() {
		t.Error("default tag expand should be true")
	}

	// Set tag expand
	wb.SetTagExpand(false)
	if wb.TagExpand() {
		t.Error("tag expand should be false after SetTagExpand(false)")
	}
}

// TestWindowBaseMaxLines tests max lines management.
func TestWindowBaseMaxLines(t *testing.T) {
	wb := NewWindowBase()

	// Default should be 0
	if wb.MaxLines() != 0 {
		t.Errorf("default max lines should be 0; got %d", wb.MaxLines())
	}

	// Set max lines
	wb.SetMaxLines(25)
	if wb.MaxLines() != 25 {
		t.Errorf("max lines should be 25; got %d", wb.MaxLines())
	}
}

// TestWindowBaseMouseRegion tests mouse region tracking.
func TestWindowBaseMouseRegion(t *testing.T) {
	wb := NewWindowBase()

	// Default should not be in any region
	if wb.IsMouseInTag() || wb.IsMouseInBody() || wb.IsMouseInScrollbar() {
		t.Error("default should not be in any mouse region")
	}

	// Set up regions
	tagRect := image.Rect(0, 0, 800, 20)
	bodyRect := image.Rect(0, 21, 800, 600)
	scrollRect := image.Rect(0, 21, 20, 600)

	// Test mouse in tag
	wb.UpdateMouseRegion(image.Pt(400, 10), tagRect, bodyRect, scrollRect)
	if !wb.IsMouseInTag() {
		t.Error("mouse should be in tag")
	}
	if wb.IsMouseInBody() {
		t.Error("mouse should not be in body")
	}

	// Test mouse in body (not scrollbar)
	wb.UpdateMouseRegion(image.Pt(400, 300), tagRect, bodyRect, scrollRect)
	if wb.IsMouseInTag() {
		t.Error("mouse should not be in tag")
	}
	if !wb.IsMouseInBody() {
		t.Error("mouse should be in body")
	}
	if wb.IsMouseInScrollbar() {
		t.Error("mouse should not be in scrollbar")
	}

	// Test mouse in scrollbar (within body)
	wb.UpdateMouseRegion(image.Pt(10, 300), tagRect, bodyRect, scrollRect)
	if !wb.IsMouseInBody() {
		t.Error("mouse should be in body")
	}
	if !wb.IsMouseInScrollbar() {
		t.Error("mouse should be in scrollbar")
	}
}

// TestWindowBaseClickState tests click state for double-click detection.
func TestWindowBaseClickState(t *testing.T) {
	wb := NewWindowBase()

	// Default should be (0, 0)
	pos, msec := wb.ClickState()
	if pos != 0 || msec != 0 {
		t.Errorf("default click state should be (0, 0); got (%d, %d)", pos, msec)
	}

	// Set click state
	wb.SetClickState(100, 12345)
	pos, msec = wb.ClickState()
	if pos != 100 || msec != 12345 {
		t.Errorf("click state should be (100, 12345); got (%d, %d)", pos, msec)
	}
}

// TestWindowBaseRedraw tests redraw flag management.
func TestWindowBaseRedraw(t *testing.T) {
	wb := NewWindowBase()

	// Default should not need redraw
	if wb.NeedsRedraw() {
		t.Error("default should not need redraw")
	}

	// Setting rect should trigger redraw
	wb.SetRect(image.Rect(0, 0, 800, 600))
	if !wb.NeedsRedraw() {
		t.Error("should need redraw after setting rect")
	}

	// Clear redraw flag
	wb.ClearRedrawFlag()
	if wb.NeedsRedraw() {
		t.Error("should not need redraw after clearing flag")
	}
}

// TestWindowBaseClearPreviewCache tests clearing preview cache.
func TestWindowBaseClearPreviewCache(t *testing.T) {
	wb := NewWindowBase()

	// Set preview mode first
	wb.SetPreviewMode(true)

	// Clear cache should not panic and should not affect preview mode
	wb.ClearPreviewCache()
	if !wb.IsPreviewMode() {
		t.Error("clearing cache should not affect preview mode")
	}
}

// TestWindowBaseResetEventState tests resetting event state.
func TestWindowBaseResetEventState(t *testing.T) {
	wb := NewWindowBase()

	// Set some event state
	wb.Events.SetMouseInTag(true)
	wb.Events.SetSelectionActive(true)
	wb.Events.SetSelection(10, 20)

	// Reset
	wb.ResetEventState()

	// Everything should be reset
	if wb.IsMouseInTag() {
		t.Error("mouse in tag should be reset")
	}
	if wb.Events.IsSelectionActive() {
		t.Error("selection active should be reset")
	}
	start, end := wb.Events.Selection()
	if start != 0 || end != 0 {
		t.Errorf("selection should be reset; got (%d, %d)", start, end)
	}
}

// TestWindowBaseComposition tests that WindowBase properly composes state types.
func TestWindowBaseComposition(t *testing.T) {
	wb := NewWindowBase()

	// Simulate a complete window setup
	wb.SetID(1)
	wb.SetRect(image.Rect(0, 0, 800, 600))
	wb.SetTagRect(image.Rect(0, 0, 800, 20))
	wb.SetBodyRect(image.Rect(0, 21, 800, 600))
	wb.SetButtonRect(image.Rect(0, 0, 16, 16))
	wb.SetTagLines(2)
	wb.SetMaxLines(25)
	wb.SetAddr(Range{Start: 100, End: 200})
	wb.SetLimit(Range{Start: 0, End: 1000})
	wb.SetDirty(true)
	wb.SetPreviewMode(true)

	// Verify all state is correctly set across components
	if wb.ID() != 1 {
		t.Errorf("ID should be 1; got %d", wb.ID())
	}
	if !wb.Rect().Eq(image.Rect(0, 0, 800, 600)) {
		t.Errorf("Rect mismatch; got %v", wb.Rect())
	}
	if wb.TagLines() != 2 {
		t.Errorf("TagLines should be 2; got %d", wb.TagLines())
	}
	if wb.MaxLines() != 25 {
		t.Errorf("MaxLines should be 25; got %d", wb.MaxLines())
	}
	addr := wb.Addr()
	if addr.Start != 100 || addr.End != 200 {
		t.Errorf("Addr should be (100, 200); got (%d, %d)", addr.Start, addr.End)
	}
	if !wb.IsDirty() {
		t.Error("should be dirty")
	}
	if !wb.IsPreviewMode() {
		t.Error("should be in preview mode")
	}

	// Verify state is synchronized across components
	if !wb.State.IsDirty() {
		t.Error("State.IsDirty should match")
	}
	if !wb.Draw.IsDirty() {
		t.Error("Draw.IsDirty should match")
	}
	if !wb.Preview.IsPreviewMode() {
		t.Error("Preview.IsPreviewMode should match")
	}
	if !wb.Draw.IsPreviewMode() {
		t.Error("Draw.IsPreviewMode should match")
	}
}

// TestWindowInterfacePolymorphism tests that Window interface can be used polymorphically.
func TestWindowInterfacePolymorphism(t *testing.T) {
	// Create a WindowBase and use it as Window interface
	wb := NewWindowBase()
	wb.SetID(42)
	wb.SetRect(image.Rect(0, 0, 800, 600))
	wb.SetPreviewMode(true)

	// Use as Window interface
	var w Window = wb

	if w.ID() != 42 {
		t.Errorf("Window.ID() should be 42; got %d", w.ID())
	}
	if !w.Rect().Eq(image.Rect(0, 0, 800, 600)) {
		t.Errorf("Window.Rect() mismatch; got %v", w.Rect())
	}
	if !w.IsPreviewMode() {
		t.Error("Window.IsPreviewMode() should be true")
	}
	if w.IsDirty() {
		t.Error("Window.IsDirty() should be false")
	}
}
