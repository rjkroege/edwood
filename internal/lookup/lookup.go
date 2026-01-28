// Package lookup provides unified window and file lookup utilities for edwood.
//
// This package centralizes the various lookup patterns used throughout edwood
// into a single, testable package with consistent interfaces.
package lookup

import (
	"path/filepath"
	"strings"
)

// WindowInfo provides read-only access to window properties needed for lookups.
type WindowInfo interface {
	ID() int
	Name() string
}

// ColumnInfo provides read-only access to column properties needed for lookups.
type ColumnInfo interface {
	NumWindows() int
}

// LookupResult wraps a lookup result with explicit Found flag.
type LookupResult[T any] struct {
	Window T
	Found  bool
}

// NewFound creates a LookupResult indicating the window was found.
func NewFound[T any](w T) *LookupResult[T] {
	return &LookupResult[T]{Window: w, Found: true}
}

// NewNotFound creates a LookupResult indicating the window was not found.
func NewNotFound[T any]() *LookupResult[T] {
	return &LookupResult[T]{Found: false}
}

// PathMatcher handles path matching with working directory context.
type PathMatcher struct {
	workDir string
}

// NewPathMatcher creates a new PathMatcher with the given working directory.
func NewPathMatcher(workDir string) *PathMatcher {
	return &PathMatcher{workDir: workDir}
}

// Matches returns true if the pattern matches the target path.
// If pattern is relative and a workDir was provided, pattern is resolved
// relative to workDir before comparison.
func (pm *PathMatcher) Matches(pattern, target string) bool {
	pattern = NormalizePath(pattern)
	target = NormalizePath(target)

	// If pattern is relative, make it absolute relative to workDir
	if !filepath.IsAbs(pattern) && pm.workDir != "" {
		pattern = filepath.Join(pm.workDir, pattern)
	}

	return pattern == target
}

// WorkDir returns the working directory for this matcher.
func (pm *PathMatcher) WorkDir() string {
	return pm.workDir
}

// NormalizePath normalizes a path by removing trailing slashes.
// This handles both forward slashes (Unix) and backslashes (Windows).
func NormalizePath(path string) string {
	return strings.TrimRight(path, "\\/")
}

// FindByID searches for a window by ID across all columns using the provided accessors.
// This is a generic helper that works with any window/column/row types.
func FindByID[W WindowInfo, C any, R any](
	row R,
	id int,
	getColumns func(R) []C,
	getWindows func(C) []W,
) W {
	var zero W
	for _, col := range getColumns(row) {
		for _, win := range getWindows(col) {
			if win.ID() == id {
				return win
			}
		}
	}
	return zero
}

// FindByName searches for a window by file name across all columns using the provided accessors.
// The hasColumn function checks if the window has a valid column reference (filtering out
// windows that are being closed or moved).
func FindByName[W WindowInfo, C any, R any](
	row R,
	name string,
	getColumns func(R) []C,
	getWindows func(C) []W,
	hasColumn func(W) bool,
) W {
	var zero W
	name = NormalizePath(name)
	for _, col := range getColumns(row) {
		for _, win := range getWindows(col) {
			winName := NormalizePath(win.Name())
			if winName == name && hasColumn(win) {
				return win
			}
		}
	}
	return zero
}

// FindContainingY finds the window containing the given Y coordinate within a column.
// Returns the index and the window. If y is beyond all windows, returns the length
// of the windows slice as the index and the last window (or nil if column is empty).
func FindContainingY[W any](
	windows []W,
	y int,
	getRectMaxY func(W) int,
) (int, W) {
	var lastWin W
	for i, win := range windows {
		lastWin = win
		if y < getRectMaxY(win) {
			return i, win
		}
	}
	return len(windows), lastWin
}

// FindWindowIndex returns the index of the window in the slice, or -1 if not found.
// Uses pointer comparison for identity.
func FindWindowIndex[W comparable](windows []W, win W) int {
	var zero W
	if win == zero {
		return -1
	}
	for i, w := range windows {
		if w == win {
			return i
		}
	}
	return -1
}

// ForAll calls the provided function for each window in the row.
func ForAll[W any, C any, R any](
	row R,
	getColumns func(R) []C,
	getWindows func(C) []W,
	fn func(W),
) {
	for _, col := range getColumns(row) {
		for _, win := range getWindows(col) {
			fn(win)
		}
	}
}

// Finder provides a convenient wrapper for window lookup operations.
// It stores the accessors once so they don't need to be passed to each lookup call.
type Finder[W WindowInfo, C any, R any] struct {
	row        R
	getColumns func(R) []C
	getWindows func(C) []W
	hasColumn  func(W) bool
}

// NewFinder creates a new Finder with the given row and accessors.
func NewFinder[W WindowInfo, C any, R any](
	row R,
	getColumns func(R) []C,
	getWindows func(C) []W,
	hasColumn func(W) bool,
) *Finder[W, C, R] {
	return &Finder[W, C, R]{
		row:        row,
		getColumns: getColumns,
		getWindows: getWindows,
		hasColumn:  hasColumn,
	}
}

// ByID looks up a window by its unique ID.
func (f *Finder[W, C, R]) ByID(id int) W {
	return FindByID(f.row, id, f.getColumns, f.getWindows)
}

// ByName looks up a window by its file name.
func (f *Finder[W, C, R]) ByName(name string) W {
	return FindByName(f.row, name, f.getColumns, f.getWindows, f.hasColumn)
}

// All calls the provided function for each window.
func (f *Finder[W, C, R]) All(fn func(W)) {
	ForAll(f.row, f.getColumns, f.getWindows, fn)
}

// Row returns the row associated with this Finder.
func (f *Finder[W, C, R]) Row() R {
	return f.row
}
