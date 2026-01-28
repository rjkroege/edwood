// Package wind provides the Window type and related components for edwood.
// This file contains tests for window event handling functionality.
package wind

import (
	"image"
	"testing"
)

// TestMouseButtonConstants verifies that mouse button constants are correct.
func TestMouseButtonConstants(t *testing.T) {
	// Verify bit values match expected patterns
	if MouseB1 != 1 {
		t.Errorf("MouseB1 should be 1; got %d", MouseB1)
	}
	if MouseB2 != 2 {
		t.Errorf("MouseB2 should be 2; got %d", MouseB2)
	}
	if MouseB3 != 4 {
		t.Errorf("MouseB3 should be 4; got %d", MouseB3)
	}
	if MouseB4 != 8 {
		t.Errorf("MouseB4 should be 8; got %d", MouseB4)
	}
	if MouseB5 != 16 {
		t.Errorf("MouseB5 should be 16; got %d", MouseB5)
	}
}

// TestEventTypeConstants verifies that event type constants are defined.
func TestEventTypeConstants(t *testing.T) {
	if EventNone != 0 {
		t.Errorf("EventNone should be 0; got %d", EventNone)
	}
	if EventMouseClick <= EventNone {
		t.Error("EventMouseClick should be > EventNone")
	}
	if EventMouseDrag <= EventMouseClick {
		t.Error("EventMouseDrag should be > EventMouseClick")
	}
	if EventMouseScroll <= EventMouseDrag {
		t.Error("EventMouseScroll should be > EventMouseDrag")
	}
	if EventKey <= EventMouseScroll {
		t.Error("EventKey should be > EventMouseScroll")
	}
}

// TestEventStateNew tests that a new EventState has correct defaults.
func TestEventStateNew(t *testing.T) {
	es := NewEventState()
	if es == nil {
		t.Fatal("NewEventState returned nil")
	}

	// Check default values
	if es.LastMousePos() != (image.Point{}) {
		t.Error("new EventState should have zero mouse position")
	}
	if es.LastMouseButtons() != 0 {
		t.Error("new EventState should have no buttons pressed")
	}
	if es.IsMouseInTag() {
		t.Error("new EventState should not have mouse in tag")
	}
	if es.IsMouseInBody() {
		t.Error("new EventState should not have mouse in body")
	}
	if es.IsMouseInScrollbar() {
		t.Error("new EventState should not have mouse in scrollbar")
	}
	if es.IsSelectionActive() {
		t.Error("new EventState should not have active selection")
	}
	if es.IsChordActive() {
		t.Error("new EventState should not have active chord")
	}
	if es.IsScrollLatched() {
		t.Error("new EventState should not be scroll latched")
	}
	if es.CurrentEvent() != EventNone {
		t.Error("new EventState should have EventNone")
	}
}

// TestEventStateMousePos tests mouse position tracking.
func TestEventStateMousePos(t *testing.T) {
	es := NewEventState()

	pos := image.Pt(100, 200)
	es.SetLastMousePos(pos)

	if es.LastMousePos() != pos {
		t.Errorf("mouse position should be %v; got %v", pos, es.LastMousePos())
	}
}

// TestEventStateMouseButtons tests mouse button tracking.
func TestEventStateMouseButtons(t *testing.T) {
	es := NewEventState()

	es.SetLastMouseButtons(MouseB1 | MouseB3)

	buttons := es.LastMouseButtons()
	if buttons&MouseB1 == 0 {
		t.Error("MouseB1 should be set")
	}
	if buttons&MouseB2 != 0 {
		t.Error("MouseB2 should not be set")
	}
	if buttons&MouseB3 == 0 {
		t.Error("MouseB3 should be set")
	}
}

// TestEventStateMouseInTag tests mouse-in-tag tracking.
func TestEventStateMouseInTag(t *testing.T) {
	es := NewEventState()

	es.SetMouseInTag(true)
	if !es.IsMouseInTag() {
		t.Error("mouse should be in tag after SetMouseInTag(true)")
	}

	es.SetMouseInTag(false)
	if es.IsMouseInTag() {
		t.Error("mouse should not be in tag after SetMouseInTag(false)")
	}
}

