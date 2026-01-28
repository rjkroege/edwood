package main

import (
	"image"
	"log"
	"os"
	"strconv"

	"9fans.net/go/plumb"
	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/frame"
	"github.com/rjkroege/edwood/internal/ui"
)

// globals holds all global state for the Edwood editor.
// This struct centralizes state that was originally scattered across
// package-level variables, facilitating testing and eventual refactoring.
type globals struct {
	// ═══════════════════════════════════════════════════════════════════
	// Undo/Redo State
	// ═══════════════════════════════════════════════════════════════════

	// globalincref indicates that file reference counts should be incremented
	// globally during Edit X commands, preventing premature file cleanup while
	// a multi-file edit operation is in progress.
	globalincref bool

	// seq is the global undo/redo sequence counter. It is incremented before
	// marking a file with file.Mark(seq) to create undo checkpoints. All files
	// share this counter to maintain consistent undo ordering across windows.
	seq int

	// ═══════════════════════════════════════════════════════════════════
	// Tab Settings
	// ═══════════════════════════════════════════════════════════════════

	// maxtab is the tab stop width in units of the '0' character width.
	// Configured via the "tabstop" environment variable; defaults to 4.
	maxtab uint

	// tabexpand controls whether tabs are expanded to spaces on input.
	// Configured via the "tabexpand" environment variable; defaults to false.
	tabexpand bool

	// ═══════════════════════════════════════════════════════════════════
	// Input Devices
	// ═══════════════════════════════════════════════════════════════════

	// tagfont is the font specification string used for window tags.
	tagfont string

	// mouse holds the current mouse position and button state.
	mouse *draw.Mouse

	// mousectl provides mouse event channel and cursor control.
	mousectl *draw.Mousectl

	// keyboardctl provides keyboard event channel.
	keyboardctl *draw.Keyboardctl

	// mousestate tracks saved mouse position state for cursor restoration.
	// Used by operations that temporarily move the cursor (like column resizing)
	// to restore the cursor position after the operation completes.
	mousestate *ui.MouseState

	// ═══════════════════════════════════════════════════════════════════
	// UI Button Images
	// ═══════════════════════════════════════════════════════════════════

	// modbutton is the image displayed in the tag scroll area when a window
	// has unsaved modifications (dirty state). Displays a filled blue box.
	modbutton draw.Image

	// colbutton is the image displayed in the column tag scroll area.
	// Displays a solid purple-blue box to indicate column headers.
	colbutton draw.Image

	// button is the default image displayed in the tag scroll area for
	// unmodified windows. Displays an unfilled bordered box.
	button draw.Image

	// but2col is the highlight color (red, 0xAA0000FF) used during mouse
	// button 2 (middle button) text sweeps to indicate execute selection.
	but2col draw.Image

	// but3col is the highlight color (green, 0x006600FF) used during mouse
	// button 3 (right button) text sweeps to indicate look/plumb selection.
	but3col draw.Image

	// ═══════════════════════════════════════════════════════════════════
	// Main UI Structure
	// ═══════════════════════════════════════════════════════════════════

	// row is the main Row containing all columns and windows. This is the
	// root of the window hierarchy: Row -> Columns -> Windows -> Text.
	row Row

	// ═══════════════════════════════════════════════════════════════════
	// Text Selection State
	// ═══════════════════════════════════════════════════════════════════

	// seltext is the Text where the most recent mouse click occurred.
	// Used by execute commands to determine the selection context.
	// Set in look.go when processing clicks.
	seltext *Text

	// argtext is the Text containing the argument for B2B3 chord commands.
	// When a B2 sweep includes an argument, this points to the Text
	// containing that argument for command execution.
	argtext *Text

	// mousetext tracks which Text the mouse is currently over.
	// Global because Text.Close needs to clear it when a Text is destroyed.
	mousetext *Text

	// typetext tracks which Text is receiving keyboard input.
	// Global because Text.Close needs to clear it when a Text is destroyed.
	typetext *Text

	// barttext is the Text that receives keyboard input from the keyboard
	// thread. Shared between mousethread and keyboardthread to coordinate
	// which Text receives typed characters. Set when focus changes.
	barttext *Text

	// ═══════════════════════════════════════════════════════════════════
	// Active Window/Column State
	// ═══════════════════════════════════════════════════════════════════

	// activewin is the currently focused Window. Used by external commands
	// (via $winid) to determine which window to operate on. Set when a
	// window gains focus; cleared when the window is closed.
	activewin *Window

	// activecol is the currently active Column. New windows are created
	// in the active column by default. Set when clicking in a column;
	// cleared when the column is deleted.
	activecol *Column

	// ═══════════════════════════════════════════════════════════════════
	// Clipboard (Snarf Buffer)
	// ═══════════════════════════════════════════════════════════════════

	// snarfbuf holds the internal clipboard contents as raw bytes.
	// Synchronized with the system clipboard via display.WriteSnarf()
	// after cut/copy operations.
	snarfbuf []byte

	// snarfContext stores rich text metadata (content type, formatting)
	// associated with the snarfed selection. Used to preserve formatting
	// when pasting within Edwood's rich text preview mode.
	snarfContext *SelectionContext

	// ═══════════════════════════════════════════════════════════════════
	// Environment/Directory State
	// ═══════════════════════════════════════════════════════════════════

	// home is the user's home directory path, obtained from os.UserHomeDir().
	// Used for ~ expansion in file paths and as fallback directory.
	home string

	// acmeshell is the shell to use for executing commands, from the
	// "acmeshell" environment variable. If empty, defaults to the system shell.
	acmeshell string

	// wdir is the current working directory for the editor session.
	// Used to resolve relative file paths. Persisted in dump files.
	wdir string

	// ═══════════════════════════════════════════════════════════════════
	// Color Schemes
	// ═══════════════════════════════════════════════════════════════════

	// tagcolors holds the color scheme for window tags (title bars).
	// Indices are frame.ColBack, ColHigh, ColBord, ColText, ColHText.
	// Defaults: pale blue-green background, pale grey-green highlight,
	// purple-blue border, black text.
	tagcolors [frame.NumColours]draw.Image

	// textcolors holds the color scheme for window body text.
	// Indices are frame.ColBack, ColHigh, ColBord, ColText, ColHText.
	// Defaults: pale yellow background, dark yellow highlight,
	// yellow-green border, black text.
	textcolors [frame.NumColours]draw.Image

	// ═══════════════════════════════════════════════════════════════════
	// Edit Command State
	// ═══════════════════════════════════════════════════════════════════

	// editing tracks the current state of Edit command processing:
	//   Inactive (0)   - no edit command running
	//   Inserting (1)  - edit command is inserting text
	//   Collecting (2) - edit command is collecting input
	// Used to coordinate between the Edit command parser and 9P file operations.
	editing int

	// ═══════════════════════════════════════════════════════════════════
	// Inter-goroutine Communication Channels
	// ═══════════════════════════════════════════════════════════════════

	// cplumb receives plumb messages from the Plan 9 plumber service.
	// Plumb messages request opening files or performing lookups.
	cplumb chan *plumb.Message

	// cwait receives process exit status when external commands complete.
	// Used by waitthread to track command execution results.
	cwait chan ProcessState

	// ccommand receives new Command structs when external commands are started.
	// Used by waitthread to track running commands for the Kill menu.
	ccommand chan *Command

	// ckill receives command name patterns to terminate. When a name is
	// sent, waitthread attempts to kill matching running commands.
	ckill chan string

	// cxfidalloc is used to request new Xfid allocations from the 9P server.
	// Send nil to request an Xfid; receive the allocated Xfid back.
	cxfidalloc chan *Xfid

	// cxfidfree receives Xfids to be returned to the free pool.
	cxfidfree chan *Xfid

	// cnewwindow coordinates window creation from 9P requests.
	// Send nil to request a new window; receive the created Window back.
	cnewwindow chan *Window

	// cexit signals editor shutdown. Closed when Exit command is executed.
	cexit chan struct{}

	// csignal receives OS signals (SIGINT, etc.) for graceful shutdown.
	csignal chan os.Signal

	// cerr receives error messages from background goroutines for display
	// in the +Errors window.
	cerr chan error

	// cedit signals completion of Edit command execution. Used to synchronize
	// between editthread and callers waiting for edit completion.
	cedit chan int

	// cwarn is used to signal warning conditions that need user attention.
	// Sends trigger warning display in the UI.
	cwarn chan uint

	// ═══════════════════════════════════════════════════════════════════
	// Edit Command Synchronization
	// ═══════════════════════════════════════════════════════════════════

	// editoutlk is a mutex channel (capacity 1) that serializes access to
	// the edit command output. Used to prevent interleaved output from
	// concurrent edit operations writing to the same window.
	editoutlk chan bool

	// ═══════════════════════════════════════════════════════════════════
	// Window ID Generation
	// ═══════════════════════════════════════════════════════════════════

	// WinID is the counter for generating unique window IDs. Incremented
	// each time a new Window is created. The ID is exposed to external
	// programs via the $winid environment variable.
	WinID int
}

