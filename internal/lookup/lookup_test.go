// Package lookup provides unified window and file lookup utilities for edwood.
//
// This package centralizes the various lookup patterns used throughout edwood
// into a single, testable package with consistent interfaces.
package lookup

import (
	"testing"
)

// ===============================
// Mock types for testing
// ===============================

// MockWindow represents a minimal window interface for testing lookups.
type MockWindow struct {
	id       int
	name     string // File name associated with window
	col      *MockColumn
	rectMinY int
	rectMaxY int
}

func (w *MockWindow) ID() int             { return w.id }
func (w *MockWindow) Name() string        { return w.name }
func (w *MockWindow) Column() *MockColumn { return w.col }
func (w *MockWindow) RectMinY() int       { return w.rectMinY }
func (w *MockWindow) RectMaxY() int       { return w.rectMaxY }

// MockColumn represents a minimal column interface for testing lookups.
type MockColumn struct {
	windows []*MockWindow
}

func (c *MockColumn) Windows() []*MockWindow { return c.windows }
func (c *MockColumn) NumWindows() int        { return len(c.windows) }

// MockRow represents a minimal row interface for testing lookups.
type MockRow struct {
	columns []*MockColumn
}

func (r *MockRow) Columns() []*MockColumn { return r.columns }

// ===============================
// WindowByID tests
// ===============================

// TestWindowByIDEmptyRow tests lookup in an empty row.
func TestWindowByIDEmptyRow(t *testing.T) {
	row := &MockRow{columns: []*MockColumn{}}

	result := findWindowByID(row, 1)
	if result != nil {
		t.Errorf("expected nil for empty row, got %v", result)
	}
}

// TestWindowByIDEmptyColumns tests lookup in row with empty columns.
func TestWindowByIDEmptyColumns(t *testing.T) {
	row := &MockRow{
		columns: []*MockColumn{
			{windows: []*MockWindow{}},
			{windows: []*MockWindow{}},
		},
	}

	result := findWindowByID(row, 1)
	if result != nil {
		t.Errorf("expected nil for empty columns, got %v", result)
	}
}

// TestWindowByIDFound tests finding an existing window.
func TestWindowByIDFound(t *testing.T) {
	win1 := &MockWindow{id: 1, name: "file1.go"}
	win2 := &MockWindow{id: 2, name: "file2.go"}
	win3 := &MockWindow{id: 3, name: "file3.go"}

	col1 := &MockColumn{windows: []*MockWindow{win1, win2}}
	col2 := &MockColumn{windows: []*MockWindow{win3}}
	row := &MockRow{columns: []*MockColumn{col1, col2}}

	testCases := []struct {
		name     string
		id       int
		expected *MockWindow
	}{
		{"first window", 1, win1},
		{"middle window", 2, win2},
		{"window in second column", 3, win3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findWindowByID(row, tc.id)
			if result != tc.expected {
				t.Errorf("expected window %d, got %v", tc.expected.id, result)
			}
		})
	}
}

// TestWindowByIDNotFound tests lookup for non-existent window.
func TestWindowByIDNotFound(t *testing.T) {
	win := &MockWindow{id: 1, name: "file.go"}
	col := &MockColumn{windows: []*MockWindow{win}}
	row := &MockRow{columns: []*MockColumn{col}}

	result := findWindowByID(row, 999)
	if result != nil {
		t.Errorf("expected nil for non-existent ID, got %v", result)
	}
}

// ===============================
// WindowByName tests
// ===============================

// TestWindowByNameEmptyRow tests file lookup in an empty row.
func TestWindowByNameEmptyRow(t *testing.T) {
	row := &MockRow{columns: []*MockColumn{}}

	result := findWindowByName(row, "test.go")
	if result != nil {
		t.Errorf("expected nil for empty row, got %v", result)
	}
}

// TestWindowByNameExactMatch tests exact name matching.
func TestWindowByNameExactMatch(t *testing.T) {
	win := &MockWindow{id: 1, name: "/path/to/file.go"}
	col := &MockColumn{windows: []*MockWindow{win}}
	row := &MockRow{columns: []*MockColumn{col}}
	win.col = col

	result := findWindowByName(row, "/path/to/file.go")
	if result != win {
		t.Errorf("expected to find window, got %v", result)
	}
}