// TestEventStateMouseInBody tests mouse-in-body tracking.
func TestEventStateMouseInBody(t *testing.T) {
	es := NewEventState()

	es.SetMouseInBody(true)
	if !es.IsMouseInBody() {
		t.Error("mouse should be in body after SetMouseInBody(true)")
	}

	es.SetMouseInBody(false)
	if es.IsMouseInBody() {
		t.Error("mouse should not be in body after SetMouseInBody(false)")
	}
}

// TestEventStateMouseInScrollbar tests mouse-in-scrollbar tracking.
func TestEventStateMouseInScrollbar(t *testing.T) {
	es := NewEventState()

	es.SetMouseInScrollbar(true)
	if !es.IsMouseInScrollbar() {
		t.Error("mouse should be in scrollbar after SetMouseInScrollbar(true)")
	}

	es.SetMouseInScrollbar(false)
	if es.IsMouseInScrollbar() {
		t.Error("mouse should not be in scrollbar after SetMouseInScrollbar(false)")
	}
}

// TestEventStateSelection tests selection tracking.
func TestEventStateSelection(t *testing.T) {
	es := NewEventState()

	es.SetSelectionActive(true)
	if !es.IsSelectionActive() {
		t.Error("selection should be active")
	}

	es.SetSelection(10, 50)
	start, end := es.Selection()
	if start != 10 {
		t.Errorf("selection start should be 10; got %d", start)
	}
	if end != 50 {
		t.Errorf("selection end should be 50; got %d", end)
	}

	pt := image.Pt(150, 250)
	es.SetSelectPoint(pt)
	if es.SelectPoint() != pt {
		t.Errorf("select point should be %v; got %v", pt, es.SelectPoint())
	}
}

// TestEventStateClickState tests click state tracking.
func TestEventStateClickState(t *testing.T) {
	es := NewEventState()

	es.SetClickState(42, 1000, 2)
	pos, msec, count := es.ClickState()
	if pos != 42 {
		t.Errorf("click pos should be 42; got %d", pos)
	}
	if msec != 1000 {
		t.Errorf("click msec should be 1000; got %d", msec)
	}
	if count != 2 {
		t.Errorf("click count should be 2; got %d", count)
	}
}

// TestEventStateLastClickPos tests last click position tracking.
func TestEventStateLastClickPos(t *testing.T) {
	es := NewEventState()

	es.SetLastClickPos(99)
	if es.LastClickPos() != 99 {
		t.Errorf("last click pos should be 99; got %d", es.LastClickPos())
	}
}

// TestEventStateDoubleClick tests double-click detection.
func TestEventStateDoubleClick(t *testing.T) {
	es := NewEventState()

	// First click at position 10, time 1000
	es.RecordClick(10, 1000)
	pos, msec, count := es.ClickState()
	if pos != 10 || msec != 1000 || count != 1 {
		t.Errorf("first click: pos=%d, msec=%d, count=%d; expected 10, 1000, 1", pos, msec, count)
	}

	// Second click at same position, within threshold
	es.RecordClick(10, 1400)
	pos, msec, count = es.ClickState()
	if count != 2 {
		t.Errorf("double-click: count should be 2; got %d", count)
	}

	// Third click at same position, within threshold
	es.RecordClick(10, 1700)
	pos, msec, count = es.ClickState()
	if count != 3 {
		t.Errorf("triple-click: count should be 3; got %d", count)
	}

	// Click at different position resets count
	es.RecordClick(20, 2100)
	pos, msec, count = es.ClickState()
	if count != 1 {
		t.Errorf("new position click: count should be 1; got %d", count)
	}
}

