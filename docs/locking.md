# Edwood Locking Patterns

This document describes all locking mechanisms in the Edwood codebase, their expected behavior, and known issues.

---

## Overview

Edwood uses multiple synchronization mechanisms:
- **Mutexes** (`sync.Mutex`, `sync.RWMutex`) for protecting shared state
- **Condition variables** (`sync.Cond`) for signaling between goroutines
- **Channel-based locks** (`chan bool` with capacity 1) for serialization
- **Reference counting** for window lifecycle management

---

## Lock Acquisition Ordering

**Critical Rule**: When acquiring multiple locks, always follow this order to prevent deadlocks:

```
Row lock (global.row.lk)
    └─► Window lock (w.lk)
           └─► Frame lock (f.lk)
```

This ordering is established in xfid.go:196-200:
```go
// We need to lock row here before locking window (just like mousethread)
// in order to synchronize mousetext with mousethread: mousetext is
// set to nil when the associated window is closed.
global.row.lk.Lock()
w.Lock('E')
```

---

## Core Locks

### Row Lock (`global.row.lk`)

| Property | Value |
|----------|-------|
| **File** | `row.go:23` |
| **Type** | `sync.Mutex` |
| **Protects** | Row structure, column list, row tag |

**Acquisition Points**:
- `row.Type()` (row.go:278) - keyboard input handling
- `row.Dump()` (acme.go:147-149) - session serialization
- `flushwarnings()` (acme.go:367-369) - warning display
- `xfidflush()` / `xfidclose()` (xfid.go:54-55, 199-200) - 9P operations

**Hold Duration**: Brief, typically for navigation/lookup operations.

**Known Issues**:
- TODO at util.go:70: Access to `global.row.col` should be inside row lock
- TODO at util.go:139-141: Multiple places access global row without locking
- TODO at acme.go:714: Window creation should be in row lock

---

### Window Lock (`w.lk`)

| Property | Value |
|----------|-------|
| **File** | `wind.go:23` |
| **Type** | `sync.Mutex` |
| **Protects** | Window state, tags, body text, reference count, event handling |

**Key Methods**:
- `Lock(owner int)` (wind.go:368-378) - Acquires mutex, increments ref, locks clones
- `Unlock()` (wind.go:388-395) - Releases mutex, decrements ref, unlocks clones
- `lock1(owner int)` (wind.go:361-365) - Lock single window (no clones)
- `unlock1()` (wind.go:381-385) - Unlock single window

**Owner Parameter**: The `owner` parameter identifies who holds the lock:
- `'K'` - Keyboard thread
- `'E'` - Error window operations
- `'M'` - Mouse thread
- Command rune - Edit commands

**Acquisition Points**:
- `row.Type()` (row.go:290-300) - keyboard input
- `xfidopen()` / `xfidclose()` (xfid.go:89, 200) - 9P operations
- Edit command execution (ecmd.go)

**Hold Duration**: Can be held during text edits, file I/O, and command execution.

**Known Issues**:
- TODO at wind.go:360: Lock should be internal Window detail
- Suspect locking behavior in errorwin() (util.go:99-100)

---

### Frame Lock (`f.lk`)

| Property | Value |
|----------|-------|
| **File** | `frame/frame.go:261` |
| **Type** | `sync.Mutex` |
| **Protects** | Frame display state, box model, selection, text metrics |

**Locked Methods**:
- `Insert()` / `InsertByte()` (frame/insert.go:139, 145)
- `Delete()` (frame/delete.go:8)
- `Ptofchar()` / `Charofpt()` (frame/ptofchar.go:43, 64)
- `Select()` / `SelectOpt()` (frame/select.go:49, 26)
- `DrawSel()` / `Redraw()` (frame/draw.go:38, 198)
- `Maxtab()` (frame/frame.go:167)
- `GetFrameFillStatus()` (frame/frame.go:184)
- `TextOccupiedHeight()` / `textoccupiedheightimpl()` (frame/frame.go:194, 202)
- `GetSelectionExtent()` (frame/select.go:10)
- `IsLastLineFull()` (frame/frame.go:214)
- `Rect()` (frame/frame.go:220)
- `DefaultFontHeight()` (frame/frame.go:304)
- `Init()` (frame/frame.go:312)
- `Clear()` (frame/frame.go:344)

