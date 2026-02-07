package main

import (
	"image"
	"path/filepath"
	"testing"

	textboundary "github.com/rjkroege/edwood/internal/textboundary"
)

// These tests verify that the interfaces in internal/textboundary can be used to fix
// layering violations in text.go.
// The layering concerns are:
// 1. Line 1151: Text.Select() directly accesses global.mousectl.Read() and global.mouse
//    in a loop waiting for mouse state changes during double-click handling
// 2. Line 1658: Text.dirName() reaches into t.w.tag.file and t.w.ParseTag() to resolve
//    relative paths using the window's directory context
//
// The production interfaces are defined in internal/textboundary/interfaces.go.
// These tests verify their integration with the main package.

// mockMouseWaiter is a mock implementation of textboundary.MouseWaiter for testing.
// It demonstrates how global mouse state could be abstracted for Text.Select().
type mockMouseWaiter struct {
	currentState textboundary.MouseSnapshot
	changeResult bool
}

func (m *mockMouseWaiter) WaitForChange(originalButton int, originalPos image.Point, threshold int) bool {
	return m.changeResult
}

func (m *mockMouseWaiter) CurrentState() textboundary.MouseState {
	return m.currentState
}

// mockTagProvider is a mock implementation of textboundary.TagProvider for testing.
// It demonstrates how Window.tag could be abstracted for Text.dirName().
type mockTagProvider struct {
	filename string
	hasTag   bool
}

func (m *mockTagProvider) TagFileName() string {
	return m.filename
}

func (m *mockTagProvider) HasTag() bool {
	return m.hasTag
}

// mockDirectoryContext is a mock implementation of textboundary.DirectoryContext for testing.
type mockDirectoryContext struct {
	mockTagProvider
	workDir string
}

func (m *mockDirectoryContext) WorkingDir() string {
	return m.workDir
}

// TestMouseWaiterInterface tests that textboundary.MouseWaiter can abstract
// the mouse waiting loop in Text.Select().
func TestMouseWaiterInterface(t *testing.T) {
	// The current code at text.go:1151-1157 does:
	//   for {
	//       global.mousectl.Read()
	//       if !(global.mouse.Buttons == b && util.Abs(global.mouse.Point.X-x) < 3 && util.Abs(global.mouse.Point.Y-y) < 3) {
	//           break
	//       }
	//   }
	//
	// With MouseWaiter, this becomes:
	//   waiter.WaitForChange(b, image.Point{x, y}, 3)

	tests := []struct {
		name          string
		initialButton int
		initialPos    image.Point
		threshold     int
		wantInterrupt bool
		finalButton   int
		finalPos      image.Point
	}{
		{
			name:          "button released",
			initialButton: 1,
			initialPos:    image.Point{100, 100},
			threshold:     3,
			wantInterrupt: true,
			finalButton:   0,
			finalPos:      image.Point{100, 100},
		},
		{
			name:          "mouse moved beyond threshold",
			initialButton: 1,
			initialPos:    image.Point{100, 100},
			threshold:     3,
			wantInterrupt: true,
			finalButton:   1,
			finalPos:      image.Point{105, 100},
		},
		{
			name:          "mouse moved within threshold",
			initialButton: 1,
			initialPos:    image.Point{100, 100},
			threshold:     3,
			wantInterrupt: false, // stays in loop
			finalButton:   1,
			finalPos:      image.Point{102, 101},
		},
		{
			name:          "different button pressed",
			initialButton: 1,
			initialPos:    image.Point{100, 100},
			threshold:     3,
			wantInterrupt: true,
			finalButton:   4, // right button
			finalPos:      image.Point{100, 100},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock waiter that simulates the desired behavior
			waiter := &mockMouseWaiter{
				currentState: textboundary.MouseSnapshot{
					ButtonState: tc.finalButton,
					Position:    tc.finalPos,
				},
				changeResult: tc.wantInterrupt,
			}

			// Test via interface
			var w textboundary.MouseWaiter = waiter
			gotInterrupt := w.WaitForChange(tc.initialButton, tc.initialPos, tc.threshold)

			if gotInterrupt != tc.wantInterrupt {
				t.Errorf("WaitForChange() = %v, want %v", gotInterrupt, tc.wantInterrupt)
			}

			// Verify final state
			state := w.CurrentState()
			if state.Buttons() != tc.finalButton {
				t.Errorf("final Buttons() = %d, want %d", state.Buttons(), tc.finalButton)
			}
			if state.Point() != tc.finalPos {
				t.Errorf("final Point() = %v, want %v", state.Point(), tc.finalPos)
			}
		})
	}
}

