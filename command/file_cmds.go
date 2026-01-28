// Package command provides command dispatch functionality for edwood.
// This file contains types and helpers for file commands (Get, Put, Putall, New, Del).
package command

import (
	"strings"
)

// FileInfo represents information about a file for command operations.
// This type captures the state needed to make decisions about file commands
// like Del, Get, Put, etc.
type FileInfo struct {
	name         string
	isDir        bool
	dirty        bool
	hasObservers bool // true if multiple windows view this file
}

// NewFileInfo creates a new FileInfo with the given attributes.
func NewFileInfo(name string, isDir, dirty, hasObservers bool) *FileInfo {
	return &FileInfo{
		name:         name,
		isDir:        isDir,
		dirty:        dirty,
		hasObservers: hasObservers,
	}
}

// Name returns the file name.
func (f *FileInfo) Name() string { return f.name }

// IsDir returns true if this is a directory.
func (f *FileInfo) IsDir() bool { return f.isDir }

// IsDirty returns true if the file has unsaved changes.
func (f *FileInfo) IsDirty() bool { return f.dirty }

// HasMultipleObservers returns true if multiple windows observe this file.
func (f *FileInfo) HasMultipleObservers() bool { return f.hasObservers }

// PutallCandidate represents a window that may be saved by the Putall command.
type PutallCandidate struct {
	Name       string
	Dirty      bool
	HasEvent   bool // nopen[QWevent] > 0 means running external command
	FileExists bool
}

// ShouldSave returns true if this candidate should be saved by Putall.
// Putall skips files that:
// - Have a running external command (HasEvent)
// - Are not dirty
// - Don't exist on disk (new files that haven't been Put yet)
func (c *PutallCandidate) ShouldSave() bool {
	if c.HasEvent {
		return false
	}
	if !c.Dirty {
		return false
	}
	if !c.FileExists {
		return false
	}
	return true
}

// CanDelete determines if a file can be deleted based on the del() logic.
// Parameters:
//   - forceClose: true if "Delete" command (force), false if "Del" command
//
// Returns true if the window can be closed.
func (f *FileInfo) CanDelete(forceClose bool) bool {
	// From del(): flag1 || et.w.body.file.HasMultipleObservers() || et.w.Clean(false)
	// Clean(false) returns true if file is not dirty
	return forceClose || f.hasObservers || !f.dirty
}

// CanGet determines if a file can be loaded with the Get command.
// Parameters:
//   - newName: the name of the file to load
//   - newIsDir: true if the new file is a directory
//
// Returns (canLoad, errorReason) where errorReason is non-empty if canLoad is false.
func (f *FileInfo) CanGet(newName string, newIsDir bool) (bool, string) {
	// Check for dirty window (unless same name - reload is allowed)
	if f.dirty && f.name != newName {
		return false, "dirty"
	}

	// Check for directory with multiple observers
	if newIsDir && f.hasObservers {
		return false, "directory"
	}

	return true, ""
}

// CanPut determines if a file can be saved with the Put command.
// Returns (canSave, errorReason) where errorReason is non-empty if canSave is false.
func (f *FileInfo) CanPut() (bool, string) {
	if f.isDir {
		return false, "directory"
	}
	return true, ""
}

// ParseNewArgs parses the argument string for the New command.
// The New command accepts space-separated file names.
// Returns nil if arg is empty (creating an empty window).
func ParseNewArgs(arg string) []string {
	if arg == "" {
		return nil
	}

	// Normalize whitespace (matches behavior in newx())
	normalized := wsre.ReplaceAllString(arg, " ")
	normalized = strings.TrimSpace(normalized)

	if normalized == "" {
		return nil
	}

	// Split on spaces and filter empty strings
	parts := strings.Split(normalized, " ")
	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// GetNameResult encapsulates the result of resolving a filename for Get/Put commands.
type GetNameResult struct {
	Name      string
	UseArgDir bool // true if name should be prefixed with arg directory
}

// ResolveGetName determines the filename to use for Get command.
// Parameters:
//   - currentName: the current file name in the window
//   - arg: the argument from the command
//
// Returns the resolved filename.
func ResolveGetName(currentName, arg string) string {
	if arg != "" {
		return arg
	}
	return currentName
}

// ResolvePutName determines the filename to use for Put command.
// Parameters:
//   - currentName: the current file name in the window
//   - arg: the argument from the command
//
// Returns the resolved filename.
func ResolvePutName(currentName, arg string) string {
	if arg != "" {
		return arg
	}
	return currentName
}

// FileCommandRegistry provides standard file command entries for registration.
type FileCommandRegistry struct{}

// NewFileCommandRegistry creates a new FileCommandRegistry.
func NewFileCommandRegistry() *FileCommandRegistry {
	return &FileCommandRegistry{}
}

// RegisterFileCommands registers all file commands with the dispatcher.
// The commands registered are: Del, Delete, Get, New, Put, Putall
func (r *FileCommandRegistry) RegisterFileCommands(d *Dispatcher) {
	// These match the entries in globalexectab for file commands
	// Format: name, mark (undoable), flag1, flag2
	d.RegisterCommand(NewCommandEntry("Del", false, false, true))
	d.RegisterCommand(NewCommandEntry("Delete", false, true, true)) // flag1=true means force delete
	d.RegisterCommand(NewCommandEntry("Get", false, true, true))
	d.RegisterCommand(NewCommandEntry("New", false, true, true))
	d.RegisterCommand(NewCommandEntry("Put", false, true, true))
	d.RegisterCommand(NewCommandEntry("Putall", false, true, true))
}
