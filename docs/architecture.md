# Edwood Architecture

This document describes the high-level architecture of Edwood, a Go port of Rob Pike's Acme editor.

## Overview

Edwood is a programmable text editor based on Plan 9's Acme. It provides:
- A tiled window manager for text files
- Mouse-driven command execution (button chords)
- A 9P file server for external program integration
- Rich text preview mode for markdown rendering

---

## Component Hierarchy

The UI follows a strict containment hierarchy:

```
Row (global.row)
├── tag: Text (Rowtag) - "Newcol Kill Putall Dump Exit"
└── col: []*Column
    ├── tag: Text (Columntag) - "New Cut Paste Snarf Sort Zerox Delcol"
    └── w: []*Window
        ├── tag: Text (Tag) - filename and commands
        └── body: Text (Body) - file contents
```

### Row

The root container. Manages columns and the global tag. Contains:
- `display`: Graphics context
- `lk`: Mutex for thread-safe access
- `col`: Slice of Column pointers
- `tag`: Row-level command tag

**File:** `row.go`

### Column

Vertical container for windows. Manages:
- Window layout and resizing
- Column tag with standard commands
- Drag operations for window rearrangement

**File:** `col.go`

### Window

Container for a file view. Contains:
- `tag`: Filename and window-specific commands
- `body`: File content display
- Preview mode state for rich text rendering
- Lock for thread-safe operations

**File:** `wind.go`

### Text

View onto an `ObservableEditableBuffer`. Manages:
- Frame display (via `frame.Frame`)
- Selection state (q0, q1)
- Scroll position (org)
- Input handling

**File:** `text.go`

---

## Package Structure

```
edwood/
├── Main Application
│   ├── acme.go         # Entry point, main loop, thread spawning
│   ├── globals.go      # Global state struct
│   ├── dat.go          # Constants, types, Qid helpers
│   ├── row.go          # Row - top-level container
│   ├── col.go          # Column - window container
│   ├── wind.go         # Window - file view container
│   ├── text.go         # Text - buffer view with frame
│   └── look.go         # Mouse click handling, plumbing
│
├── Commands
│   ├── exec.go         # Command dispatch table, built-in commands
│   ├── ecmd.go         # Edit command implementations
│   ├── edit.go         # Sam-style edit language parser
│   └── addr.go         # Address parsing and evaluation
│
├── 9P File Server
│   ├── fsys.go         # 9P server implementation
│   ├── xfid.go         # File operation handlers (read/write/etc.)
│   └── ninep/          # 9P utilities
│
├── Text Management
│   ├── file/                  # Buffer implementation
│   │   ├── buffer.go          # Gap buffer
│   │   ├── observable_*.go    # Observer pattern for updates
│   │   └── diskdetails.go     # File metadata
│   ├── frame/                 # Frame rendering (libframe port)
│   │   ├── frame.go           # Frame struct
│   │   ├── box.go             # Box model for text
│   │   ├── insert.go          # Text insertion
│   │   ├── delete.go          # Text deletion
│   │   └── select.go          # Selection handling
│   └── runes/                 # Rune utilities
│
├── Rich Text (Preview Mode)
│   ├── richtext.go            # RichText component
│   ├── rich/                  # Rich text rendering
│   │   ├── frame.go           # RichFrame - styled text layout
│   │   ├── style.go           # Style definitions
│   │   ├── span.go            # Attributed text spans
│   │   ├── box.go             # Styled boxes
│   │   ├── layout.go          # Layout engine
│   │   └── image.go           # Image loading/caching
│   └── markdown/              # Markdown parsing
│       ├── parse.go           # Markdown → Spans
│       ├── sourcemap.go       # Position mapping
│       └── linkmap.go         # Link tracking
│
├── Display
│   ├── draw/                  # Graphics abstraction
│   │   └── drawutil/          # Scroll bar utilities
│   └── scrl.go                # Scroll bar rendering
│
├── Utilities
│   ├── util.go                # Error windows, helpers
│   ├── complete/              # Filename completion
│   ├── dumpfile/              # Session save/restore
│   ├── regexp/                # Regex for rune slices
│   └── sam/                   # Sam edit log
│
└── Commands (cmd/)
    ├── E/                     # Edit script command
    ├── win/                   # New terminal window
    └── logtowin/              # Logging utility
```

---

## Threading Model

Edwood uses multiple goroutines coordinated via channels:

```
┌─────────────────────────────────────────────────────────────────┐
│                         Main Thread                              │
│  (initialization, then blocks on cexit/csignal)                 │
└─────────────────────────────────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
┌───────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ mousethread   │     │ keyboardthread  │     │ waitthread      │
│               │     │                 │     │                 │
│ - Mouse input │     │ - Keyboard      │     │ - Process exit  │
│ - Resize      │     │   input         │     │ - Command mgmt  │
│ - Plumb msgs  │     │ - Timer for     │     │ - Error display │
│ - Warnings    │     │   tag commit    │     │                 │
└───────────────┘     └─────────────────┘     └─────────────────┘

┌───────────────────┐     ┌─────────────────────────────────────┐
│ newwindowthread   │     │ xfidallocthread                     │
│                   │     │                                     │
│ - 9P window       │     │ - Xfid pool management              │
│   creation        │     │ - Spawns xfidctl goroutines         │
└───────────────────┘     └─────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    fsysproc (9P server)                         │
│  - Handles 9P messages from clients                             │
│  - Dispatches to xfidctl workers                                │
└─────────────────────────────────────────────────────────────────┘
```