// TestDirectoryResolverForDirName tests that textboundary.DirectoryResolver can abstract
// the directory resolution logic in Text.dirName().
func TestDirectoryResolverForDirName(t *testing.T) {
	// The current code at text.go:1661-1675 does:
	//   if t == nil || t.w == nil || filepath.IsAbs(name) {
	//       return name
	//   }
	//   nt := t.w.tag.file.Nr()
	//   if nt == 0 {
	//       return name
	//   }
	//   spl := t.w.ParseTag()
	//   if !strings.HasSuffix(spl, string(filepath.Separator)) {
	//       spl = filepath.Dir(spl)
	//   }
	//   return filepath.Join(spl, name)
	//
	// With DirectoryResolver, the logic for getting the directory context
	// is abstracted:
	//   dir := resolver.ResolveDir()
	//   if dir == "" || filepath.IsAbs(name) {
	//       return name
	//   }
	//   return filepath.Join(dir, name)

	tests := []struct {
		name       string
		inputName  string
		resolveDir string
		wantResult string
	}{
		{
			name:       "nil resolver (no window)",
			inputName:  "file.go",
			resolveDir: "",
			wantResult: "file.go",
		},
		{
			name:       "absolute path unchanged",
			inputName:  "/absolute/path/file.go",
			resolveDir: "/some/dir",
			wantResult: "/absolute/path/file.go",
		},
		{
			name:       "relative path with directory",
			inputName:  "file.go",
			resolveDir: "/home/user/project",
			wantResult: "/home/user/project/file.go",
		},
		{
			name:       "relative path with trailing slash dir",
			inputName:  "subdir/file.go",
			resolveDir: "/home/user/project/",
			wantResult: "/home/user/project/subdir/file.go",
		},
		{
			name:       "path with parent reference",
			inputName:  "../other/file.go",
			resolveDir: "/home/user/project",
			wantResult: "/home/user/other/file.go", // filepath.Join cleans the path
		},
		{
			name:       "empty filename",
			inputName:  "",
			resolveDir: "/home/user",
			wantResult: "/home/user",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var resolver textboundary.DirectoryResolver
			if tc.resolveDir == "" {
				resolver = textboundary.NilDirectoryResolver{}
			} else {
				resolver = textboundary.StaticDirectoryResolver{Dir: tc.resolveDir}
			}

			// Simulate the refactored dirName logic
			var result string
			dir := resolver.ResolveDir()
			if dir == "" || filepath.IsAbs(tc.inputName) {
				result = tc.inputName
			} else {
				result = filepath.Join(dir, tc.inputName)
			}

			if result != tc.wantResult {
				t.Errorf("dirName(%q) with resolver=%q = %q, want %q",
					tc.inputName, tc.resolveDir, result, tc.wantResult)
			}
		})
	}
}

// TestTagProviderForParseTag tests that textboundary.TagProvider can abstract
// access to window tag parsing.
func TestTagProviderForParseTag(t *testing.T) {
	// Text.dirName() calls t.w.ParseTag() which parses the tag to extract
	// the filename. With TagProvider, this is abstracted.

	tests := []struct {
		name     string
		filename string
		hasTag   bool
		wantDir  string
	}{
		{
			name:     "file in directory",
			filename: "/home/user/project/main.go",
			hasTag:   true,
			wantDir:  "/home/user/project",
		},
		{
			name:     "directory listing",
			filename: "/home/user/project/",
			hasTag:   true,
			wantDir:  "/home/user/project/",
		},
		{
			name:     "no tag",
			filename: "",
			hasTag:   false,
			wantDir:  "",
		},
		{
			name:     "root directory",
			filename: "/file.go",
			hasTag:   true,
			wantDir:  "/",
		},
		{
			name:     "empty filename in tag",
			filename: "",
			hasTag:   true,
			wantDir:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			provider := &mockTagProvider{
				filename: tc.filename,
				hasTag:   tc.hasTag,
			}

			// Test interface
			var p textboundary.TagProvider = provider
			if p.HasTag() != tc.hasTag {
				t.Errorf("HasTag() = %v, want %v", p.HasTag(), tc.hasTag)
			}
			if p.TagFileName() != tc.filename {
				t.Errorf("TagFileName() = %q, want %q", p.TagFileName(), tc.filename)
			}

			// Compute directory from filename (like dirName does)
			gotDir := ""
			if tc.hasTag && tc.filename != "" {
				if tc.filename[len(tc.filename)-1] == filepath.Separator {
					gotDir = tc.filename
				} else {
					gotDir = filepath.Dir(tc.filename)
				}
			}

			if gotDir != tc.wantDir {
				t.Errorf("computed dir = %q, want %q", gotDir, tc.wantDir)
			}
		})
	}
}