// Singleton global object.
var global *globals

// Preserve existing global semantics.
// TODO(rjk): Remove this *eventually*.
func init() {
	global = makeglobals()
}

func makeglobals() *globals {
	g := &globals{
		acmeshell:  os.Getenv("acmeshell"),
		editing:    Inactive,
		editoutlk:  make(chan bool, 1),
		cwait:      make(chan ProcessState),
		ccommand:   make(chan *Command),
		ckill:      make(chan string),
		cxfidalloc: make(chan *Xfid),
		cxfidfree:  make(chan *Xfid),
		cnewwindow: make(chan *Window),
		csignal:    make(chan os.Signal, 1),
		cerr:       make(chan error),
		cedit:      make(chan int),
		cexit:      make(chan struct{}),
		cwarn:      make(chan uint),
		mousestate: ui.NewMouseState(),
	}

	if home, err := os.UserHomeDir(); err == nil {
		g.home = home
	} else {
		log.Fatalf("could not get user home directory: %v", err)
	}

	if pwd, err := os.Getwd(); err == nil {
		g.wdir = pwd
	} else {
		log.Fatalf("could not get working directory: %v", err)
	}

	p := os.Getenv("tabstop")
	if p != "" {
		mt, _ := strconv.ParseInt(p, 10, 32)
		g.maxtab = uint(mt)
	}
	if g.maxtab == 0 {
		g.maxtab = 4
	}

	b := os.Getenv("tabexpand")
	if b != "" {
		te, _ := strconv.ParseBool(b)
		g.tabexpand = te
	} else {
		g.tabexpand = false
	}

	return g
}