// TestWindowByNameTrailingSlash tests that trailing slashes are handled.
func TestWindowByNameTrailingSlash(t *testing.T) {
	win := &MockWindow{id: 1, name: "/path/to/dir"}
	col := &MockColumn{windows: []*MockWindow{win}}
	row := &MockRow{columns: []*MockColumn{col}}
	win.col = col

	testCases := []struct {
		name     string
		query    string
		expected *MockWindow
	}{
		{"no trailing slash", "/path/to/dir", win},
		{"with trailing slash", "/path/to/dir/", win},
		{"multiple trailing slashes", "/path/to/dir///", win},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findWindowByName(row, tc.query)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestWindowByNameNotFound tests lookup for non-existent file.
func TestWindowByNameNotFound(t *testing.T) {
	win := &MockWindow{id: 1, name: "/path/to/file.go"}
	col := &MockColumn{windows: []*MockWindow{win}}
	row := &MockRow{columns: []*MockColumn{col}}

	result := findWindowByName(row, "/other/file.go")
	if result != nil {
		t.Errorf("expected nil for non-existent file, got %v", result)
	}
}

// TestWindowByNameNilColumn tests that windows with nil column are skipped.
func TestWindowByNameNilColumn(t *testing.T) {
	win := &MockWindow{id: 1, name: "/path/to/file.go", col: nil}
	col := &MockColumn{windows: []*MockWindow{win}}
	row := &MockRow{columns: []*MockColumn{col}}

	result := findWindowByName(row, "/path/to/file.go")
	if result != nil {
		t.Errorf("expected nil for window with nil column, got %v", result)
	}
}

// ===============================
// WindowContainingY tests
// ===============================

// TestWindowContainingYEmpty tests lookup in an empty column.
func TestWindowContainingYEmpty(t *testing.T) {
	col := &MockColumn{windows: []*MockWindow{}}

	idx, win := findWindowContainingY(col, 100)
	if idx != 0 || win != nil {
		t.Errorf("expected (0, nil) for empty column, got (%d, %v)", idx, win)
	}
}

// TestWindowContainingYSingleWindow tests lookup with single window.
func TestWindowContainingYSingleWindow(t *testing.T) {
	win := &MockWindow{id: 1, rectMinY: 0, rectMaxY: 200}
	col := &MockColumn{windows: []*MockWindow{win}}

	testCases := []struct {
		name        string
		y           int
		expectIdx   int
		expectFound bool
	}{
		{"y at top", 0, 0, true},
		{"y in middle", 100, 0, true},
		{"y at max-1", 199, 0, true},
		{"y at max", 200, 1, false},
		{"y beyond", 300, 1, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idx, w := findWindowContainingY(col, tc.y)
			if idx != tc.expectIdx {
				t.Errorf("expected index %d, got %d", tc.expectIdx, idx)
			}
			if tc.expectFound && w != win {
				t.Errorf("expected to find window")
			}
			if !tc.expectFound && w != nil && idx < len(col.windows) {
				t.Errorf("expected not to find window within bounds")
			}
		})
	}
}

// TestWindowContainingYMultipleWindows tests lookup across multiple windows.
func TestWindowContainingYMultipleWindows(t *testing.T) {
	win1 := &MockWindow{id: 1, rectMinY: 0, rectMaxY: 100}
	win2 := &MockWindow{id: 2, rectMinY: 100, rectMaxY: 200}
	win3 := &MockWindow{id: 3, rectMinY: 200, rectMaxY: 300}
	col := &MockColumn{windows: []*MockWindow{win1, win2, win3}}

	testCases := []struct {
		name      string
		y         int
		expectIdx int
		expectWin *MockWindow
	}{
		{"first window", 50, 0, win1},
		{"boundary 1-2", 100, 1, win2},
		{"second window", 150, 1, win2},
		{"boundary 2-3", 200, 2, win3},
		{"third window", 250, 2, win3},
		{"beyond all", 350, 3, win3}, // returns last window when beyond
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idx, w := findWindowContainingY(col, tc.y)
			if idx != tc.expectIdx {
				t.Errorf("expected index %d, got %d", tc.expectIdx, idx)
			}
			if idx < len(col.windows) && w != tc.expectWin {
				t.Errorf("expected window %d, got %v", tc.expectWin.id, w)
			}
		})
	}
}

