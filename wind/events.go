// Package wind provides the Window type and related components for edwood.
// This file contains event handling types and methods for windows.
package wind

import (
	"image"
)

// MouseButton represents a mouse button state.
type MouseButton int

const (
	// MouseB1 is button 1 (left click).
	MouseB1 MouseButton = 1 << iota
	// MouseB2 is button 2 (middle click).
	MouseB2
	// MouseB3 is button 3 (right click).
	MouseB3
	// MouseB4 is button 4 (scroll up).
	MouseB4
	// MouseB5 is button 5 (scroll down).
	MouseB5
)

// EventType represents the type of event being processed.
type EventType int

const (
	// EventNone indicates no event.
	EventNone EventType = iota
	// EventMouseClick indicates a mouse click event.
	EventMouseClick
	// EventMouseDrag indicates a mouse drag event.
	EventMouseDrag
	// EventMouseScroll indicates a scroll wheel event.
	EventMouseScroll
	// EventKey indicates a keyboard event.
	EventKey
)

// EventHandler defines the interface for handling window events.
// This interface abstracts the event processing used by windows.
type EventHandler interface {
	// HandleMouse processes a mouse event. Returns true if the event was handled.
	HandleMouse(pos image.Point, buttons MouseButton) bool
	// HandleKey processes a keyboard event. Returns true if the event was handled.
	HandleKey(key rune) bool
}

// EventState tracks the state needed for window event handling.
// This encapsulates event-related state that was previously part of Window.
type EventState struct {
	// Mouse state
	lastMousePos     image.Point // last known mouse position
	lastMouseButtons MouseButton // last known button state
	mouseInTag       bool        // true if mouse is in tag area
	mouseInBody      bool        // true if mouse is in body area
	mouseInScrollbar bool        // true if mouse is in scrollbar area

	// Selection state
	selectionActive bool        // true during active selection
	selectionStart  int         // rune offset where selection started
	selectionEnd    int         // rune offset where selection ends
	selectPoint     image.Point // point where selection started

	// Double-click state
	clickPos     int    // rune position of last null-click
	clickMsec    uint32 // timestamp of last null-click
	clickCount   int    // number of clicks in sequence
	lastClickPos int    // position of previous click for double-click detection

	// Chord state (for B1+B2, B1+B3 chording)
	chordButtons MouseButton // buttons held during chord
	chordActive  bool        // true during chord operation

	// Scroll state
	scrollLatch      bool        // true when scroll is latched
	scrollLatchBtn   MouseButton // which button is latched
	scrollHorizontal bool        // true for horizontal scroll

	// Event processing
	currentEvent EventType // type of event being processed
	eventHandled bool      // true if current event was handled
}

// NewEventState creates a new EventState with default values.
func NewEventState() *EventState {
	return &EventState{}
}

// LastMousePos returns the last known mouse position.
func (es *EventState) LastMousePos() image.Point {
	return es.lastMousePos
}

// SetLastMousePos sets the last mouse position.
func (es *EventState) SetLastMousePos(pos image.Point) {
	es.lastMousePos = pos
}

// LastMouseButtons returns the last known button state.
func (es *EventState) LastMouseButtons() MouseButton {
	return es.lastMouseButtons
}

// SetLastMouseButtons sets the last button state.
func (es *EventState) SetLastMouseButtons(buttons MouseButton) {
	es.lastMouseButtons = buttons
}

// IsMouseInTag returns true if the mouse is in the tag area.
func (es *EventState) IsMouseInTag() bool {
	return es.mouseInTag
}

// SetMouseInTag sets whether the mouse is in the tag area.
func (es *EventState) SetMouseInTag(inTag bool) {
	es.mouseInTag = inTag
}

// IsMouseInBody returns true if the mouse is in the body area.
func (es *EventState) IsMouseInBody() bool {
	return es.mouseInBody
}

// SetMouseInBody sets whether the mouse is in the body area.
func (es *EventState) SetMouseInBody(inBody bool) {
	es.mouseInBody = inBody
}

// IsMouseInScrollbar returns true if the mouse is in the scrollbar area.
func (es *EventState) IsMouseInScrollbar() bool {
	return es.mouseInScrollbar
}

// SetMouseInScrollbar sets whether the mouse is in the scrollbar area.
func (es *EventState) SetMouseInScrollbar(inScrollbar bool) {
	es.mouseInScrollbar = inScrollbar
}

// IsSelectionActive returns true during active selection.
func (es *EventState) IsSelectionActive() bool {
	return es.selectionActive
}

// SetSelectionActive sets whether selection is active.
func (es *EventState) SetSelectionActive(active bool) {
	es.selectionActive = active
}

// Selection returns the current selection range (start, end).
func (es *EventState) Selection() (int, int) {
	return es.selectionStart, es.selectionEnd
}

// SetSelection sets the selection range.
func (es *EventState) SetSelection(start, end int) {
	es.selectionStart = start
	es.selectionEnd = end
}

// SelectPoint returns the point where selection started.
func (es *EventState) SelectPoint() image.Point {
	return es.selectPoint
}

// SetSelectPoint sets the point where selection started.
func (es *EventState) SetSelectPoint(pt image.Point) {
	es.selectPoint = pt
}

