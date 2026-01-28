// Package command provides command dispatch functionality for edwood.
package command

import (
	"testing"
)

// =============================================================================
// Tests for File Command Interfaces and Types
// =============================================================================
//
// These tests verify the interfaces and behaviors needed for file commands
// (Get, Put, Putall, New, Del) that will be extracted from exec.go.
//
// The actual command implementations depend on main package types (Text, Window,
// File, etc.). These tests verify the command package can support file operations
// through well-defined interfaces.

// =============================================================================
// Del Command Tests
// =============================================================================

// TestFileInfoForDel tests FileInfo behavior for the Del command.
// Del closes a window, optionally checking for unsaved changes.
func TestFileInfoForDel(t *testing.T) {
	tests := []struct {
		name        string
		fileInfo    *FileInfo
		forceClose  bool // flag1 in del() - true means "Delete" (force), false means "Del"
		expectClose bool
	}{
		// Del (not forced) should only close clean files
		{
			name:        "Del clean file",
			fileInfo:    NewFileInfo("test.txt", false, false, false),
			forceClose:  false,
			expectClose: true,
		},
		{
			name:        "Del dirty file without force",
			fileInfo:    NewFileInfo("test.txt", false, true, false),
			forceClose:  false,
			expectClose: false, // Won't close dirty file without force
		},
		// Delete (forced) should close even dirty files
		{
			name:        "Delete dirty file with force",
			fileInfo:    NewFileInfo("test.txt", false, true, false),
			forceClose:  true,
			expectClose: true,
		},
		// Files with multiple observers can always be closed
		{
			name:        "Del file with multiple observers",
			fileInfo:    NewFileInfo("test.txt", false, true, true),
			forceClose:  false,
			expectClose: true, // Multiple observers means closing is safe
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use the CanDelete method
			shouldClose := tc.fileInfo.CanDelete(tc.forceClose)

			if shouldClose != tc.expectClose {
				t.Errorf("expected close=%v, got %v", tc.expectClose, shouldClose)
			}
		})
	}
}

// TestCanDelete tests the CanDelete method directly.
func TestCanDelete(t *testing.T) {
	tests := []struct {
		name       string
		dirty      bool
		observers  bool
		force      bool
		wantDelete bool
	}{
		{"clean file, no force", false, false, false, true},
		{"dirty file, no force", true, false, false, false},
		{"dirty file, with force", true, false, true, true},
		{"dirty file, multiple observers", true, true, false, true},
		{"clean file, with force", false, false, true, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fi := NewFileInfo("test.txt", false, tc.dirty, tc.observers)
			if got := fi.CanDelete(tc.force); got != tc.wantDelete {
				t.Errorf("CanDelete(%v) = %v, want %v", tc.force, got, tc.wantDelete)
			}
		})
	}
}

// =============================================================================
// Get Command Tests
// =============================================================================

// TestFileInfoForGet tests FileInfo behavior for the Get command.
// Get loads/reloads a file into a window.
func TestFileInfoForGet(t *testing.T) {
	tests := []struct {
		name        string
		fileInfo    *FileInfo
		newName     string
		newIsDir    bool
		expectLoad  bool
		expectError string
	}{
		{
			name:        "Get file into empty window",
			fileInfo:    NewFileInfo("", false, false, false),
			newName:     "test.txt",
			newIsDir:    false,
			expectLoad:  true,
			expectError: "",
		},
		{
			name:        "Get same file (reload)",
			fileInfo:    NewFileInfo("test.txt", false, false, false),
			newName:     "test.txt",
			newIsDir:    false,
			expectLoad:  true,
			expectError: "",
		},
		{
			name:        "Get different file into dirty window",
			fileInfo:    NewFileInfo("test.txt", false, true, false),
			newName:     "other.txt",
			newIsDir:    false,
			expectLoad:  false, // Won't load if window is dirty
			expectError: "dirty",
		},
		{
			name:        "Get directory with multiple observers",
			fileInfo:    NewFileInfo("test.txt", false, false, true),
			newName:     "/some/dir",
			newIsDir:    true,
			expectLoad:  false,
			expectError: "directory", // Can't read directory with multiple windows
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use the CanGet method
			canLoad, errorReason := tc.fileInfo.CanGet(tc.newName, tc.newIsDir)

			if canLoad != tc.expectLoad {
				t.Errorf("expected load=%v, got %v", tc.expectLoad, canLoad)
			}
			if tc.expectError != "" && errorReason != tc.expectError {
				t.Errorf("expected error containing %q, got %q", tc.expectError, errorReason)
			}
		})
	}
}

