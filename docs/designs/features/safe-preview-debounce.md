# Safe Preview Debounce Design

## Problem

`SchedulePreviewUpdate()` in `wind.go:1526-1542` uses `time.AfterFunc` to debounce preview updates. The timer callback fires on a **separate goroutine** created by the Go runtime, but `UpdatePreview()` accesses Window state (`previewMode`, `richBody`, `body.file`, `previewSourceMap`, `previewLinkMap`, `display`) without synchronization. This creates data races with the mouse thread and keyboard thread, which access the same Window fields under `g.row.lk`.

The comment at line 1538 acknowledges the problem:
```go
// UpdatePreview must be called from the main goroutine
// Use the global display channel to schedule the update
w.UpdatePreview()  // BUG: called directly from timer goroutine
```

### What races exist

1. **Timer goroutine vs mousethread**: `mousethread` holds `g.row.lk` while processing mouse events that read/write `w.richBody`, `w.previewSourceMap`, `w.previewLinkMap`, selection state. The timer goroutine calls `UpdatePreview()` which writes all of these without holding the lock.

2. **Timer goroutine vs keyboardthread**: `keyboardthread` calls `g.row.Type()` which can trigger `SchedulePreviewUpdate()` (via `text.go:452`). The timer goroutine may fire while keyboard handling is in progress.

3. **Timer goroutine vs display**: `UpdatePreview()` calls `w.richBody.Render()` and `w.display.Flush()`. While `Display.Flush()` has internal synchronization, the rich text layout and rendering is not thread-safe.

### Additional bug: no timer cancellation on window close

`Window.Close()` (wind.go:411) and `previewcmd()` exit path (exec.go:1214) do not cancel `previewUpdateTimer`. If the window is closed or preview mode is toggled off while a timer is pending, the callback fires on a dead or non-preview window, accessing stale pointers.

## Architecture Context

Edwood does **not** have a single main goroutine event loop. It runs 5+ worker goroutines (mousethread, keyboardthread, waitthread, newwindowthread, xfidallocthread) that process events in parallel. Shared state is protected by `g.row.lk`, a mutex that each thread acquires before touching Row/Column/Window state.

Key observation: `mousethread` (acme.go:362-396) is the goroutine that handles display flushing and resize. It acquires `g.row.lk`, processes the event, unlocks, then calls `display.Flush()`. This is the closest thing to a "main UI goroutine."

The `keyboardthread` (acme.go:535-585) has an existing debounce pattern using `time.NewTimer` + select that is safe because the timer fires into the same select loop, so the tag commit runs on the keyboard goroutine itself.

## Design: Lock-Based Synchronization

### Approach

The simplest correct fix is to have the timer callback acquire `g.row.lk` before calling `UpdatePreview()`, matching the pattern used by every other goroutine that touches Window state. This is simpler and more consistent with the existing codebase than alternatives (channels, work queues).

### Why not channel-based dispatch?

The `mousethread` select loop (acme.go:366-394) could accept a work channel, but:
- Adding a channel to `mousethread`'s select requires modifying a critical event loop that handles resize, mouse, plumb messages, and warnings.
- The existing `g.cwarn` channel in mousethread already demonstrates the pattern but is a simple signal, not a work queue.
- The lock-based approach matches the existing codebase pattern (waitthread, xfidctl, etc. all acquire `g.row.lk`).
- A channel-based approach doesn't solve the close-cancellation problem; the lock approach does (by checking `w.previewMode` after acquiring the lock).

### Implementation

#### 1. Store reference to row lock on Window

The timer callback needs access to `g.row.lk` to synchronize. Currently, `Window` doesn't have a reference to the row lock. We need to pass it.

Options:
- **(A)** Store `*sync.Mutex` (pointer to `g.row.lk`) on Window. Set during window initialization.
- **(B)** Store a reference to `*globals` on Window. Window already has `display` which comes from globals.
- **(C)** Use a package-level `global` variable. The codebase already uses a package-level `global` variable of type `*globals` extensively (e.g., `global.seq`, `global.activewin`, etc.).

**Choice: (C)** — use the existing `global` package variable. The timer callback will do `global.row.lk.Lock()` / `global.row.lk.Unlock()`. This is the simplest change and consistent with how other code in the package accesses the row lock.

#### 2. Modified `SchedulePreviewUpdate()`

```go
func (w *Window) SchedulePreviewUpdate() {
    if !w.previewMode || w.richBody == nil {
        return
    }

    // Cancel any pending update
    if w.previewUpdateTimer != nil {
        w.previewUpdateTimer.Stop()
    }

    // Schedule a new update after the delay.
    // The callback runs on a timer goroutine; acquire the row lock
    // to synchronize with mousethread, keyboardthread, and others.
    w.previewUpdateTimer = time.AfterFunc(previewUpdateDelay, func() {
        global.row.lk.Lock()
        defer global.row.lk.Unlock()

        // Re-check: window may have exited preview mode or been closed
        // while we waited for the lock.
        if !w.previewMode || w.richBody == nil {
            return
        }

        w.UpdatePreview()
    })
}
```

