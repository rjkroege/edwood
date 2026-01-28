// Package command provides command dispatch functionality for edwood.
package command

import (
	"testing"
)

// =============================================================================
// Tests for Window Command Interfaces and Types
// =============================================================================
//
// These tests verify the interfaces and behaviors needed for window commands
// (Newcol, Delcol, Sort, Zerox) that will be extracted from exec.go.
//
// The actual command implementations depend on main package types (Row, Column,
// Window, etc.). These tests verify the command package can support window
// operations through well-defined interfaces.

// =============================================================================
// Column State Tests
// =============================================================================

// TestColumnStateNew tests ColumnState creation.
func TestColumnStateNew(t *testing.T) {
	windows := []WindowInfo{
		{Name: "file_a.go", IsDirty: false, HasRunningCmd: false},
		{Name: "file_b.go", IsDirty: false, HasRunningCmd: false},
	}
	cs := NewColumnState(windows)

	if cs.WindowCount() != 2 {
		t.Errorf("WindowCount() = %d, want 2", cs.WindowCount())
	}
}

// TestColumnStateIsClean tests the IsClean method.
func TestColumnStateIsClean(t *testing.T) {
	tests := []struct {
		name    string
		windows []WindowInfo
		want    bool
	}{
		{
			name:    "empty column is clean",
			windows: []WindowInfo{},
			want:    true,
		},
		{
			name: "all clean windows",
			windows: []WindowInfo{
				{Name: "a.go", IsDirty: false, HasRunningCmd: false},
				{Name: "b.go", IsDirty: false, HasRunningCmd: false},
			},
			want: true,
		},
		{
			name: "one dirty window",
			windows: []WindowInfo{
				{Name: "a.go", IsDirty: false, HasRunningCmd: false},
				{Name: "b.go", IsDirty: true, HasRunningCmd: false},
			},
			want: false,
		},
		{
			name: "all dirty windows",
			windows: []WindowInfo{
				{Name: "a.go", IsDirty: true, HasRunningCmd: false},
				{Name: "b.go", IsDirty: true, HasRunningCmd: false},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cs := NewColumnState(tc.windows)
			if got := cs.IsClean(); got != tc.want {
				t.Errorf("IsClean() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestColumnStateHasRunningCommand tests detection of running commands.
func TestColumnStateHasRunningCommand(t *testing.T) {
	tests := []struct {
		name    string
		windows []WindowInfo
		want    bool
	}{
		{
			name:    "empty column has no running commands",
			windows: []WindowInfo{},
			want:    false,
		},
		{
			name: "no running commands",
			windows: []WindowInfo{
				{Name: "a.go", IsDirty: false, HasRunningCmd: false},
				{Name: "b.go", IsDirty: false, HasRunningCmd: false},
			},
			want: false,
		},
		{
			name: "one running command",
			windows: []WindowInfo{
				{Name: "a.go", IsDirty: false, HasRunningCmd: true},
				{Name: "b.go", IsDirty: false, HasRunningCmd: false},
			},
			want: true,
		},
		{
			name: "multiple running commands",
			windows: []WindowInfo{
				{Name: "a.go", IsDirty: false, HasRunningCmd: true},
				{Name: "b.go", IsDirty: false, HasRunningCmd: true},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cs := NewColumnState(tc.windows)
			if got := cs.HasRunningCommand(); got != tc.want {
				t.Errorf("HasRunningCommand() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestColumnStateRunningCommandWindows tests finding which windows have running commands.
func TestColumnStateRunningCommandWindows(t *testing.T) {
	windows := []WindowInfo{
		{Name: "a.go", IsDirty: false, HasRunningCmd: true},
		{Name: "b.go", IsDirty: false, HasRunningCmd: false},
		{Name: "c.go", IsDirty: false, HasRunningCmd: true},
	}
	cs := NewColumnState(windows)

	running := cs.RunningCommandWindows()
	if len(running) != 2 {
		t.Errorf("expected 2 running command windows, got %d", len(running))
	}
	if running[0] != "a.go" || running[1] != "c.go" {
		t.Errorf("expected [a.go, c.go], got %v", running)
	}
}

// =============================================================================
// Delcol Command Tests
// =============================================================================

// TestDelcolCanDelete tests the CanDelete method for Delcol.
func TestDelcolCanDelete(t *testing.T) {
	tests := []struct {
		name        string
		windows     []WindowInfo
		wantDelete  bool
		wantReason  string
	}{
		{
			name:        "empty column can be deleted",
			windows:     []WindowInfo{},
			wantDelete:  true,
			wantReason:  "",
		},
		{
			name: "clean column with no commands can be deleted",
			windows: []WindowInfo{
				{Name: "a.go", IsDirty: false, HasRunningCmd: false},
				{Name: "b.go", IsDirty: false, HasRunningCmd: false},
			},
			wantDelete:  true,
			wantReason:  "",
		},
		{
			name: "dirty column cannot be deleted",
			windows: []WindowInfo{
				{Name: "a.go", IsDirty: true, HasRunningCmd: false},
			},
			wantDelete:  false,
			wantReason:  "dirty",
		},
		{
			name: "column with running command cannot be deleted",
			windows: []WindowInfo{
				{Name: "a.go", IsDirty: false, HasRunningCmd: true},
			},
			wantDelete:  false,
			wantReason:  "running",
		},
		{
			name: "dirty column with running command - running takes precedence",
			windows: []WindowInfo{
				{Name: "a.go", IsDirty: true, HasRunningCmd: true},
			},
			wantDelete:  false,
			wantReason:  "running",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cs := NewColumnState(tc.windows)
			canDelete, reason := cs.CanDelete()
			if canDelete != tc.wantDelete {
				t.Errorf("CanDelete() = %v, want %v", canDelete, tc.wantDelete)
			}
			if reason != tc.wantReason {
				t.Errorf("CanDelete() reason = %q, want %q", reason, tc.wantReason)
			}
		})
	}
}

// =============================================================================
// Sort Command Tests
// =============================================================================

// TestSortWindowNames tests sorting window names.
func TestSortWindowNames(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "already sorted",
			input: []string{"a.go", "b.go", "c.go"},
			want:  []string{"a.go", "b.go", "c.go"},
		},
		{
			name:  "reverse sorted",
			input: []string{"c.go", "b.go", "a.go"},
			want:  []string{"a.go", "b.go", "c.go"},
		},
		{
			name:  "mixed order",
			input: []string{"main.go", "api.go", "util.go", "db.go"},
			want:  []string{"api.go", "db.go", "main.go", "util.go"},
		},
		{
			name:  "empty list",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "single window",
			input: []string{"only.go"},
			want:  []string{"only.go"},
		},
		{
			name:  "with paths",
			input: []string{"/home/user/z.go", "/home/user/a.go"},
			want:  []string{"/home/user/a.go", "/home/user/z.go"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SortWindowNames(tc.input)
			if len(got) != len(tc.want) {
				t.Errorf("SortWindowNames() returned %d items, want %d", len(got), len(tc.want))
				return
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("SortWindowNames()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestSortOperationNeedsColumn tests that Sort requires a column.
func TestSortOperationNeedsColumn(t *testing.T) {
	op := NewSortOperation()

	if !op.RequiresColumn() {
		t.Error("Sort should require a column")
	}
}

// =============================================================================
// Zerox Command Tests
// =============================================================================

// TestZeroxCanClone tests when Zerox can clone a window.
func TestZeroxCanClone(t *testing.T) {
	tests := []struct {
		name      string
		isDir     bool
		hasWindow bool
		wantClone bool
		wantError string
	}{
		{
			name:      "regular file can be cloned",
			isDir:     false,
			hasWindow: true,
			wantClone: true,
			wantError: "",
		},
		{
			name:      "directory cannot be cloned",
			isDir:     true,
			hasWindow: true,
			wantClone: false,
			wantError: "directory",
		},
		{
			name:      "no window to clone",
			isDir:     false,
			hasWindow: false,
			wantClone: false,
			wantError: "no window",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			zs := NewZeroxState(tc.hasWindow, tc.isDir, "test.go")
			canClone, errReason := zs.CanClone()
			if canClone != tc.wantClone {
				t.Errorf("CanClone() = %v, want %v", canClone, tc.wantClone)
			}
			if errReason != tc.wantError {
				t.Errorf("CanClone() error = %q, want %q", errReason, tc.wantError)
			}
		})
	}
}

// TestZeroxStateGetters tests ZeroxState getter methods.
func TestZeroxStateGetters(t *testing.T) {
	zs := NewZeroxState(true, false, "/path/to/file.go")

	if !zs.HasWindow() {
		t.Error("HasWindow() = false, want true")
	}
	if zs.IsDirectory() {
		t.Error("IsDirectory() = true, want false")
	}
	if zs.FileName() != "/path/to/file.go" {
		t.Errorf("FileName() = %q, want %q", zs.FileName(), "/path/to/file.go")
	}
}

// =============================================================================
// Newcol Command Tests
// =============================================================================

// TestNewcolOperation tests the Newcol operation.
func TestNewcolOperation(t *testing.T) {
	op := NewNewcolOperation()

	// Newcol always succeeds if there's a row
	if !op.RequiresRow() {
		t.Error("Newcol should require a row")
	}

	// After creating a column, a window should be added
	if !op.AddsWindowToColumn() {
		t.Error("Newcol should add a window to the new column")
	}
}

// =============================================================================
// Window Info Tests
// =============================================================================

// TestWindowInfoGetters tests WindowInfo getter methods.
func TestWindowInfoGetters(t *testing.T) {
	wi := WindowInfo{
		Name:          "test.go",
		IsDirty:       true,
		HasRunningCmd: true,
	}

	if wi.Name != "test.go" {
		t.Errorf("Name = %q, want %q", wi.Name, "test.go")
	}
	if !wi.IsDirty {
		t.Error("IsDirty = false, want true")
	}
	if !wi.HasRunningCmd {
		t.Error("HasRunningCmd = false, want true")
	}
}

// =============================================================================
// Window Command Entry Tests
// =============================================================================

// TestWindowCommandEntries tests that window command entries have correct properties.
func TestWindowCommandEntries(t *testing.T) {
	d := NewDispatcher()
	reg := NewWindowCommandRegistry()
	reg.RegisterWindowCommands(d)

	// Verify Newcol is not undoable
	newcol := d.LookupCommand("Newcol")
	if newcol == nil {
		t.Fatal("Newcol command not found")
	}
	if newcol.Mark() {
		t.Error("Newcol should not be undoable (mark=false)")
	}

	// Verify Delcol is not undoable
	delcol := d.LookupCommand("Delcol")
	if delcol == nil {
		t.Fatal("Delcol command not found")
	}
	if delcol.Mark() {
		t.Error("Delcol should not be undoable (mark=false)")
	}

	// Verify Sort is not undoable
	sortCmd := d.LookupCommand("Sort")
	if sortCmd == nil {
		t.Fatal("Sort command not found")
	}
	if sortCmd.Mark() {
		t.Error("Sort should not be undoable (mark=false)")
	}

	// Verify Zerox is not undoable
	zerox := d.LookupCommand("Zerox")
	if zerox == nil {
		t.Fatal("Zerox command not found")
	}
	if zerox.Mark() {
		t.Error("Zerox should not be undoable (mark=false)")
	}
}

// =============================================================================
// Window Command Dispatch Tests
// =============================================================================

// TestWindowCommandDispatch tests that window commands can be looked up correctly.
func TestWindowCommandDispatch(t *testing.T) {
	d := NewDispatcher()
	reg := NewWindowCommandRegistry()
	reg.RegisterWindowCommands(d)

	tests := []struct {
		input    string
		wantName string
		found    bool
	}{
		{"Newcol", "Newcol", true},
		{"Delcol", "Delcol", true},
		{"Sort", "Sort", true},
		{"Zerox", "Zerox", true},
		{"newcol", "", false}, // Case sensitive
		{"SORT", "", false},   // Case sensitive
		{"Clone", "", false},  // Not registered (acme uses Zerox)
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			cmd := d.LookupCommand(tc.input)
			if tc.found {
				if cmd == nil {
					t.Errorf("expected to find command for %q", tc.input)
					return
				}
				if cmd.Name() != tc.wantName {
					t.Errorf("expected name %q, got %q", tc.wantName, cmd.Name())
				}
			} else {
				if cmd != nil {
					t.Errorf("expected nil for %q, got %v", tc.input, cmd)
				}
			}
		})
	}
}

// TestWindowCommandRegistryIntegration tests the full registration flow.
func TestWindowCommandRegistryIntegration(t *testing.T) {
	d := NewDispatcher()
	reg := NewWindowCommandRegistry()
	reg.RegisterWindowCommands(d)

	// Should have exactly 4 window commands
	cmds := d.Commands()
	if len(cmds) != 4 {
		t.Errorf("expected 4 commands, got %d", len(cmds))
	}

	// Verify all expected commands are present
	expected := map[string]bool{
		"Newcol": true,
		"Delcol": true,
		"Sort":   true,
		"Zerox":  true,
	}

	for _, cmd := range cmds {
		if !expected[cmd.Name()] {
			t.Errorf("unexpected command: %s", cmd.Name())
		}
		delete(expected, cmd.Name())
	}

	if len(expected) > 0 {
		t.Errorf("missing commands: %v", expected)
	}
}

// =============================================================================
// Combined Command Registry Tests
// =============================================================================

// TestAllWindowAndFileCommandsRegistered tests coexistence with file commands.
func TestAllWindowAndFileCommandsRegistered(t *testing.T) {
	d := NewDispatcher()

	fileReg := NewFileCommandRegistry()
	fileReg.RegisterFileCommands(d)

	windowReg := NewWindowCommandRegistry()
	windowReg.RegisterWindowCommands(d)

	// Should have 10 total commands (6 file + 4 window)
	cmds := d.Commands()
	if len(cmds) != 10 {
		t.Errorf("expected 10 commands, got %d", len(cmds))
	}

	// Verify some from each category
	if d.LookupCommand("Del") == nil {
		t.Error("Del (file command) should be registered")
	}
	if d.LookupCommand("Newcol") == nil {
		t.Error("Newcol (window command) should be registered")
	}
	if d.LookupCommand("Sort") == nil {
		t.Error("Sort (window command) should be registered")
	}
}

// TestAllCommandTypesRegistered tests that file, edit, and window commands coexist.
func TestAllCommandTypesRegistered(t *testing.T) {
	d := NewDispatcher()

	fileReg := NewFileCommandRegistry()
	fileReg.RegisterFileCommands(d)

	editReg := NewEditCommandRegistry()
	editReg.RegisterEditCommands(d)

	windowReg := NewWindowCommandRegistry()
	windowReg.RegisterWindowCommands(d)

	// Should have 15 total commands (6 file + 5 edit + 4 window)
	cmds := d.Commands()
	if len(cmds) != 15 {
		t.Errorf("expected 15 commands, got %d", len(cmds))
	}

	// Spot check each category
	if d.LookupCommand("Put") == nil {
		t.Error("Put (file command) should be registered")
	}
	if d.LookupCommand("Cut") == nil {
		t.Error("Cut (edit command) should be registered")
	}
	if d.LookupCommand("Zerox") == nil {
		t.Error("Zerox (window command) should be registered")
	}
}