// TestCanGet tests the CanGet method directly.
func TestCanGet(t *testing.T) {
	tests := []struct {
		name        string
		currentName string
		dirty       bool
		observers   bool
		newName     string
		isDir       bool
		wantLoad    bool
		wantError   string
	}{
		{"clean file, same name", "a.txt", false, false, "a.txt", false, true, ""},
		{"clean file, different name", "a.txt", false, false, "b.txt", false, true, ""},
		{"dirty file, same name (reload)", "a.txt", true, false, "a.txt", false, true, ""},
		{"dirty file, different name", "a.txt", true, false, "b.txt", false, false, "dirty"},
		{"directory with observers", "a.txt", false, true, "/dir", true, false, "directory"},
		{"directory without observers", "a.txt", false, false, "/dir", true, true, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fi := NewFileInfo(tc.currentName, false, tc.dirty, tc.observers)
			canLoad, errReason := fi.CanGet(tc.newName, tc.isDir)
			if canLoad != tc.wantLoad {
				t.Errorf("CanGet() canLoad = %v, want %v", canLoad, tc.wantLoad)
			}
			if errReason != tc.wantError {
				t.Errorf("CanGet() errReason = %q, want %q", errReason, tc.wantError)
			}
		})
	}
}

// =============================================================================
// Put Command Tests
// =============================================================================

// TestFileInfoForPut tests FileInfo behavior for the Put command.
// Put saves a file to disk.
func TestFileInfoForPut(t *testing.T) {
	tests := []struct {
		name       string
		fileInfo   *FileInfo
		targetName string
		expectSave bool
		expectMsg  string
	}{
		{
			name:       "Put normal file",
			fileInfo:   NewFileInfo("test.txt", false, true, false),
			targetName: "test.txt",
			expectSave: true,
			expectMsg:  "",
		},
		{
			name:       "Put directory (should fail)",
			fileInfo:   NewFileInfo("/some/dir", true, false, false),
			targetName: "/some/dir",
			expectSave: false,
			expectMsg:  "directory",
		},
		{
			name:       "Put to different name",
			fileInfo:   NewFileInfo("test.txt", false, true, false),
			targetName: "other.txt",
			expectSave: true,
			expectMsg:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use the CanPut method
			canSave, msg := tc.fileInfo.CanPut()

			if canSave != tc.expectSave {
				t.Errorf("expected save=%v, got %v", tc.expectSave, canSave)
			}
			if tc.expectMsg != "" && msg != tc.expectMsg {
				t.Errorf("expected message %q, got %q", tc.expectMsg, msg)
			}
		})
	}
}

// TestCanPut tests the CanPut method directly.
func TestCanPut(t *testing.T) {
	tests := []struct {
		name      string
		isDir     bool
		wantSave  bool
		wantError string
	}{
		{"regular file", false, true, ""},
		{"directory", true, false, "directory"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fi := NewFileInfo("test", tc.isDir, false, false)
			canSave, errReason := fi.CanPut()
			if canSave != tc.wantSave {
				t.Errorf("CanPut() canSave = %v, want %v", canSave, tc.wantSave)
			}
			if errReason != tc.wantError {
				t.Errorf("CanPut() errReason = %q, want %q", errReason, tc.wantError)
			}
		})
	}
}

// =============================================================================
// Putall Command Tests
// =============================================================================

// TestPutallCandidates tests which files Putall would save.
func TestPutallCandidates(t *testing.T) {
	candidates := []*PutallCandidate{
		{Name: "file1.txt", Dirty: true, HasEvent: false, FileExists: true},
		{Name: "file2.txt", Dirty: false, HasEvent: false, FileExists: true},
		{Name: "file3.txt", Dirty: true, HasEvent: true, FileExists: true},   // Running command
		{Name: "file4.txt", Dirty: true, HasEvent: false, FileExists: false}, // New file
	}

	var saved []string
	var skipped []string

	for _, c := range candidates {
		if c.ShouldSave() {
			saved = append(saved, c.Name)
		} else {
			reason := ""
			if c.HasEvent {
				reason = " (running command)"
			} else if !c.Dirty {
				reason = " (not dirty)"
			} else if !c.FileExists {
				reason = " (file doesn't exist)"
			}
			skipped = append(skipped, c.Name+reason)
		}
	}

	if len(saved) != 1 || saved[0] != "file1.txt" {
		t.Errorf("expected to save only [file1.txt], got %v", saved)
	}
	if len(skipped) != 3 {
		t.Errorf("expected 3 skipped files, got %d: %v", len(skipped), skipped)
	}
}