// ===============================
// WindowIndex tests
// ===============================

// TestWindowIndexFound tests finding a window's index.
func TestWindowIndexFound(t *testing.T) {
	win1 := &MockWindow{id: 1}
	win2 := &MockWindow{id: 2}
	win3 := &MockWindow{id: 3}
	col := &MockColumn{windows: []*MockWindow{win1, win2, win3}}

	testCases := []struct {
		name      string
		win       *MockWindow
		expectIdx int
	}{
		{"first window", win1, 0},
		{"middle window", win2, 1},
		{"last window", win3, 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idx := findWindowIndex(col, tc.win)
			if idx != tc.expectIdx {
				t.Errorf("expected index %d, got %d", tc.expectIdx, idx)
			}
		})
	}
}

// TestWindowIndexNotFound tests lookup for non-member window.
func TestWindowIndexNotFound(t *testing.T) {
	win1 := &MockWindow{id: 1}
	win2 := &MockWindow{id: 2}
	other := &MockWindow{id: 99}
	col := &MockColumn{windows: []*MockWindow{win1, win2}}

	idx := findWindowIndex(col, other)
	if idx != -1 {
		t.Errorf("expected -1 for non-member, got %d", idx)
	}
}

// TestWindowIndexEmpty tests lookup in empty column.
func TestWindowIndexEmpty(t *testing.T) {
	col := &MockColumn{windows: []*MockWindow{}}
	win := &MockWindow{id: 1}

	idx := findWindowIndex(col, win)
	if idx != -1 {
		t.Errorf("expected -1 for empty column, got %d", idx)
	}
}

// TestWindowIndexNilWindow tests lookup with nil window.
func TestWindowIndexNilWindow(t *testing.T) {
	win1 := &MockWindow{id: 1}
	col := &MockColumn{windows: []*MockWindow{win1}}

	idx := findWindowIndex(col, nil)
	if idx != -1 {
		t.Errorf("expected -1 for nil window, got %d", idx)
	}
}

// ===============================
// AllWindows iterator tests
// ===============================

// TestAllWindowsEmpty tests iteration over empty row.
func TestAllWindowsEmpty(t *testing.T) {
	row := &MockRow{columns: []*MockColumn{}}
	count := 0

	forAllWindows(row, func(w *MockWindow) {
		count++
	})

	if count != 0 {
		t.Errorf("expected 0 iterations, got %d", count)
	}
}

// TestAllWindowsIteratesAll tests that all windows are visited.
func TestAllWindowsIteratesAll(t *testing.T) {
	win1 := &MockWindow{id: 1}
	win2 := &MockWindow{id: 2}
	win3 := &MockWindow{id: 3}
	col1 := &MockColumn{windows: []*MockWindow{win1, win2}}
	col2 := &MockColumn{windows: []*MockWindow{win3}}
	row := &MockRow{columns: []*MockColumn{col1, col2}}

	visited := make(map[int]bool)
	forAllWindows(row, func(w *MockWindow) {
		visited[w.id] = true
	})

	if len(visited) != 3 {
		t.Errorf("expected 3 windows visited, got %d", len(visited))
	}
	for _, win := range []*MockWindow{win1, win2, win3} {
		if !visited[win.id] {
			t.Errorf("window %d was not visited", win.id)
		}
	}
}

// TestAllWindowsOrder tests iteration order (columns first, then windows within).
func TestAllWindowsOrder(t *testing.T) {
	win1 := &MockWindow{id: 1}
	win2 := &MockWindow{id: 2}
	win3 := &MockWindow{id: 3}
	win4 := &MockWindow{id: 4}
	col1 := &MockColumn{windows: []*MockWindow{win1, win2}}
	col2 := &MockColumn{windows: []*MockWindow{win3, win4}}
	row := &MockRow{columns: []*MockColumn{col1, col2}}

	var order []int
	forAllWindows(row, func(w *MockWindow) {
		order = append(order, w.id)
	})

	expected := []int{1, 2, 3, 4}
	if len(order) != len(expected) {
		t.Errorf("expected %d iterations, got %d", len(expected), len(order))
	}
	for i, id := range expected {
		if i >= len(order) || order[i] != id {
			t.Errorf("order[%d]: expected %d, got %d", i, id, order[i])
		}
	}
}