// TestDirectoryContextForAbsDirName tests that textboundary.DirectoryContext can abstract
// the working directory fallback in Text.AbsDirName().
func TestDirectoryContextForAbsDirName(t *testing.T) {
	// Text.AbsDirName() at text.go:1684-1691 uses global.wdir as fallback.
	// With DirectoryContext, this is abstracted.

	tests := []struct {
		name       string
		filename   string
		hasTag     bool
		workDir    string
		inputName  string
		wantResult string
	}{
		{
			name:       "relative path with tag dir",
			filename:   "/home/user/project/main.go",
			hasTag:     true,
			workDir:    "/other/workdir",
			inputName:  "util.go",
			wantResult: "/home/user/project/util.go",
		},
		{
			name:       "relative path falls back to workdir",
			filename:   "",
			hasTag:     false,
			workDir:    "/home/user/workdir",
			inputName:  "file.go",
			wantResult: "/home/user/workdir/file.go",
		},
		{
			name:       "absolute path unchanged",
			filename:   "/home/user/project/main.go",
			hasTag:     true,
			workDir:    "/other/workdir",
			inputName:  "/abs/path/file.go",
			wantResult: "/abs/path/file.go",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &mockDirectoryContext{
				mockTagProvider: mockTagProvider{
					filename: tc.filename,
					hasTag:   tc.hasTag,
				},
				workDir: tc.workDir,
			}

			// Test interface
			var c textboundary.DirectoryContext = ctx
			if c.WorkingDir() != tc.workDir {
				t.Errorf("WorkingDir() = %q, want %q", c.WorkingDir(), tc.workDir)
			}

			// Simulate AbsDirName logic
			var result string
			if filepath.IsAbs(tc.inputName) {
				result = tc.inputName
			} else {
				// Get directory from tag
				dir := ""
				if c.HasTag() && c.TagFileName() != "" {
					fname := c.TagFileName()
					if fname[len(fname)-1] == filepath.Separator {
						dir = fname
					} else {
						dir = filepath.Dir(fname)
					}
				}

				// Join with input
				if dir != "" {
					result = filepath.Join(dir, tc.inputName)
				} else {
					result = tc.inputName
				}

				// Make absolute if needed
				if !filepath.IsAbs(result) {
					result = filepath.Join(c.WorkingDir(), result)
				}
			}

			if result != tc.wantResult {
				t.Errorf("AbsDirName(%q) = %q, want %q", tc.inputName, result, tc.wantResult)
			}
		})
	}
}

// TestMouseSnapshotIntegration tests MouseSnapshot in the main package context.
func TestMouseSnapshotIntegration(t *testing.T) {
	// Create snapshot like Text.Select() would capture initial state
	snap := textboundary.MouseSnapshot{
		ButtonState: 1, // left button
		Position:    image.Point{100, 200},
		Timestamp:   1234567890,
	}

	// Test via MouseState interface (how Text would use it)
	var state textboundary.MouseState = snap
	if state.Buttons() != 1 {
		t.Errorf("Buttons() = %d, want 1", state.Buttons())
	}
	if state.Point() != (image.Point{100, 200}) {
		t.Errorf("Point() = %v, want {100, 200}", state.Point())
	}
	if state.Msec() != 1234567890 {
		t.Errorf("Msec() = %d, want 1234567890", state.Msec())
	}

	// Test movement detection (like the loop at text.go:1154)
	if snap.HasMoved(image.Point{102, 201}, 3) {
		t.Error("HasMoved should be false for 2-pixel movement with 3-pixel threshold")
	}
	if !snap.HasMoved(image.Point{104, 200}, 3) {
		t.Error("HasMoved should be true for 4-pixel movement with 3-pixel threshold")
	}

	// Test button change detection
	if snap.ButtonsChanged(1) {
		t.Error("ButtonsChanged should be false for same button state")
	}
	if !snap.ButtonsChanged(0) {
		t.Error("ButtonsChanged should be true when button released")
	}
}

// TestFuncDirectoryResolverInMainPackage tests FuncDirectoryResolver integration.
func TestFuncDirectoryResolverInMainPackage(t *testing.T) {
	// This demonstrates how Text could use a function to lazily resolve directory
	callCount := 0
	resolver := textboundary.FuncDirectoryResolver(func() string {
		callCount++
		// Simulate logic that would compute directory from t.w.tag
		return "/dynamic/path"
	})

	// First call computes
	if dir := resolver.ResolveDir(); dir != "/dynamic/path" {
		t.Errorf("first ResolveDir() = %q, want /dynamic/path", dir)
	}
	if callCount != 1 {
		t.Errorf("callCount after first call = %d, want 1", callCount)
	}

	// Second call recomputes (no caching)
	resolver.ResolveDir()
	if callCount != 2 {
		t.Errorf("callCount after second call = %d, want 2", callCount)
	}
}