// TestPutallCandidateShouldSave tests the ShouldSave method directly.
func TestPutallCandidateShouldSave(t *testing.T) {
	tests := []struct {
		name       string
		dirty      bool
		hasEvent   bool
		fileExists bool
		wantSave   bool
	}{
		{"dirty, no event, exists", true, false, true, true},
		{"not dirty", false, false, true, false},
		{"has event", true, true, true, false},
		{"file doesn't exist", true, false, false, false},
		{"all blockers", true, true, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &PutallCandidate{
				Name:       "test.txt",
				Dirty:      tc.dirty,
				HasEvent:   tc.hasEvent,
				FileExists: tc.fileExists,
			}
			if got := c.ShouldSave(); got != tc.wantSave {
				t.Errorf("ShouldSave() = %v, want %v", got, tc.wantSave)
			}
		})
	}
}

// =============================================================================
// New Command Tests
// =============================================================================

// TestNewCommandArgParsing tests argument parsing for the New command.
func TestNewCommandArgParsing(t *testing.T) {
	tests := []struct {
		name      string
		arg       string
		wantFiles []string
	}{
		{
			name:      "empty arg creates empty window",
			arg:       "",
			wantFiles: nil, // Empty window, no files
		},
		{
			name:      "single file",
			arg:       "test.txt",
			wantFiles: []string{"test.txt"},
		},
		{
			name:      "multiple files",
			arg:       "file1.txt file2.txt file3.txt",
			wantFiles: []string{"file1.txt", "file2.txt", "file3.txt"},
		},
		{
			name:      "files with extra whitespace",
			arg:       "  file1.txt   file2.txt  ",
			wantFiles: []string{"file1.txt", "file2.txt"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use ParseNewArgs from file_cmds.go
			files := ParseNewArgs(tc.arg)

			if tc.wantFiles == nil {
				if len(files) != 0 {
					t.Errorf("expected empty/nil files, got %v", files)
				}
				return
			}

			if len(files) != len(tc.wantFiles) {
				t.Errorf("expected %d files, got %d: %v", len(tc.wantFiles), len(files), files)
				return
			}

			for i, want := range tc.wantFiles {
				if files[i] != want {
					t.Errorf("file[%d]: expected %q, got %q", i, want, files[i])
				}
			}
		})
	}
}

