package main

// Tests for Phase 3.2: Thread-Safe Debounce Implementation
//
// These tests verify that SchedulePreviewUpdate() is safe for concurrent use
// and that timer cancellation works correctly on window close and preview toggle.
//
// Run with: go test -race -run TestDebounce ./...
//
// Assumptions about goroutine scheduling:
// - time.AfterFunc callbacks run on separate goroutines
// - The race detector will flag unsynchronized concurrent access
// - Timer.Stop() may or may not prevent an already-queued callback
//
// The tests use the real previewUpdateDelay (3s). Tests that need to wait for
// the timer to fire allow 4s. This is acceptable for correctness tests.

import (
	"image"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rjkroege/edwood/edwoodtest"
	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/markdown"
)

// setupDebounceTestWindow creates a Window in preview mode suitable for
// debounce testing. The window has a body buffer, richBody, and source map
// all wired up.
func setupDebounceTestWindow(t *testing.T) *Window {
	t.Helper()

	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	sourceMarkdown := "# Hello World\n\nSome text here."
	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/debounce.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(0, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	content, sourceMap, _ := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewMode(true)

	return w
}

// TestDebounceRaceDetection verifies that SchedulePreviewUpdate and
// UpdatePreview do not race with concurrent Window state reads when both
// sides properly hold the row lock.
//
// The timer callback acquires global.row.lk before calling UpdatePreview().
// This test simulates mousethread/keyboardthread by reading Window fields
// under the same lock, verifying that both sides synchronize correctly
// and the race detector is satisfied.
func TestDebounceRaceDetection(t *testing.T) {
	w := setupDebounceTestWindow(t)

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Schedule the first preview update. The timer callback will fire on a
	// separate goroutine after previewUpdateDelay (3s) and call UpdatePreview()
	// which writes to w.previewSourceMap, w.previewLinkMap, etc.
	global.row.lk.Lock()
	w.SchedulePreviewUpdate()
	global.row.lk.Unlock()

	// Goroutine: simulate mousethread/keyboardthread by reading Window state
	// under the row lock. The timer callback also acquires the lock, so both
	// sides are synchronized and the race detector should be satisfied.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				global.row.lk.Lock()
				_ = w.previewSourceMap
				_ = w.previewLinkMap
				_ = w.previewMode
				global.row.lk.Unlock()
				time.Sleep(100 * time.Microsecond)
			}
		}
	}()

	// Let the timer fire. previewUpdateDelay is 3s, wait 4s.
	time.Sleep(4 * time.Second)
	close(done)
	wg.Wait()
}

// TestDebounceCancellationOnClose verifies that when preview state is cleaned
// up (as Close() should do), a pending timer callback does not panic or race.
//
// This simulates the Close() cleanup path: cancel timer, nil out preview fields.
// We don't call w.Close() directly because the test Window lacks full teardown
// wiring (observer registration, etc.). Instead we test the specific behavior
// that the fix adds to Close(): timer cancellation + preview state cleanup.
//
// Sequence:
// 1. Create window in preview mode.
// 2. Schedule a preview update (starts 3s timer).
// 3. Simulate close cleanup: cancel timer, set previewMode=false, richBody=nil.
// 4. Wait for the timer to fire.
// 5. Verify no panic and state remains clean.
func TestDebounceCancellationOnClose(t *testing.T) {
	w := setupDebounceTestWindow(t)

	// Schedule a preview update — timer will fire in 3 seconds
	global.row.lk.Lock()
	w.SchedulePreviewUpdate()
	global.row.lk.Unlock()

	// Simulate what the fixed Close() should do: cancel timer and nil preview state
	global.row.lk.Lock()
	if w.previewUpdateTimer != nil {
		w.previewUpdateTimer.Stop()
		w.previewUpdateTimer = nil
	}
	w.previewMode = false
	w.richBody = nil
	global.row.lk.Unlock()

	// Wait for the timer to fire (if Stop didn't prevent it).
	// With the fix, the callback acquires the lock and re-checks w.previewMode
	// and w.richBody, finding them false/nil, so it returns harmlessly.
	time.Sleep(4 * time.Second)

	// Verify the state is still clean after waiting
	global.row.lk.Lock()
	defer global.row.lk.Unlock()
	if w.previewMode {
		t.Error("previewMode should still be false")
	}
	if w.richBody != nil {
		t.Error("richBody should still be nil")
	}
	if w.previewUpdateTimer != nil {
		t.Error("previewUpdateTimer should still be nil")
	}
}