// TestEventStateDoubleClickTimeout tests double-click timeout.
func TestEventStateDoubleClickTimeout(t *testing.T) {
	es := NewEventState()

	// First click
	es.RecordClick(10, 1000)

	// Second click after timeout
	es.RecordClick(10, 2000) // 1000ms later, > 500ms threshold
	_, _, count := es.ClickState()
	if count != 1 {
		t.Errorf("click after timeout: count should be 1; got %d", count)
	}
}

// TestEventStateIsDoubleClick tests the IsDoubleClick method.
func TestEventStateIsDoubleClick(t *testing.T) {
	es := NewEventState()

	// No prior click
	if es.IsDoubleClick(10, 1000, 500) {
		t.Error("should not be double-click with no prior click")
	}

	// First click
	es.RecordClick(10, 1000)

	// Same position, within threshold
	if !es.IsDoubleClick(10, 1400, 500) {
		t.Error("should be double-click at same position within threshold")
	}

	// Same position, after threshold
	if es.IsDoubleClick(10, 1600, 500) {
		t.Error("should not be double-click after threshold")
	}

	// Different position, within threshold
	if es.IsDoubleClick(20, 1400, 500) {
		t.Error("should not be double-click at different position")
	}
}

// TestEventStateClearClickState tests clearing click state.
func TestEventStateClearClickState(t *testing.T) {
	es := NewEventState()

	es.RecordClick(10, 1000)
	es.RecordClick(10, 1400) // double-click
	es.ClearClickState()

	pos, msec, count := es.ClickState()
	if pos != 0 || msec != 0 || count != 0 {
		t.Errorf("after clear: pos=%d, msec=%d, count=%d; expected all 0", pos, msec, count)
	}
	if es.LastClickPos() != 0 {
		t.Error("after clear: last click pos should be 0")
	}
}

// TestEventStateChord tests chord state tracking.
func TestEventStateChord(t *testing.T) {
	es := NewEventState()

	es.StartChord(MouseB1 | MouseB2)
	if !es.IsChordActive() {
		t.Error("chord should be active after StartChord")
	}
	buttons, active := es.ChordState()
	if buttons != MouseB1|MouseB2 {
		t.Errorf("chord buttons should be B1|B2; got %d", buttons)
	}
	if !active {
		t.Error("chord should be active")
	}

	es.EndChord()
	if es.IsChordActive() {
		t.Error("chord should not be active after EndChord")
	}
	buttons, active = es.ChordState()
	if buttons != 0 {
		t.Errorf("chord buttons should be 0 after EndChord; got %d", buttons)
	}
}

// TestEventStateSetChordState tests setting chord state directly.
func TestEventStateSetChordState(t *testing.T) {
	es := NewEventState()

	es.SetChordState(MouseB1|MouseB3, true)
	buttons, active := es.ChordState()
	if buttons != MouseB1|MouseB3 {
		t.Errorf("chord buttons should be B1|B3; got %d", buttons)
	}
	if !active {
		t.Error("chord should be active")
	}
}

// TestEventStateScrollLatch tests scroll latch state tracking.
func TestEventStateScrollLatch(t *testing.T) {
	es := NewEventState()

	// Start vertical scroll latch
	es.StartScrollLatch(MouseB1, false)
	if !es.IsScrollLatched() {
		t.Error("scroll should be latched after StartScrollLatch")
	}
	latched, button, horizontal := es.ScrollState()
	if !latched {
		t.Error("scroll should be latched")
	}
	if button != MouseB1 {
		t.Errorf("scroll button should be B1; got %d", button)
	}
	if horizontal {
		t.Error("scroll should not be horizontal")
	}

	// End scroll latch
	es.EndScrollLatch()
	if es.IsScrollLatched() {
		t.Error("scroll should not be latched after EndScrollLatch")
	}
	latched, button, horizontal = es.ScrollState()
	if latched {
		t.Error("scroll should not be latched after EndScrollLatch")
	}
}