// ClickState returns the double-click tracking state (pos, msec, count).
func (es *EventState) ClickState() (int, uint32, int) {
	return es.clickPos, es.clickMsec, es.clickCount
}

// SetClickState sets the double-click tracking state.
func (es *EventState) SetClickState(pos int, msec uint32, count int) {
	es.clickPos = pos
	es.clickMsec = msec
	es.clickCount = count
}

// LastClickPos returns the position of the previous click.
func (es *EventState) LastClickPos() int {
	return es.lastClickPos
}

// SetLastClickPos sets the position of the previous click.
func (es *EventState) SetLastClickPos(pos int) {
	es.lastClickPos = pos
}

// IsDoubleClick checks if this click constitutes a double-click.
// A double-click occurs when:
// - The current click is at the same position as the last click
// - The time since last click is less than the threshold (500ms typical)
func (es *EventState) IsDoubleClick(pos int, msec uint32, threshold uint32) bool {
	if es.clickCount == 0 {
		return false
	}
	return pos == es.clickPos && msec-es.clickMsec < threshold
}

// RecordClick records a click for double-click detection.
func (es *EventState) RecordClick(pos int, msec uint32) {
	if es.IsDoubleClick(pos, msec, 500) {
		es.clickCount++
	} else {
		es.clickCount = 1
	}
	es.lastClickPos = es.clickPos
	es.clickPos = pos
	es.clickMsec = msec
}

// ClearClickState clears the double-click state.
func (es *EventState) ClearClickState() {
	es.clickPos = 0
	es.clickMsec = 0
	es.clickCount = 0
	es.lastClickPos = 0
}

// ChordState returns the chord state (buttons, active).
func (es *EventState) ChordState() (MouseButton, bool) {
	return es.chordButtons, es.chordActive
}

// SetChordState sets the chord state.
func (es *EventState) SetChordState(buttons MouseButton, active bool) {
	es.chordButtons = buttons
	es.chordActive = active
}

// IsChordActive returns true during chord operation.
func (es *EventState) IsChordActive() bool {
	return es.chordActive
}

// StartChord starts a chord operation with the given buttons.
func (es *EventState) StartChord(buttons MouseButton) {
	es.chordButtons = buttons
	es.chordActive = true
}

// EndChord ends the current chord operation.
func (es *EventState) EndChord() {
	es.chordButtons = 0
	es.chordActive = false
}

// ScrollState returns the scroll latch state (latched, button, horizontal).
func (es *EventState) ScrollState() (bool, MouseButton, bool) {
	return es.scrollLatch, es.scrollLatchBtn, es.scrollHorizontal
}

// SetScrollState sets the scroll latch state.
func (es *EventState) SetScrollState(latched bool, button MouseButton, horizontal bool) {
	es.scrollLatch = latched
	es.scrollLatchBtn = button
	es.scrollHorizontal = horizontal
}

// IsScrollLatched returns true when scroll is latched.
func (es *EventState) IsScrollLatched() bool {
	return es.scrollLatch
}

// StartScrollLatch starts scroll latching.
func (es *EventState) StartScrollLatch(button MouseButton, horizontal bool) {
	es.scrollLatch = true
	es.scrollLatchBtn = button
	es.scrollHorizontal = horizontal
}

// EndScrollLatch ends scroll latching.
func (es *EventState) EndScrollLatch() {
	es.scrollLatch = false
	es.scrollLatchBtn = 0
	es.scrollHorizontal = false
}

// CurrentEvent returns the type of event being processed.
func (es *EventState) CurrentEvent() EventType {
	return es.currentEvent
}

// SetCurrentEvent sets the current event type.
func (es *EventState) SetCurrentEvent(event EventType) {
	es.currentEvent = event
	es.eventHandled = false
}

// IsEventHandled returns true if the current event was handled.
func (es *EventState) IsEventHandled() bool {
	return es.eventHandled
}

// MarkEventHandled marks the current event as handled.
func (es *EventState) MarkEventHandled() {
	es.eventHandled = true
}

// Reset resets all event state to default values.
func (es *EventState) Reset() {
	es.lastMousePos = image.Point{}
	es.lastMouseButtons = 0
	es.mouseInTag = false
	es.mouseInBody = false
	es.mouseInScrollbar = false
	es.selectionActive = false
	es.selectionStart = 0
	es.selectionEnd = 0
	es.selectPoint = image.Point{}
	es.clickPos = 0
	es.clickMsec = 0
	es.clickCount = 0
	es.lastClickPos = 0
	es.chordButtons = 0
	es.chordActive = false
	es.scrollLatch = false
	es.scrollLatchBtn = 0
	es.scrollHorizontal = false
	es.currentEvent = EventNone
	es.eventHandled = false
}

// UpdateMouseRegion updates the mouse region flags based on the current position
// and the given tag and body rectangles.
func (es *EventState) UpdateMouseRegion(pos image.Point, tagRect, bodyRect, scrollRect image.Rectangle) {
	es.mouseInTag = pos.In(tagRect)
	es.mouseInBody = pos.In(bodyRect)
	es.mouseInScrollbar = pos.In(scrollRect)
}