**Pattern**: All frame operations use `defer f.lk.Unlock()` for exception safety.

**Hold Duration**: Short, protecting implementation details.

**Debug Support**: Disabled `debugginglock` wrapper exists (frame/frame.go:238-254) for detecting reentrancy.

---

### Text Lock (`t.lk`)

| Property | Value |
|----------|-------|
| **File** | `text.go:85` |
| **Type** | `sync.Mutex` |
| **Protects** | Text view state (org, q0, q1, selection, frame state) |

**Locked Methods**:
- `Show()` (text.go:1239) - Scrolls and selects text range

**Hold Duration**: Short, protects view state during scroll/selection operations.

---

### Event Log Lock (`eventlog.lk`)

| Property | Value |
|----------|-------|
| **File** | `logf.go:14` |
| **Type** | `sync.Mutex` |
| **Paired With** | `sync.Cond` (`eventlog.r`) for signaling |
| **Protects** | Event log buffer and reader tracking |

**Methods**:
- `xfidlogopen()` (logf.go:31)
- `xfidlogclose()` (logf.go:38)
- `xfidlogread()` (logf.go:51) - Uses condition variable Wait/Broadcast
- `xfidlogflush()` (logf.go:87)
- `xfidlog()` (logf.go:118)

**Known Issues**:
- TODO at logf.go:61: "Did I get the Rendez right?" - uncertainty about condition variable correctness

---

### Mount Directory Lock (`mnt.lk`)

| Property | Value |
|----------|-------|
| **File** | `fsys.go:98` |
| **Type** | `sync.Mutex` |
| **Protects** | Reference-counted mount directory map |

**Methods**:
- `Mnt.Add()` (fsys.go:171)
- `Mnt.IncRef()` (fsys.go:189)
- `Mnt.DecRef()` (fsys.go:199)

**Pattern**: Simple ref-counting with lock/unlock around increment/decrement.

---

### Warnings Lock (`warningsMu`)

| Property | Value |
|----------|-------|
| **File** | `util.go:206` |
| **Type** | `sync.Mutex` |
| **Protects** | Global warnings list |

**Methods**:
- `warning()` (util.go:244) - Locks to append

**Known Issues**:
- TODO at util.go:209: `flushwarnings()` does not lock `warningsMu`

---

### Control Lock (`w.ctrllock`)

| Property | Value |
|----------|-------|
| **File** | `wind.go:52` |
| **Type** | `sync.Mutex` |
| **Status** | **DISABLED** |

**Purpose**: Exclusive use lock for window control (lock/unlock ctl messages).

**Critical Issue** (xfid.go:569-580):
```go
// Lock/unlock can hang or crash Edwood. They don't appear to be
// used for anything useful, so disable for now.
```

Both lock and unlock operations return `ErrBadCtl` to clients.

---

## Channel-Based Locks

### Edit Output Lock (`editoutlk`)

| Property | Value |
|----------|-------|
| **Global** | `global.editoutlk` (globals.go:246) |
| **Per-Window** | `w.editoutlk` (wind.go:68) |
| **Type** | `chan bool` (capacity 1) |
| **Protects** | Edit command output serialization |

**Purpose**: Prevents interleaved output from concurrent edit operations.

**Usage Pattern**:
```go
// Acquire
select { case w.editoutlk <- true: ... }

// Release
<-w.editoutlk
```

**Acquisition Points**:
- `xfidopen()` (xfid.go:149, 164)
- `xfidclose()` (xfid.go:235, 243)

---

## Reference Counting

### Ref Type

| Property | Value |
|----------|-------|
| **File** | `dat.go:155-164` |
| **Type** | `type Ref int` |
| **Purpose** | Window lifecycle management |

**Methods**:
```go
func (r *Ref) Inc() { *r++ }
func (r *Ref) Dec() int { *r--; return int(*r) }
```