// TestEventStateScrollLatchHorizontal tests horizontal scroll latch.
func TestEventStateScrollLatchHorizontal(t *testing.T) {
	es := NewEventState()

	es.StartScrollLatch(MouseB2, true)
	latched, button, horizontal := es.ScrollState()
	if !latched {
		t.Error("scroll should be latched")
	}
	if button != MouseB2 {
		t.Errorf("scroll button should be B2; got %d", button)
	}
	if !horizontal {
		t.Error("scroll should be horizontal")
	}
}

// TestEventStateSetScrollState tests setting scroll state directly.
func TestEventStateSetScrollState(t *testing.T) {
	es := NewEventState()

	es.SetScrollState(true, MouseB3, true)
	latched, button, horizontal := es.ScrollState()
	if !latched {
		t.Error("scroll should be latched")
	}
	if button != MouseB3 {
		t.Errorf("scroll button should be B3; got %d", button)
	}
	if !horizontal {
		t.Error("scroll should be horizontal")
	}
}

// TestEventStateCurrentEvent tests current event tracking.
func TestEventStateCurrentEvent(t *testing.T) {
	es := NewEventState()

	es.SetCurrentEvent(EventMouseClick)
	if es.CurrentEvent() != EventMouseClick {
		t.Errorf("current event should be EventMouseClick; got %d", es.CurrentEvent())
	}
	if es.IsEventHandled() {
		t.Error("event should not be handled after SetCurrentEvent")
	}

	es.MarkEventHandled()
	if !es.IsEventHandled() {
		t.Error("event should be handled after MarkEventHandled")
	}

	// Setting new event clears handled flag
	es.SetCurrentEvent(EventKey)
	if es.IsEventHandled() {
		t.Error("event should not be handled after SetCurrentEvent")
	}
}

// TestEventStateReset tests the Reset method.
func TestEventStateReset(t *testing.T) {
	es := NewEventState()

	// Set various state
	es.SetLastMousePos(image.Pt(100, 200))
	es.SetLastMouseButtons(MouseB1 | MouseB2)
	es.SetMouseInTag(true)
	es.SetMouseInBody(true)
	es.SetMouseInScrollbar(true)
	es.SetSelectionActive(true)
	es.SetSelection(10, 50)
	es.SetSelectPoint(image.Pt(150, 250))
	es.RecordClick(42, 1000)
	es.StartChord(MouseB1 | MouseB2)
	es.StartScrollLatch(MouseB1, true)
	es.SetCurrentEvent(EventMouseClick)
	es.MarkEventHandled()

	// Reset
	es.Reset()

	// Verify all state is reset
	if es.LastMousePos() != (image.Point{}) {
		t.Error("after reset: mouse position should be zero")
	}
	if es.LastMouseButtons() != 0 {
		t.Error("after reset: mouse buttons should be 0")
	}
	if es.IsMouseInTag() {
		t.Error("after reset: mouse should not be in tag")
	}
	if es.IsMouseInBody() {
		t.Error("after reset: mouse should not be in body")
	}
	if es.IsMouseInScrollbar() {
		t.Error("after reset: mouse should not be in scrollbar")
	}
	if es.IsSelectionActive() {
		t.Error("after reset: selection should not be active")
	}
	start, end := es.Selection()
	if start != 0 || end != 0 {
		t.Error("after reset: selection should be 0,0")
	}
	if es.SelectPoint() != (image.Point{}) {
		t.Error("after reset: select point should be zero")
	}
	pos, msec, count := es.ClickState()
	if pos != 0 || msec != 0 || count != 0 {
		t.Error("after reset: click state should be zeroed")
	}
	if es.IsChordActive() {
		t.Error("after reset: chord should not be active")
	}
	if es.IsScrollLatched() {
		t.Error("after reset: scroll should not be latched")
	}
	if es.CurrentEvent() != EventNone {
		t.Error("after reset: current event should be EventNone")
	}
	if es.IsEventHandled() {
		t.Error("after reset: event should not be handled")
	}
}