// TODO(rjk): Can separate this out even better.
func (g *globals) iconinit(display draw.Display) {
	if g.tagcolors[frame.ColBack] == nil {
		g.tagcolors[frame.ColBack] = display.AllocImageMix(draw.Palebluegreen, draw.White)
		g.tagcolors[frame.ColHigh], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palegreygreen)
		g.tagcolors[frame.ColBord], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Purpleblue)
		g.tagcolors[frame.ColText] = display.Black()
		g.tagcolors[frame.ColHText] = display.Black()
		g.textcolors[frame.ColBack] = display.AllocImageMix(draw.Paleyellow, draw.White)
		g.textcolors[frame.ColHigh], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Darkyellow)
		g.textcolors[frame.ColBord], _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Yellowgreen)
		g.textcolors[frame.ColText] = display.Black()
		g.textcolors[frame.ColHText] = display.Black()
	}

	// ...
	r := image.Rect(0, 0, display.ScaleSize(Scrollwid+ButtonBorder), fontget(g.tagfont, display).Height()+1)
	g.button, _ = display.AllocImage(r, display.ScreenImage().Pix(), false, draw.Notacolor)
	g.button.Draw(r, g.tagcolors[frame.ColBack], nil, r.Min)
	r.Max.X -= display.ScaleSize(ButtonBorder)
	g.button.Border(r, display.ScaleSize(ButtonBorder), g.tagcolors[frame.ColBord], image.Point{})

	r = g.button.R()
	g.modbutton, _ = display.AllocImage(r, display.ScreenImage().Pix(), false, draw.Notacolor)
	g.modbutton.Draw(r, g.tagcolors[frame.ColBack], nil, r.Min)
	r.Max.X -= display.ScaleSize(ButtonBorder)
	g.modbutton.Border(r, display.ScaleSize(ButtonBorder), g.tagcolors[frame.ColBord], image.Point{})
	r = r.Inset(display.ScaleSize(ButtonBorder))
	tmp, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)
	g.modbutton.Draw(r, tmp, nil, image.Point{})

	r = g.button.R()
	g.colbutton, _ = display.AllocImage(r, display.ScreenImage().Pix(), false, draw.Purpleblue)

	g.but2col, _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xAA0000FF)
	g.but3col, _ = display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x006600FF)
}