// ===============================
// Path normalization tests
// ===============================

// TestNormalizePathTrailingSlash tests slash normalization.
func TestNormalizePathTrailingSlash(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"/path/to/file", "/path/to/file"},
		{"/path/to/dir/", "/path/to/dir"},
		{"/path/to/dir///", "/path/to/dir"},
		{"/", ""},       // Edge case: root
		{"///", ""},     // Edge case: only slashes
		{"file.go", "file.go"},
		{"", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := NormalizePath(tc.input)
			if result != tc.expected {
				t.Errorf("NormalizePath(%q): expected %q, got %q", tc.input, tc.expected, result)
			}
		})
	}
}

// TestNormalizePathBackslash tests backslash handling (Windows paths).
func TestNormalizePathBackslash(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{`C:\path\to\file`, `C:\path\to\file`},
		{`C:\path\to\dir\`, `C:\path\to\dir`},
		{`C:\path\to\dir\\\`, `C:\path\to\dir`},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := NormalizePath(tc.input)
			if result != tc.expected {
				t.Errorf("NormalizePath(%q): expected %q, got %q", tc.input, tc.expected, result)
			}
		})
	}
}

// ===============================
// Finder (unified lookup) tests
// ===============================

// TestFinderNew tests creating a new Finder.
func TestFinderNew(t *testing.T) {
	row := &MockRow{columns: []*MockColumn{}}
	f := newMockFinder(row)

	if f == nil {
		t.Fatal("newMockFinder returned nil")
	}
	if f.row != row {
		t.Error("MockFinder.row not set correctly")
	}
}

// TestFinderByID tests Finder.ByID method.
func TestFinderByID(t *testing.T) {
	win := &MockWindow{id: 42, name: "test.go"}
	col := &MockColumn{windows: []*MockWindow{win}}
	row := &MockRow{columns: []*MockColumn{col}}
	f := newMockFinder(row)

	result := f.ByID(42)
	if result != win {
		t.Errorf("expected window, got %v", result)
	}

	result = f.ByID(999)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// TestFinderByName tests Finder.ByName method.
func TestFinderByName(t *testing.T) {
	win := &MockWindow{id: 1, name: "/path/to/test.go"}
	col := &MockColumn{windows: []*MockWindow{win}}
	row := &MockRow{columns: []*MockColumn{col}}
	win.col = col
	f := newMockFinder(row)

	result := f.ByName("/path/to/test.go")
	if result != win {
		t.Errorf("expected window, got %v", result)
	}

	result = f.ByName("/other/file.go")
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// TestFinderAll tests Finder.All method.
func TestFinderAll(t *testing.T) {
	win1 := &MockWindow{id: 1}
	win2 := &MockWindow{id: 2}
	col := &MockColumn{windows: []*MockWindow{win1, win2}}
	row := &MockRow{columns: []*MockColumn{col}}
	f := newMockFinder(row)

	var all []*MockWindow
	f.All(func(w *MockWindow) {
		all = append(all, w)
	})

	if len(all) != 2 {
		t.Errorf("expected 2 windows, got %d", len(all))
	}
}

// ===============================
// PathMatcher tests
// ===============================

// TestPathMatcherExact tests exact path matching.
func TestPathMatcherExact(t *testing.T) {
	pm := NewPathMatcher("/home/user/project")

	testCases := []struct {
		name     string
		pattern  string
		target   string
		expected bool
	}{
		{"exact match", "/home/user/file.go", "/home/user/file.go", true},
		{"different paths", "/home/user/file.go", "/home/other/file.go", false},
		{"case sensitive", "/home/User/file.go", "/home/user/file.go", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := pm.Matches(tc.pattern, tc.target)
			if result != tc.expected {
				t.Errorf("Matches(%q, %q): expected %v, got %v", tc.pattern, tc.target, tc.expected, result)
			}
		})
	}
}

// TestPathMatcherRelativeAbsolute tests relative to absolute path resolution.
func TestPathMatcherRelativeAbsolute(t *testing.T) {
	pm := NewPathMatcher("/home/user/project")

	testCases := []struct {
		name     string
		pattern  string
		target   string
		expected bool
	}{
		{"relative to absolute", "file.go", "/home/user/project/file.go", true},
		{"relative subdir", "src/main.go", "/home/user/project/src/main.go", true},
		{"absolute stays absolute", "/other/file.go", "/other/file.go", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := pm.Matches(tc.pattern, tc.target)
			if result != tc.expected {
				t.Errorf("Matches(%q, %q): expected %v, got %v", tc.pattern, tc.target, tc.expected, result)
			}
		})
	}
}

// ===============================
// Result type tests
// ===============================

// TestLookupResultFound tests LookupResult for found windows.
func TestLookupResultFound(t *testing.T) {
	win := &MockWindow{id: 1, name: "test.go"}
	result := &LookupResult[*MockWindow]{
		Window: win,
		Found:  true,
	}

	if !result.Found {
		t.Error("expected Found to be true")
	}
	if result.Window != win {
		t.Error("expected Window to be set")
	}
}

// TestLookupResultNotFound tests LookupResult for not found.
func TestLookupResultNotFound(t *testing.T) {
	result := &LookupResult[*MockWindow]{
		Window: nil,
		Found:  false,
	}

	if result.Found {
		t.Error("expected Found to be false")
	}
	if result.Window != nil {
		t.Error("expected Window to be nil")
	}
}

// ===============================
// Tests for exported generic functions
// ===============================

// TestFindByIDGeneric tests the generic FindByID function.
func TestFindByIDGeneric(t *testing.T) {
	win1 := &MockWindow{id: 1, name: "file1.go"}
	win2 := &MockWindow{id: 2, name: "file2.go"}
	col := &MockColumn{windows: []*MockWindow{win1, win2}}
	row := &MockRow{columns: []*MockColumn{col}}

	getColumns := func(r *MockRow) []*MockColumn { return r.columns }
	getWindows := func(c *MockColumn) []*MockWindow { return c.windows }

	result := FindByID(row, 1, getColumns, getWindows)
	if result != win1 {
		t.Errorf("expected win1, got %v", result)
	}

	result = FindByID(row, 999, getColumns, getWindows)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// TestFindByNameGeneric tests the generic FindByName function.
func TestFindByNameGeneric(t *testing.T) {
	win := &MockWindow{id: 1, name: "/path/to/file.go"}
	col := &MockColumn{windows: []*MockWindow{win}}
	row := &MockRow{columns: []*MockColumn{col}}
	win.col = col

	getColumns := func(r *MockRow) []*MockColumn { return r.columns }
	getWindows := func(c *MockColumn) []*MockWindow { return c.windows }
	hasColumn := func(w *MockWindow) bool { return w.col != nil }

	result := FindByName(row, "/path/to/file.go", getColumns, getWindows, hasColumn)
	if result != win {
		t.Errorf("expected win, got %v", result)
	}

	result = FindByName(row, "/other/file.go", getColumns, getWindows, hasColumn)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// TestFindContainingYGeneric tests the generic FindContainingY function.
func TestFindContainingYGeneric(t *testing.T) {
	win1 := &MockWindow{id: 1, rectMinY: 0, rectMaxY: 100}
	win2 := &MockWindow{id: 2, rectMinY: 100, rectMaxY: 200}
	windows := []*MockWindow{win1, win2}

	getRectMaxY := func(w *MockWindow) int { return w.rectMaxY }

	idx, w := FindContainingY(windows, 50, getRectMaxY)
	if idx != 0 || w != win1 {
		t.Errorf("expected (0, win1), got (%d, %v)", idx, w)
	}

	idx, w = FindContainingY(windows, 150, getRectMaxY)
	if idx != 1 || w != win2 {
		t.Errorf("expected (1, win2), got (%d, %v)", idx, w)
	}
}

// TestFindWindowIndexGeneric tests the generic FindWindowIndex function.
func TestFindWindowIndexGeneric(t *testing.T) {
	win1 := &MockWindow{id: 1}
	win2 := &MockWindow{id: 2}
	windows := []*MockWindow{win1, win2}

	idx := FindWindowIndex(windows, win1)
	if idx != 0 {
		t.Errorf("expected 0, got %d", idx)
	}

	idx = FindWindowIndex(windows, win2)
	if idx != 1 {
		t.Errorf("expected 1, got %d", idx)
	}

	other := &MockWindow{id: 99}
	idx = FindWindowIndex(windows, other)
	if idx != -1 {
		t.Errorf("expected -1, got %d", idx)
	}
}

// TestForAllGeneric tests the generic ForAll function.
func TestForAllGeneric(t *testing.T) {
	win1 := &MockWindow{id: 1}
	win2 := &MockWindow{id: 2}
	col := &MockColumn{windows: []*MockWindow{win1, win2}}
	row := &MockRow{columns: []*MockColumn{col}}

	getColumns := func(r *MockRow) []*MockColumn { return r.columns }
	getWindows := func(c *MockColumn) []*MockWindow { return c.windows }

	var visited []int
	ForAll(row, getColumns, getWindows, func(w *MockWindow) {
		visited = append(visited, w.id)
	})

	if len(visited) != 2 || visited[0] != 1 || visited[1] != 2 {
		t.Errorf("expected [1, 2], got %v", visited)
	}
}

// TestGenericFinderType tests the generic Finder type from lookup.go.
func TestGenericFinderType(t *testing.T) {
	win := &MockWindow{id: 42, name: "/path/to/file.go"}
	col := &MockColumn{windows: []*MockWindow{win}}
	row := &MockRow{columns: []*MockColumn{col}}
	win.col = col

	getColumns := func(r *MockRow) []*MockColumn { return r.columns }
	getWindows := func(c *MockColumn) []*MockWindow { return c.windows }
	hasColumn := func(w *MockWindow) bool { return w.col != nil }

	finder := NewFinder(row, getColumns, getWindows, hasColumn)

	result := finder.ByID(42)
	if result != win {
		t.Errorf("expected win, got %v", result)
	}

	result = finder.ByName("/path/to/file.go")
	if result != win {
		t.Errorf("expected win, got %v", result)
	}

	var visited []*MockWindow
	finder.All(func(w *MockWindow) {
		visited = append(visited, w)
	})
	if len(visited) != 1 || visited[0] != win {
		t.Errorf("expected [win], got %v", visited)
	}
}

// ===============================
// Helper implementations for tests
// ===============================

// findWindowByID searches for a window by ID across all columns.
func findWindowByID(row *MockRow, id int) *MockWindow {
	for _, col := range row.columns {
		for _, win := range col.windows {
			if win.id == id {
				return win
			}
		}
	}
	return nil
}

// findWindowByName searches for a window by file name across all columns.
func findWindowByName(row *MockRow, name string) *MockWindow {
	name = NormalizePath(name)
	for _, col := range row.columns {
		for _, win := range col.windows {
			winName := NormalizePath(win.name)
			if winName == name {
				if win.col != nil {
					return win
				}
			}
		}
	}
	return nil
}

// findWindowContainingY finds the window containing the given Y coordinate.
func findWindowContainingY(col *MockColumn, y int) (int, *MockWindow) {
	var lastWin *MockWindow
	for i, win := range col.windows {
		lastWin = win
		if y < win.rectMaxY {
			return i, win
		}
	}
	return len(col.windows), lastWin
}

// findWindowIndex returns the index of the window in the column, or -1 if not found.
func findWindowIndex(col *MockColumn, win *MockWindow) int {
	if win == nil {
		return -1
	}
	for i, w := range col.windows {
		if w == win {
			return i
		}
	}
	return -1
}

// forAllWindows calls the provided function for each window in the row.
func forAllWindows(row *MockRow, fn func(*MockWindow)) {
	for _, col := range row.columns {
		for _, win := range col.windows {
			fn(win)
		}
	}
}

// MockFinder provides unified window lookup capabilities for tests.
type MockFinder struct {
	row *MockRow
}

// newMockFinder creates a new MockFinder for the given row.
func newMockFinder(row *MockRow) *MockFinder {
	return &MockFinder{row: row}
}

// ByID looks up a window by its unique ID.
func (f *MockFinder) ByID(id int) *MockWindow {
	return findWindowByID(f.row, id)
}

// ByName looks up a window by its file name.
func (f *MockFinder) ByName(name string) *MockWindow {
	return findWindowByName(f.row, name)
}

// All calls the provided function for each window.
func (f *MockFinder) All(fn func(*MockWindow)) {
	forAllWindows(f.row, fn)
}