// TestEventStateUpdateMouseRegion tests the UpdateMouseRegion method.
func TestEventStateUpdateMouseRegion(t *testing.T) {
	es := NewEventState()

	tagRect := image.Rect(0, 0, 800, 20)
	bodyRect := image.Rect(0, 21, 800, 600)
	scrollRect := image.Rect(0, 21, 20, 600)

	// Point in tag
	es.UpdateMouseRegion(image.Pt(100, 10), tagRect, bodyRect, scrollRect)
	if !es.IsMouseInTag() {
		t.Error("mouse should be in tag")
	}
	if es.IsMouseInBody() {
		t.Error("mouse should not be in body")
	}
	if es.IsMouseInScrollbar() {
		t.Error("mouse should not be in scrollbar")
	}

	// Point in body (not scrollbar)
	es.UpdateMouseRegion(image.Pt(100, 300), tagRect, bodyRect, scrollRect)
	if es.IsMouseInTag() {
		t.Error("mouse should not be in tag")
	}
	if !es.IsMouseInBody() {
		t.Error("mouse should be in body")
	}
	if es.IsMouseInScrollbar() {
		t.Error("mouse should not be in scrollbar")
	}

	// Point in scrollbar (which is also in body)
	es.UpdateMouseRegion(image.Pt(10, 300), tagRect, bodyRect, scrollRect)
	if es.IsMouseInTag() {
		t.Error("mouse should not be in tag")
	}
	if !es.IsMouseInBody() {
		t.Error("mouse should be in body (scrollbar overlaps body)")
	}
	if !es.IsMouseInScrollbar() {
		t.Error("mouse should be in scrollbar")
	}

	// Point outside all regions
	es.UpdateMouseRegion(image.Pt(-10, -10), tagRect, bodyRect, scrollRect)
	if es.IsMouseInTag() {
		t.Error("mouse should not be in tag")
	}
	if es.IsMouseInBody() {
		t.Error("mouse should not be in body")
	}
	if es.IsMouseInScrollbar() {
		t.Error("mouse should not be in scrollbar")
	}
}

// TestEventStateChordingScenarios tests common chord scenarios.
func TestEventStateChordingScenarios(t *testing.T) {
	es := NewEventState()

	// B1+B2 chord (cut)
	es.StartChord(MouseB1 | MouseB2)
	buttons, active := es.ChordState()
	if !active {
		t.Error("B1+B2 chord should be active")
	}
	if buttons&MouseB1 == 0 || buttons&MouseB2 == 0 {
		t.Error("B1+B2 chord should have both buttons")
	}
	es.EndChord()

	// B1+B3 chord (paste)
	es.StartChord(MouseB1 | MouseB3)
	buttons, active = es.ChordState()
	if !active {
		t.Error("B1+B3 chord should be active")
	}
	if buttons&MouseB1 == 0 || buttons&MouseB3 == 0 {
		t.Error("B1+B3 chord should have both buttons")
	}
	es.EndChord()

	// B1+B2+B3 chord (snarf)
	es.StartChord(MouseB1 | MouseB2 | MouseB3)
	buttons, active = es.ChordState()
	if !active {
		t.Error("B1+B2+B3 chord should be active")
	}
	if buttons != MouseB1|MouseB2|MouseB3 {
		t.Error("B1+B2+B3 chord should have all three buttons")
	}
	es.EndChord()
}

// TestEventStateMultipleRegions tests that mouse can be in multiple overlapping regions.
func TestEventStateMultipleRegions(t *testing.T) {
	es := NewEventState()

	// Scrollbar is typically a subset of body
	tagRect := image.Rect(0, 0, 800, 20)
	bodyRect := image.Rect(0, 21, 800, 600)
	scrollRect := image.Rect(0, 21, 20, 600) // scrollbar is part of body

	// Point in scrollbar (also in body)
	es.UpdateMouseRegion(image.Pt(10, 100), tagRect, bodyRect, scrollRect)
	if !es.IsMouseInBody() {
		t.Error("scrollbar point should also be in body")
	}
	if !es.IsMouseInScrollbar() {
		t.Error("scrollbar point should be in scrollbar")
	}
}