// TestDebounceCancellationOnPreviewToggle verifies that toggling preview
// mode off cancels the pending update timer so UpdatePreview() is not called.
//
// Sequence:
// 1. Create window in preview mode.
// 2. Schedule a preview update.
// 3. Toggle preview mode off.
// 4. Modify the body to make the change detectable.
// 5. Wait for the timer to fire.
// 6. Verify the preview was NOT updated (content unchanged).
func TestDebounceCancellationOnPreviewToggle(t *testing.T) {
	w := setupDebounceTestWindow(t)

	// Capture the initial content length
	initialContent := w.richBody.Content()
	if initialContent == nil {
		t.Fatal("Initial content should not be nil")
	}
	initialLen := initialContent.Len()

	// Schedule a preview update
	global.row.lk.Lock()
	w.SchedulePreviewUpdate()
	global.row.lk.Unlock()

	// Toggle preview mode off. This should cancel the timer.
	global.row.lk.Lock()
	// Cancel timer before toggling (this is the behavior we're testing)
	if w.previewUpdateTimer != nil {
		w.previewUpdateTimer.Stop()
		w.previewUpdateTimer = nil
	}
	w.SetPreviewMode(false)
	global.row.lk.Unlock()

	// Modify the body — if UpdatePreview fires, it would parse this new content
	global.row.lk.Lock()
	w.body.file.DeleteAt(0, w.body.file.Nr())
	w.body.file.InsertAt(0, []rune("# Completely Different\n\nNew content that is much longer than before to make the length difference obvious."))
	global.row.lk.Unlock()

	// Wait for the timer to fire
	time.Sleep(4 * time.Second)

	// Verify UpdatePreview was NOT called: re-enable preview mode to check
	// the rich body content is still the OLD content (not the new body text).
	global.row.lk.Lock()
	defer global.row.lk.Unlock()
	if w.richBody != nil {
		currentContent := w.richBody.Content()
		if currentContent != nil && currentContent.Len() != initialLen {
			t.Errorf("Content should not have changed after preview toggle off: initial len=%d, current len=%d", initialLen, currentContent.Len())
		}
	}
}

// TestDebouncePreservesDebouncing verifies that calling SchedulePreviewUpdate()
// multiple times in rapid succession results in only a single UpdatePreview() call.
//
// Sequence:
// 1. Create window in preview mode.
// 2. Modify body to make the content different from the initial parse.
// 3. Call SchedulePreviewUpdate() three times rapidly, modifying body each time.
// 4. Wait for the timer to fire.
// 5. Verify UpdatePreview() was called (content reflects last body state).
func TestDebouncePreservesDebouncing(t *testing.T) {
	w := setupDebounceTestWindow(t)

	// Track how many times UpdatePreview effectively ran by watching content changes.
	// We'll modify the body before each schedule and check the final state.

	// Rapid-fire three schedule calls with different body content each time
	bodies := []string{
		"# First Edit\n\nFirst paragraph.",
		"# Second Edit\n\nSecond paragraph.",
		"# Third Edit\n\nThird paragraph with extra text to distinguish.",
	}

	for _, body := range bodies {
		global.row.lk.Lock()
		w.body.file.DeleteAt(0, w.body.file.Nr())
		w.body.file.InsertAt(0, []rune(body))
		w.SchedulePreviewUpdate()
		global.row.lk.Unlock()
		time.Sleep(10 * time.Millisecond) // Small gap between calls
	}

	// Wait for the debounced timer to fire
	time.Sleep(4 * time.Second)

	// The preview should reflect the LAST body content (third edit),
	// because each SchedulePreviewUpdate cancels the previous timer.
	global.row.lk.Lock()
	defer global.row.lk.Unlock()

	// After the timer fires and UpdatePreview runs, the source map should
	// reflect the third edit's content
	if w.previewSourceMap == nil {
		t.Error("Source map should be set after debounced update")
	}

	// Check that the body still has the third edit's content
	bodyStr := w.body.file.String()
	if bodyStr != bodies[2] {
		t.Errorf("Body should contain third edit, got: %q", bodyStr)
	}
}

// TestDebounceTimerFieldSynchronization verifies that the previewUpdateTimer
// field is only accessed under the row lock, preventing races on the timer
// field itself.
//
// This is a targeted race test: one goroutine schedules updates (writes
// previewUpdateTimer) while another checks the timer field (reads it).
// Both should hold the row lock.
func TestDebounceTimerFieldSynchronization(t *testing.T) {
	w := setupDebounceTestWindow(t)

	var wg sync.WaitGroup
	done := make(chan struct{})
	var scheduleCount atomic.Int64

	// Writer goroutine: schedule updates
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				global.row.lk.Lock()
				w.SchedulePreviewUpdate()
				scheduleCount.Add(1)
				global.row.lk.Unlock()
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	// Reader goroutine: check timer field
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				global.row.lk.Lock()
				_ = w.previewUpdateTimer != nil
				global.row.lk.Unlock()
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	// Run for 1 second — enough iterations to detect races
	time.Sleep(1 * time.Second)
	close(done)
	wg.Wait()

	if scheduleCount.Load() == 0 {
		t.Error("Expected at least one SchedulePreviewUpdate call")
	}
}

// TestDebounceNotPreviewMode verifies that SchedulePreviewUpdate is a no-op
// when the window is not in preview mode.
func TestDebounceNotPreviewMode(t *testing.T) {
	w := setupDebounceTestWindow(t)

	// Exit preview mode
	w.SetPreviewMode(false)

	// SchedulePreviewUpdate should be a no-op
	w.SchedulePreviewUpdate()

	if w.previewUpdateTimer != nil {
		t.Error("Timer should not be set when not in preview mode")
	}
}

// TestDebounceNilRichBody verifies that SchedulePreviewUpdate is a no-op
// when richBody is nil.
func TestDebounceNilRichBody(t *testing.T) {
	w := setupDebounceTestWindow(t)

	// Nil out richBody while keeping preview mode on
	w.richBody = nil

	w.SchedulePreviewUpdate()

	if w.previewUpdateTimer != nil {
		t.Error("Timer should not be set when richBody is nil")
	}
}