// TestParseNewArgs tests ParseNewArgs edge cases.
func TestParseNewArgs(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"   ", nil},
		{"\t\n", nil},
		{"a", []string{"a"}},
		{"a b", []string{"a", "b"}},
		{"a  b", []string{"a", "b"}},
		{" a b ", []string{"a", "b"}},
		{"a\tb\nc", []string{"a", "b", "c"}},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := ParseNewArgs(tc.input)
			if tc.want == nil {
				if got != nil {
					t.Errorf("ParseNewArgs(%q) = %v, want nil", tc.input, got)
				}
				return
			}
			if len(got) != len(tc.want) {
				t.Errorf("ParseNewArgs(%q) = %v, want %v", tc.input, got, tc.want)
				return
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("ParseNewArgs(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// =============================================================================
// File Command Entry Tests
// =============================================================================

// TestFileCommandEntries tests that file command entries have correct properties.
func TestFileCommandEntries(t *testing.T) {
	d := NewDispatcher()
	reg := NewFileCommandRegistry()
	reg.RegisterFileCommands(d)

	// Verify Del is not undoable
	del := d.LookupCommand("Del")
	if del == nil {
		t.Fatal("Del command not found")
	}
	if del.Mark() {
		t.Error("Del should not be undoable (mark=false)")
	}

	// Verify Delete has flag1=true (force)
	delete := d.LookupCommand("Delete")
	if delete == nil {
		t.Fatal("Delete command not found")
	}
	if !delete.Flag1() {
		t.Error("Delete should have flag1=true (force close)")
	}

	// Verify Get is registered
	get := d.LookupCommand("Get")
	if get == nil {
		t.Fatal("Get command not found")
	}

	// Verify New is registered
	newCmd := d.LookupCommand("New")
	if newCmd == nil {
		t.Fatal("New command not found")
	}

	// Verify Put is not undoable
	put := d.LookupCommand("Put")
	if put == nil {
		t.Fatal("Put command not found")
	}
	if put.Mark() {
		t.Error("Put should not be undoable (mark=false)")
	}

	// Verify Putall is registered
	putall := d.LookupCommand("Putall")
	if putall == nil {
		t.Fatal("Putall command not found")
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

// TestFileCommandDispatch tests that file commands can be looked up correctly.
func TestFileCommandDispatch(t *testing.T) {
	d := NewDispatcher()
	reg := NewFileCommandRegistry()
	reg.RegisterFileCommands(d)

	tests := []struct {
		input    string
		wantName string
		found    bool
	}{
		{"Del", "Del", true},
		{"Delete", "Delete", true},
		{"Get", "Get", true},
		{"Get filename.txt", "Get", true},
		{"New", "New", true},
		{"New file1.txt file2.txt", "New", true},
		{"Put", "Put", true},
		{"Put newname.txt", "Put", true},
		{"Putall", "Putall", true},
		{"del", "", false},  // Case sensitive
		{"PUT", "", false},  // Case sensitive
		{"Save", "", false}, // Not registered
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

// TestGetNameHelper tests the getname helper function behavior.
// getname extracts the filename from command arguments.
func TestGetNameHelper(t *testing.T) {
	tests := []struct {
		name        string
		currentName string
		arg         string
		putFlag     bool // true for Put, false for Get
		wantResult  string
	}{
		{
			name:        "Get with no arg uses current name",
			currentName: "test.txt",
			arg:         "",
			putFlag:     false,
			wantResult:  "test.txt",
		},
		{
			name:        "Get with arg uses arg",
			currentName: "test.txt",
			arg:         "other.txt",
			putFlag:     false,
			wantResult:  "other.txt",
		},
		{
			name:        "Put with no arg uses current name",
			currentName: "test.txt",
			arg:         "",
			putFlag:     true,
			wantResult:  "test.txt",
		},
		{
			name:        "Put with arg uses arg",
			currentName: "test.txt",
			arg:         "backup.txt",
			putFlag:     true,
			wantResult:  "backup.txt",
		},
		{
			name:        "Empty name with no arg",
			currentName: "",
			arg:         "",
			putFlag:     false,
			wantResult:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var result string
			if tc.putFlag {
				result = ResolvePutName(tc.currentName, tc.arg)
			} else {
				result = ResolveGetName(tc.currentName, tc.arg)
			}

			if result != tc.wantResult {
				t.Errorf("expected %q, got %q", tc.wantResult, result)
			}
		})
	}
}

// TestResolveGetName tests ResolveGetName directly.
func TestResolveGetName(t *testing.T) {
	tests := []struct {
		current string
		arg     string
		want    string
	}{
		{"current.txt", "", "current.txt"},
		{"current.txt", "new.txt", "new.txt"},
		{"", "new.txt", "new.txt"},
		{"", "", ""},
	}

	for _, tc := range tests {
		if got := ResolveGetName(tc.current, tc.arg); got != tc.want {
			t.Errorf("ResolveGetName(%q, %q) = %q, want %q", tc.current, tc.arg, got, tc.want)
		}
	}
}

// TestResolvePutName tests ResolvePutName directly.
func TestResolvePutName(t *testing.T) {
	tests := []struct {
		current string
		arg     string
		want    string
	}{
		{"current.txt", "", "current.txt"},
		{"current.txt", "backup.txt", "backup.txt"},
		{"", "new.txt", "new.txt"},
		{"", "", ""},
	}

	for _, tc := range tests {
		if got := ResolvePutName(tc.current, tc.arg); got != tc.want {
			t.Errorf("ResolvePutName(%q, %q) = %q, want %q", tc.current, tc.arg, got, tc.want)
		}
	}
}

// TestFileInfoGetters tests the FileInfo getter methods.
func TestFileInfoGetters(t *testing.T) {
	fi := NewFileInfo("test.txt", true, true, true)

	if fi.Name() != "test.txt" {
		t.Errorf("Name() = %q, want %q", fi.Name(), "test.txt")
	}
	if !fi.IsDir() {
		t.Error("IsDir() = false, want true")
	}
	if !fi.IsDirty() {
		t.Error("IsDirty() = false, want true")
	}
	if !fi.HasMultipleObservers() {
		t.Error("HasMultipleObservers() = false, want true")
	}
}

// TestFileCommandRegistryIntegration tests the full registration flow.
func TestFileCommandRegistryIntegration(t *testing.T) {
	d := NewDispatcher()
	reg := NewFileCommandRegistry()
	reg.RegisterFileCommands(d)

	// Should have exactly 6 file commands
	cmds := d.Commands()
	if len(cmds) != 6 {
		t.Errorf("expected 6 commands, got %d", len(cmds))
	}

	// Verify all expected commands are present
	expected := map[string]bool{
		"Del":    true,
		"Delete": true,
		"Get":    true,
		"New":    true,
		"Put":    true,
		"Putall": true,
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