### Key Channels (in globals struct)

| Channel | Type | Purpose |
|---------|------|---------|
| `cplumb` | `*plumb.Message` | Plumb messages from plumber |
| `cwait` | `ProcessState` | Process exit notifications |
| `ccommand` | `*Command` | New command tracking |
| `ckill` | `string` | Kill command requests |
| `cxfidalloc` | `*Xfid` | Xfid allocation requests |
| `cxfidfree` | `*Xfid` | Xfid deallocation |
| `cnewwindow` | `*Window` | New window creation |
| `cexit` | `struct{}` | Editor shutdown signal |
| `cerr` | `error` | Error display requests |
| `cedit` | `int` | Edit command completion |
| `cwarn` | `uint` | Warning display trigger |

### Locking Strategy

- **Row lock (`global.row.lk`)**: Protects UI hierarchy. Acquired by mouse/keyboard threads.
- **Window lock (`w.lk`)**: Per-window operations. Methods accept lock reason character.
- **Text lock (`t.lk`)**: Buffer access within Text.

Pattern: Acquire row lock first, then window lock, then text lock.

---

## Data Flow

### Mouse Click → Command Execution

```
mousectl.C ──► mousethread ──► MovedMouse()
                                    │
                                    ▼
                             row.Which(point)
                                    │
                                    ▼
                              Text.Select2()    (B2 click)
                                    │
                                    ▼
                              execute()
                                    │
                                    ▼
                         globalexectab lookup
                                    │
                                    ▼
                            command function
```

### 9P Read Request

```
Client ──9P──► fsysproc ──► xfidctl worker
                                  │
                                  ▼
                            xfidread()
                                  │
                         ┌────────┴────────┐
                         ▼                 ▼
                    QWbody             QWevent
                         │                 │
                         ▼                 ▼
                  w.body.file.Read   Event formatting
```

### Text Insertion

```
User types ──► keyboardthread ──► row.Type()
                                      │
                                      ▼
                                 text.Type()
                                      │
                                      ▼
                              text.file.Insert()
                                      │
                          ┌───────────┴───────────┐
                          ▼                       ▼
                   Update buffer           Notify observers
                          │                       │
                          ▼                       ▼
                   file.Buffer.Insert     Text.Inserted() callback
                                                  │
                                                  ▼
                                          frame.Insert()
```

---

## 9P File Server

Edwood exposes a 9P file server at `/mnt/acme` (configurable via `-m` flag).

### File Hierarchy

```
/mnt/acme/
├── acme/               # Global directory
│   ├── cons            # Console output
│   ├── consctl         # Console control
│   ├── index           # Window list
│   ├── label           # Window manager title
│   ├── log             # Event log
│   └── new/            # Create window (walk here)
└── <winid>/            # Per-window directory
    ├── addr            # Address register
    ├── body            # Body content (append-only)
    ├── ctl             # Control commands
    ├── data            # Body with addr positioning
    ├── errors          # Error output (+Errors window)
    ├── event           # Event stream
    ├── rdsel           # Read selection
    ├── wrsel           # Write selection
    ├── tag             # Tag content (append-only)
    └── xdata           # Extended data access
```

### Qid Encoding

Window ID and file type encoded in `Qid.Path`:
```
Qid.Path = (windowID << 8) | fileType

fileType: Qdir, QWbody, QWctl, QWevent, etc.
```

---

## Key Abstractions

### ObservableEditableBuffer (file package)

Facade wrapping a gap buffer with:
- Undo/redo support via edit log
- Observer pattern for change notifications
- File metadata (name, hash, dirty state)

### Frame (frame package)

Port of Plan 9's libframe. Displays editable text in a fixed-width font grid:
- Box model for text layout
- Selection rendering
- Tick (cursor) management

### RichFrame (rich package)

Extension of Frame for styled text:
- Multiple fonts/sizes
- Colors
- Image embedding
- Markdown preview support

---

## Global State

Centralized in `globals` struct (`globals.go`). Key fields:

| Field | Purpose |
|-------|---------|
| `row` | Root UI container |
| `mouse` | Current mouse state |
| `activewin` | Focused window |
| `activecol` | Active column |
| `seltext` | Last B1-clicked text |
| `snarfbuf` | Clipboard contents |
| `seq` | Undo sequence counter |
| `WinID` | Window ID counter |

---

## Command System

### Built-in Commands (exec.go)

Defined in `globalexectab`. Commands like:
- `Cut`, `Paste`, `Snarf` - clipboard operations
- `Put`, `Get` - file I/O
- `New`, `Del`, `Zerox` - window management
- `Edit` - Sam-style editing

### Edit Language (edit.go)

Sam-compatible structured editing:
```
,x/pattern/ c/replacement/
```

Parsed by `edit.go`, executed by `ecmd.go`.

---

## Preview Mode

Windows can toggle between text and rich text preview:

```
Text Mode                          Preview Mode
┌─────────────────┐               ┌─────────────────┐
│ # Heading       │   Markdeep    │ Heading         │
│                 │   ────────►   │                 │
│ Some **bold**   │               │ Some bold       │
│ text here       │               │ text here       │
└─────────────────┘               └─────────────────┘
     body.Text                        richBody
```

Components:
- `markdown/parse.go` - Markdown → `[]rich.Span`
- `rich/frame.go` - Styled layout and rendering
- `richtext.go` - Preview display component
- `wind.go` - Preview state management
