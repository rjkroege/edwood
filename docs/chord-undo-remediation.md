# Chord Undo Remediation Plan

## Problem Summary

The preview mode chord handlers in `HandlePreviewMouse` (`wind.go:989-1009`) bypass Edwood's high-level editing primitives. They call `w.body.file.DeleteAt()` and `w.body.file.InsertAt()` directly instead of going through the `Text.Delete()`/`Text.Insert()` methods and the standard `cut()`/`paste()` functions in `exec.go`.

This breaks:

1. **Undo/Redo**: Changes have `seq=0`, causing `FlattenHistory()` to destroy undo records
2. **9P file interface**: `Text`-level observer notifications are skipped, so external programs reading via the file system interface won't see the edits
3. **System clipboard sync**: `acmeputsnarf()` is never called
4. **Commit lifecycle**: `w.Commit(&w.body)` is never called
5. **Selection state**: `t.SetSelect()` / `t.ScrDraw()` are not called

## What the Standard Path Does

The normal keyboard and B2-command paths for Cut/Paste/Snarf (`text.go:941-962`, `exec.go:355-480`) follow this sequence:

```
1. TypeCommit()              — flush any pending typed text
2. global.seq++              — advance undo sequence counter
3. file.Mark(global.seq)     — create undo checkpoint
4. cut() / paste()           — high-level functions that call:
   a. Text.Delete(q0, q1)   — which calls file.DeleteAt + notifies observers
   b. Text.Insert(q0, r)    — which calls file.InsertAt + notifies observers
   c. SetSelect(...)         — update selection state
   d. ScrDraw(...)           — update scroll state
   e. w.Commit(t)            — commit text state
   f. acmeputsnarf()         — sync to system clipboard
```

The `previewExecute()` path (B2 commands like typing "Cut" or "Paste") is correct — it increments `global.seq`, calls `Mark()`, and dispatches to the standard `cut()`/`paste()` functions.

Only the **chord** code path (B1+B2, B1+B3, B1+B2+B3) is broken.

## Current Broken Code

```go
// wind.go:983-1009
case chordButtons == 7: // B1+B2+B3: Snarf
    if snarfed := w.PreviewSnarf(); len(snarfed) > 0 {
        global.snarfbuf = snarfed
        global.snarfContext = w.selectionContext
    }

case chordButtons&2 != 0: // B1+B2: Cut
    if snarfed := w.PreviewSnarf(); len(snarfed) > 0 {
        global.snarfbuf = snarfed
        global.snarfContext = w.selectionContext
    }
    if w.body.q0 < w.body.q1 {
        w.body.file.DeleteAt(w.body.q0, w.body.q1)  // BYPASSES UNDO
        w.UpdatePreview()
    }

case chordButtons&4 != 0: // B1+B3: Paste
    if len(global.snarfbuf) > 0 {
        if w.body.q0 < w.body.q1 {
            w.body.file.DeleteAt(w.body.q0, w.body.q1)  // BYPASSES UNDO
        }
        w.body.file.InsertAt(w.body.q0, []rune(string(global.snarfbuf)))  // BYPASSES UNDO
        w.UpdatePreview()
    }
```

## Fix

Replace all three chord handlers with calls to the standard `cut()` and `paste()` functions from `exec.go`, preceded by proper undo setup.

### Phase 19A: Fix Snarf chord (B1+B2+B3)

Replace the direct snarfbuf assignment with a call to `cut()` with `dosnarf=true, docut=false`:

```go
case chordButtons == 7: // B1+B2+B3: Snarf
    cut(&w.body, &w.body, nil, true, false, "")
    global.snarfContext = w.selectionContext
```

This calls the standard snarf path which reads from the file properly and calls `acmeputsnarf()` for system clipboard sync. We preserve `snarfContext` assignment for context-aware paste.

### Phase 19B: Fix Cut chord (B1+B2)

Replace the direct `file.DeleteAt` with proper undo setup and `cut()`:

```go
case chordButtons&2 != 0: // B1+B2: Cut
    w.body.TypeCommit()
    global.seq++
    w.body.file.Mark(global.seq)
    cut(&w.body, &w.body, nil, true, true, "")
    global.snarfContext = w.selectionContext
    w.UpdatePreview()
```

### Phase 19C: Fix Paste chord (B1+B3)

Replace the direct `file.DeleteAt`/`file.InsertAt` with proper undo setup and `paste()`:

```go
case chordButtons&4 != 0: // B1+B3: Paste
    w.body.TypeCommit()
    global.seq++
    w.body.file.Mark(global.seq)
    paste(&w.body, &w.body, nil, true, false, "")
    w.UpdatePreview()
```

### Phase 19D: Update tests

Update existing chord tests to verify undo works:

- `TestPreviewChordCut`: After cut, call `w.body.Undo(true)` and verify text is restored
- `TestPreviewChordPaste`: After paste, call `w.body.Undo(true)` and verify original text is restored
- `TestPreviewChordSnarf`: Verify `acmeputsnarf()` was called (or at minimum that the standard snarf path was used)
- Add `TestPreviewChordCutUndo` and `TestPreviewChordPasteUndo` if separate tests are cleaner

### Note on snarfContext

The `snarfContext` field is preview-specific metadata for context-aware paste (Phase 18.5). The standard `cut()`/`paste()` functions don't know about it. The fix preserves `snarfContext` assignment as a post-step after calling the standard functions. This keeps the context-aware paste feature working without modifying the standard cut/paste API.

### Note on test setup code

The tests in `wind_test.go` also call `w.body.file.InsertAt()` directly for **test setup** (initializing body content). This is acceptable — tests are setting initial state, not simulating user edit operations. No changes needed there.
