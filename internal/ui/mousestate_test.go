// Package ui provides user interface utilities for edwood.
package ui

import (
	"image"
	"testing"
)

// MockDisplay implements a minimal display interface for testing.
type MockDisplay struct {
	lastMoveTo image.Point
	moveCount  int
}

func (d *MockDisplay) MoveTo(pt image.Point) error {
	d.lastMoveTo = pt
	d.moveCount++
	return nil
}

// MockWindow provides a minimal window interface for testing.
// In the actual implementation, this will be replaced by a proper interface.
type MockWindow struct {
	id      int
	display *MockDisplay
}

func (w *MockWindow) Display() MouseMover {
	return w.display
}

// TestMouseStateNew tests that a new MouseState is properly initialized.
func TestMouseStateNew(t *testing.T) {
	ms := NewMouseState()
	if ms == nil {
		t.Fatal("NewMouseState returned nil")
	}

	// A new MouseState should have no saved window
	if ms.HasSaved() {
		t.Error("new MouseState should not have saved state")
	}
}

// TestMouseStateSave tests the Save method.
func TestMouseStateSave(t *testing.T) {
	ms := NewMouseState()
	display := &MockDisplay{}
	win := &MockWindow{id: 1, display: display}
	pt := image.Point{X: 100, Y: 200}

	ms.Save(win, pt)

	if !ms.HasSaved() {
		t.Error("MouseState should have saved state after Save")
	}
}

// TestMouseStateClear tests the Clear method.
func TestMouseStateClear(t *testing.T) {
	ms := NewMouseState()
	display := &MockDisplay{}
	win := &MockWindow{id: 1, display: display}
	pt := image.Point{X: 100, Y: 200}

	ms.Save(win, pt)
	ms.Clear()

	if ms.HasSaved() {
		t.Error("MouseState should not have saved state after Clear")
	}
}

// TestMouseStateRestoreSameWindow tests Restore with the same window.
func TestMouseStateRestoreSameWindow(t *testing.T) {
	ms := NewMouseState()
	display := &MockDisplay{}
	win := &MockWindow{id: 1, display: display}
	pt := image.Point{X: 100, Y: 200}

	ms.Save(win, pt)

	restored := ms.Restore(win)

	if !restored {
		t.Error("Restore should return true for same window")
	}
	if display.lastMoveTo != pt {
		t.Errorf("MoveTo called with %v; want %v", display.lastMoveTo, pt)
	}
	if display.moveCount != 1 {
		t.Errorf("MoveTo called %d times; want 1", display.moveCount)
	}

	// After restore, saved state should be cleared
	if ms.HasSaved() {
		t.Error("MouseState should not have saved state after Restore")
	}
}

// TestMouseStateRestoreDifferentWindow tests Restore with a different window.
func TestMouseStateRestoreDifferentWindow(t *testing.T) {
	ms := NewMouseState()
	display1 := &MockDisplay{}
	display2 := &MockDisplay{}
	win1 := &MockWindow{id: 1, display: display1}
	win2 := &MockWindow{id: 2, display: display2}
	pt := image.Point{X: 100, Y: 200}

	ms.Save(win1, pt)

	restored := ms.Restore(win2)

	if restored {
		t.Error("Restore should return false for different window")
	}
	if display1.moveCount != 0 {
		t.Errorf("win1 display MoveTo called %d times; want 0", display1.moveCount)
	}
	if display2.moveCount != 0 {
		t.Errorf("win2 display MoveTo called %d times; want 0", display2.moveCount)
	}

	// After restore (even if not restored), saved state should be cleared
	if ms.HasSaved() {
		t.Error("MouseState should not have saved state after Restore")
	}
}

// TestMouseStateRestoreNilWindow tests Restore when called with nil.
func TestMouseStateRestoreNilWindow(t *testing.T) {
	ms := NewMouseState()
	display := &MockDisplay{}
	win := &MockWindow{id: 1, display: display}
	pt := image.Point{X: 100, Y: 200}

	ms.Save(win, pt)

	restored := ms.Restore(nil)

	if restored {
		t.Error("Restore should return false when called with nil")
	}
	if display.moveCount != 0 {
		t.Errorf("MoveTo should not be called when restoring nil; called %d times", display.moveCount)
	}

	// After restore (even if not restored), saved state should be cleared
	if ms.HasSaved() {
		t.Error("MouseState should not have saved state after Restore")
	}
}

// TestMouseStateRestoreNoSavedState tests Restore when nothing was saved.
func TestMouseStateRestoreNoSavedState(t *testing.T) {
	ms := NewMouseState()
	display := &MockDisplay{}
	win := &MockWindow{id: 1, display: display}

	restored := ms.Restore(win)

	if restored {
		t.Error("Restore should return false when nothing was saved")
	}
	if display.moveCount != 0 {
		t.Errorf("MoveTo should not be called when nothing was saved; called %d times", display.moveCount)
	}
}

// TestMouseStateSaveOverwrite tests that Save overwrites previous state.
func TestMouseStateSaveOverwrite(t *testing.T) {
	ms := NewMouseState()
	display1 := &MockDisplay{}
	display2 := &MockDisplay{}
	win1 := &MockWindow{id: 1, display: display1}
	win2 := &MockWindow{id: 2, display: display2}
	pt1 := image.Point{X: 100, Y: 200}
	pt2 := image.Point{X: 300, Y: 400}

	ms.Save(win1, pt1)
	ms.Save(win2, pt2)

	// Restore with win1 should fail (win2 is now saved)
	restored := ms.Restore(win1)
	if restored {
		t.Error("Restore win1 should return false after saving win2")
	}

	// Create new state and save win2 again for proper restore test
	ms2 := NewMouseState()
	ms2.Save(win2, pt2)

	restored = ms2.Restore(win2)
	if !restored {
		t.Error("Restore win2 should return true")
	}
	if display2.lastMoveTo != pt2 {
		t.Errorf("MoveTo called with %v; want %v", display2.lastMoveTo, pt2)
	}
}

// TestMouseStateClearBeforeSave tests Clear when nothing was saved.
func TestMouseStateClearBeforeSave(t *testing.T) {
	ms := NewMouseState()

	// Should not panic
	ms.Clear()

	if ms.HasSaved() {
		t.Error("Clear on empty MouseState should leave it empty")
	}
}
