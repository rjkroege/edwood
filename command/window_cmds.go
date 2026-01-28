// Package command provides command dispatch functionality for edwood.
// This file contains types and helpers for window commands (Newcol, Delcol, Sort, Zerox).
package command

import (
	"sort"
)

// =============================================================================
// Window Info Type
// =============================================================================

// WindowInfo represents information about a window for column operations.
// This type captures the state needed to make decisions about column commands
// like Delcol.
type WindowInfo struct {
	Name          string
	IsDirty       bool
	HasRunningCmd bool // true if window has running external command (nopen[QWevent] > 0)
}

// =============================================================================
// Column State Types
// =============================================================================

// ColumnState represents the state of a column for window commands.
// This type captures the state needed to make decisions about column commands
// like Delcol and Sort.
type ColumnState struct {
	windows []WindowInfo
}

// NewColumnState creates a new ColumnState with the given windows.
func NewColumnState(windows []WindowInfo) *ColumnState {
	return &ColumnState{
		windows: windows,
	}
}

// WindowCount returns the number of windows in the column.
func (c *ColumnState) WindowCount() int {
	return len(c.windows)
}

// IsClean returns true if all windows in the column are clean (not dirty).
// This matches the behavior of Column.Clean() in col.go.
func (c *ColumnState) IsClean() bool {
	for _, w := range c.windows {
		if w.IsDirty {
			return false
		}
	}
	return true
}

// HasRunningCommand returns true if any window has a running external command.
func (c *ColumnState) HasRunningCommand() bool {
	for _, w := range c.windows {
		if w.HasRunningCmd {
			return true
		}
	}
	return false
}

// RunningCommandWindows returns the names of windows with running commands.
func (c *ColumnState) RunningCommandWindows() []string {
	var names []string
	for _, w := range c.windows {
		if w.HasRunningCmd {
			names = append(names, w.Name)
		}
	}
	return names
}

// CanDelete determines if the column can be deleted.
// A column can be deleted if:
// - It has no dirty windows (or Clean() returns true)
// - No windows have running external commands
//
// Returns (canDelete, reason) where reason is non-empty if canDelete is false.
func (c *ColumnState) CanDelete() (bool, string) {
	// Check for running commands first (more specific error)
	if c.HasRunningCommand() {
		return false, "running"
	}

	// Check for dirty windows
	if !c.IsClean() {
		return false, "dirty"
	}

	return true, ""
}

// =============================================================================
// Sort Operation Types
// =============================================================================

// SortOperation represents the parameters for a sort operation.
// Sort arranges windows in a column alphabetically by file name.
type SortOperation struct{}

// NewSortOperation creates a new SortOperation.
func NewSortOperation() *SortOperation {
	return &SortOperation{}
}

// RequiresColumn returns true because Sort operates on a column.
func (s *SortOperation) RequiresColumn() bool {
	return true
}

// SortWindowNames sorts a slice of window names alphabetically.
// This matches the behavior of Column.Sort() which uses file names.
func SortWindowNames(names []string) []string {
	result := make([]string, len(names))
	copy(result, names)
	sort.Strings(result)
	return result
}

// =============================================================================
// Zerox Operation Types
// =============================================================================

// ZeroxState represents the state needed for a Zerox (clone window) operation.
type ZeroxState struct {
	hasWindow bool
	isDir     bool
	fileName  string
}

// NewZeroxState creates a new ZeroxState.
func NewZeroxState(hasWindow, isDir bool, fileName string) *ZeroxState {
	return &ZeroxState{
		hasWindow: hasWindow,
		isDir:     isDir,
		fileName:  fileName,
	}
}

// HasWindow returns true if there is a window to clone.
func (z *ZeroxState) HasWindow() bool {
	return z.hasWindow
}

// IsDirectory returns true if the file is a directory.
func (z *ZeroxState) IsDirectory() bool {
	return z.isDir
}

// FileName returns the file name.
func (z *ZeroxState) FileName() string {
	return z.fileName
}

// CanClone determines if the window can be cloned.
// A window can be cloned if:
// - There is a window (t.w != nil)
// - The file is not a directory
//
// Returns (canClone, reason) where reason is non-empty if canClone is false.
func (z *ZeroxState) CanClone() (bool, string) {
	if !z.hasWindow {
		return false, "no window"
	}
	if z.isDir {
		return false, "directory"
	}
	return true, ""
}

// =============================================================================
// Newcol Operation Types
// =============================================================================

// NewcolOperation represents the parameters for a Newcol operation.
// Newcol creates a new column and adds an empty window to it.
type NewcolOperation struct{}

// NewNewcolOperation creates a new NewcolOperation.
func NewNewcolOperation() *NewcolOperation {
	return &NewcolOperation{}
}

// RequiresRow returns true because Newcol needs a row to add the column to.
func (n *NewcolOperation) RequiresRow() bool {
	return true
}

// AddsWindowToColumn returns true because Newcol adds an empty window.
func (n *NewcolOperation) AddsWindowToColumn() bool {
	return true
}

// =============================================================================
// Window Command Registry
// =============================================================================

// WindowCommandRegistry provides standard window command entries for registration.
type WindowCommandRegistry struct{}

// NewWindowCommandRegistry creates a new WindowCommandRegistry.
func NewWindowCommandRegistry() *WindowCommandRegistry {
	return &WindowCommandRegistry{}
}

// RegisterWindowCommands registers all window commands with the dispatcher.
// The commands registered are: Newcol, Delcol, Sort, Zerox
func (r *WindowCommandRegistry) RegisterWindowCommands(d *Dispatcher) {
	// These match the entries in globalexectab for window commands
	// Format: name, mark (undoable), flag1, flag2
	// None of these commands are undoable (they don't modify text buffers)
	d.RegisterCommand(NewCommandEntry("Newcol", false, true, true)) // flags unused
	d.RegisterCommand(NewCommandEntry("Delcol", false, true, true)) // flags unused
	d.RegisterCommand(NewCommandEntry("Sort", false, true, true))   // flags unused
	d.RegisterCommand(NewCommandEntry("Zerox", false, true, true))  // flags unused
}