**Window Lifecycle**:
1. Initialized with `Inc()` in `initHeadless()` (wind.go:104)
2. Extra `Inc()` if `global.globalincref` true (wind.go:105-106)
3. `Lock()` increments ref (wind.go:363, 370)
4. `unlock1()` calls `Close()` which does `if w.ref.Dec() == 0 { cleanup }` (wind.go:405)

**Note**: Reference counting is not atomic; relies on lock protection.

---

## Multi-Window Locking (X/Y Commands)

The X and Y edit commands require special handling to safely iterate over multiple windows.

**Pattern** (ecmd.go:847-909):
1. Increment ref on ALL windows via `alllocker(w, true)` (line 878)
2. Set `global.globalincref = true` (line 879)
3. Unlock source window (line 884)
4. For each target window: `Lock()`, execute, `Unlock()` (lines 891, 895)
5. Re-lock source window (line 900)
6. Decrement all refs via `alllocker(w, false)` (line 903)
7. Reset `global.globalincref = false` (line 904)

**Purpose**: Protects against windows being deleted during batch operations.

**Known Issues**:
- TODO at ecmd.go:876-877: "We lock all windows but only mutate some of them? Improve concurrency opportunities."

---

## Other Locks

### Rich Text Image Cache Lock

| Property | Value |
|----------|-------|
| **File** | `rich/image.go:249` |
| **Type** | `sync.RWMutex` |
| **Protects** | LRU image cache for markdown preview |

### Mock Display Lock (Tests)

| Property | Value |
|----------|-------|
| **File** | `edwoodtest/draw.go:46` |
| **Type** | `sync.Mutex` |
| **Purpose** | Test utility for capturing draw operations |

### Win Command Locks (External Tool)

| Property | Value |
|----------|-------|
| **File** | `cmd/win/win.go:41, 489` |
| **Type** | `sync.Mutex` |
| **Purpose** | Serialization in external window command tool |

---

## Known Issues Summary

| Location | Issue | Severity |
|----------|-------|----------|
| util.go:70 | Row column access without lock | Medium |
| util.go:99-100 | Suspect locking in errorwin() | Medium |
| util.go:139-141 | Global row access without locking | Medium |
| util.go:209 | flushwarnings() doesn't lock warningsMu | Medium |
| acme.go:714 | Window creation not in row lock | Medium |
| xfid.go:569-580 | Lock/unlock ctl disabled (hangs/crashes) | High |
| logf.go:61 | Condition variable correctness uncertain | Low |
| ecmd.go:876-877 | Coarse locking in X/Y commands | Low |
| exec.go:922 | runproc window mutation needs lock | Medium |

---

## Testing Recommendations

1. **Race Detection**: Always run tests with `-race` flag
   ```bash
   go test -race ./...
   ```

2. **Lock Paths**: Ensure 100% coverage of lock acquisition paths

3. **Deadlock Prevention**: Test concurrent operations that acquire multiple locks

4. **Stress Testing**: Test X/Y commands with many windows to verify reference counting

---

## Summary Table

| Lock | Type | Location | Protects | Status |
|------|------|----------|----------|--------|
| `row.lk` | Mutex | row.go:23 | Row/columns/windows | Active |
| `w.lk` | Mutex | wind.go:23 | Window state | Active |
| `f.lk` | Mutex | frame.go:261 | Frame display | Active |
| `t.lk` | Mutex | text.go:85 | Text view | Active |
| `eventlog.lk` | Mutex | logf.go:14 | Event log | Active |
| `eventlog.r` | Cond | logf.go:15 | Event signaling | Active |
| `mnt.lk` | Mutex | fsys.go:98 | Mount dir refs | Active |
| `w.ctrllock` | Mutex | wind.go:52 | Window exclusive use | **Disabled** |
| `warningsMu` | Mutex | util.go:206 | Warnings list | Active |
| `editoutlk` | Chan | globals.go:246, wind.go:68 | Edit output | Active |
| `w.ref` | Ref int | wind.go:24 | Reference count | Active |