#### 3. Timer cancellation on preview exit

Add timer cancellation to `previewcmd()` (exec.go) when toggling preview off:

```go
// In previewcmd(), before SetPreviewMode(false):
if w.previewUpdateTimer != nil {
    w.previewUpdateTimer.Stop()
    w.previewUpdateTimer = nil
}
```

#### 4. Timer cancellation on window close

Add timer cancellation to `Window.Close()` (wind.go):

```go
func (w *Window) Close() {
    if w.ref.Dec() == 0 {
        // Cancel pending preview update timer to prevent callback
        // firing on a closed window.
        if w.previewUpdateTimer != nil {
            w.previewUpdateTimer.Stop()
            w.previewUpdateTimer = nil
        }
        xfidlog(w, "del")
        // ... rest of Close()
    }
}
```

### Display.Flush() placement

Currently `UpdatePreview()` calls `w.display.Flush()` at the end. In the lock-based approach, `Flush()` is called while the lock is held. This matches the pattern in `waitthread` (acme.go:608, 651, 672) where `display.Flush()` is called inside the locked section. The `mousethread` calls `display.Flush()` *outside* the lock (acme.go:370), but this is specifically because mousethread's flush is a periodic batch flush, not tied to a specific operation.

For the timer callback, calling `Flush()` inside the lock is correct because:
- It ensures the display update from `UpdatePreview()` is atomically visible.
- `Display.Flush()` is fast (just sends buffered draw ops to the display server).
- Other writers (mousethread) will see the completed state after acquiring the lock.

### Race condition: `SchedulePreviewUpdate()` itself

`SchedulePreviewUpdate()` is called from `text.go:452` and `text.go:604`, which are called during `Insert`/`Delete` operations. These callers already hold `g.row.lk` (via keyboardthread or mousethread). The field `w.previewUpdateTimer` is only accessed from:
1. `SchedulePreviewUpdate()` — called with lock held.
2. The timer callback — will acquire the lock before accessing Window state.
3. `Close()` / `previewcmd()` — called with lock held.

Since `time.Timer.Stop()` is goroutine-safe, and we only read/write `w.previewUpdateTimer` with the lock held, there is no race on the timer field itself.

### Edge case: rapid close after schedule

Sequence:
1. Timer scheduled (3s delay).
2. User closes window 1s later.
3. `Close()` calls `timer.Stop()` — timer may or may not have already fired.

If `Stop()` returns false, the callback is already running or queued. It will block on `global.row.lk.Lock()`. When it acquires the lock, the re-check `if !w.previewMode || w.richBody == nil` guards against accessing a closed window. However, `Close()` doesn't nil out `richBody` or `previewMode`. We should nil out `richBody` in `Close()` to make the guard effective:

```go
func (w *Window) Close() {
    if w.ref.Dec() == 0 {
        if w.previewUpdateTimer != nil {
            w.previewUpdateTimer.Stop()
            w.previewUpdateTimer = nil
        }
        w.previewMode = false
        w.richBody = nil
        // ... rest of Close()
    }
}
```

### Edge case: timer fires exactly when lock is held for different operation

The timer goroutine blocks on `global.row.lk.Lock()` until the current holder releases it. This is normal mutex behavior. The timer fires slightly later than the 3-second delay, which is acceptable for a debounce timer.

## Testing Strategy

### Test 1: Race detection

Run `go test -race ./...` with a test that:
1. Creates a Window in preview mode.
2. Calls `SchedulePreviewUpdate()` from multiple goroutines concurrently.
3. Concurrently reads Window state (simulating mousethread).
4. Verifies no race detector warnings.

### Test 2: Timer cancellation on close

1. Create a Window in preview mode.
2. Call `SchedulePreviewUpdate()`.
3. Immediately call `Close()`.
4. Wait > 3 seconds.
5. Verify no panic or use-after-close.

### Test 3: Timer cancellation on preview toggle

1. Create a Window in preview mode.
2. Call `SchedulePreviewUpdate()`.
3. Toggle preview mode off.
4. Wait > 3 seconds.
5. Verify `UpdatePreview()` was NOT called (content unchanged).

### Test 4: Debounce behavior preserved

1. Create a Window in preview mode.
2. Call `SchedulePreviewUpdate()` three times rapidly.
3. Wait > 3 seconds.
4. Verify `UpdatePreview()` was called exactly once.

## Files Modified

| File | Change |
|------|--------|
| `wind.go` | `SchedulePreviewUpdate()`: acquire `global.row.lk` in timer callback, re-check state. `Close()`: cancel timer, nil out preview fields. |
| `exec.go` | `previewcmd()` exit path: cancel timer before toggling preview off. |

## Not Changed

- `UpdatePreview()` itself: no changes needed, it's now always called with the lock held.
- `mousethread` / `keyboardthread`: no changes to event loops.
- No new channels or goroutines added.
- No changes to `display` interface.
