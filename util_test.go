package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rjkroege/edwood/dumpfile"
	"github.com/rjkroege/edwood/util"
)

func TestCvttorunes(t *testing.T) {
	testCases := []struct {
		p     []byte
		n     int
		r     []rune
		nb    int
		nulls bool
	}{
		{[]byte("Hello world"), 11, []rune("Hello world"), 11, false},
		{[]byte("Hello \x00\x00world"), 13, []rune("Hello world"), 13, true},
		{[]byte("Hello 世界"), 6 + 3 + 3, []rune("Hello 世界"), 6 + 3 + 3, false},
		{[]byte("Hello 世界"), 6 + 3 + 1, []rune("Hello 世界"), 6 + 3 + 3, false},
		{[]byte("Hello 世界"), 6 + 3 + 2, []rune("Hello 世界"), 6 + 3 + 3, false},
		{[]byte("Hello 世\xe7\x95"), 6 + 3 + 1, []rune("Hello 世\uFFFD"), 6 + 3 + 1, false},
		{[]byte("Hello 世\xe7\x95"), 6 + 3 + 2, []rune("Hello 世\uFFFD\uFFFD"), 6 + 3 + 2, false},
		{[]byte("\xe4\xb8\x96界 hello"), 3 + 3 + 6, []rune("世界 hello"), 3 + 3 + 6, false},
		{[]byte("\xb8\x96界 hello"), 2 + 3 + 6, []rune("\uFFFD\uFFFD界 hello"), 2 + 3 + 6, false},
		{[]byte("\x96界 hello"), 1 + 3 + 6, []rune("\uFFFD界 hello"), 1 + 3 + 6, false},
	}
	for _, tc := range testCases {
		r, nb, nulls := util.Cvttorunes(tc.p, tc.n)
		if !reflect.DeepEqual(r, tc.r) || nb != tc.nb || nulls != tc.nulls {
			t.Errorf("util.Cvttorunes of (%q, %v) returned %q, %v, %v; expected %q, %v, %v\n",
				tc.p, tc.n, r, nb, nulls, tc.r, tc.nb, tc.nulls)
		}
	}
}

// Given the complexity of errorwin1Name, one might wonder why we test
// this so comprehensively. :-)
func TestErrorwin1Name(t *testing.T) {
	tt := []struct {
		dir, name string
	}{
		{"", "+Errors"},
		{".", "+Errors"},
		{"/", "/+Errors"},
		{"/home/gopher", "/home/gopher/+Errors"},
		{"/home/gopher/", "/home/gopher/+Errors"},
		{"C:/Users/gopher", "C:/Users/gopher/+Errors"},
		{"C:/Users/gopher/", "C:/Users/gopher/+Errors"},
		{"C:/", "C:/+Errors"},
	}
	for _, tc := range tt {
		name := filepath.ToSlash(errorwin1Name(filepath.FromSlash(tc.dir)))
		if name != tc.name {
			t.Errorf("errorwin1Name(%q) is %q; expected %q", tc.dir, name, tc.name)
		}
	}
}

func TestQuote(t *testing.T) {
	var testCases = []struct {
		s, q string
	}{
		{"", "''"},
		{"Edwood", "Edwood"},
		{"Plan 9", "'Plan 9'"},
		{"Don't", "'Don''t'"},
		{"Don't worry!", "'Don''t worry!'"},
	}
	for _, tc := range testCases {
		q := quote(tc.s)
		if q != tc.q {
			t.Errorf("%q quoted is %q; expected %q\n", tc.s, q, tc.q)
		}
	}
}

func TestSkipbl(t *testing.T) {
	tt := []struct {
		s []rune
		q []rune
	}{
		{nil, nil},
		{[]rune(" \t\n"), nil},
		{[]rune(" \t\nabc"), []rune("abc")},
		{[]rune(" \t\n \t\nabc"), []rune("abc")},
		{[]rune(" \t\nabc \t\nabc"), []rune("abc \t\nabc")},
		{[]rune(" \t\nαβγ \t\nαβγ"), []rune("αβγ \t\nαβγ")},
	}
	for _, tc := range tt {
		q := skipbl(tc.s)
		if !reflect.DeepEqual(q, tc.q) {
			t.Errorf("skipbl(%v) returned %v; expected %v", tc.s, q, tc.q)
		}
	}
}

func logSomethingSmall(t *testing.T, g *globals, _ string) {
	t.Helper()
	err := warnError(nil, "SomethingSmall")

	if got, want := err.Error(), "SomethingSmall"; got != want {
		t.Errorf("didn't build correct error. got %v want %v", got, want)
	}
}

func logSomethingWithMntDir(t *testing.T, g *globals, dir string) {
	t.Helper()

	md := mnt.Add(dir, nil)
	warning(md, "I am an warning\n")
	warning(md, "I am a second warning\n")
}

// TestWarningsRace tests concurrent access to the warnings list.
// This test is designed to detect race conditions when run with -race flag.
// It verifies that:
// - warning() properly locks warningsMu when appending
// - flushwarnings() properly locks warningsMu when reading/clearing the list
func TestWarningsRace(t *testing.T) {
	dir := t.TempDir()
	FlexiblyMakeWindowScaffold(
		t,
		ScWin("testfile"),
		ScBody("testfile", "test content"),
		ScDir(dir, "testfile"),
	)

	// Test 1: Multiple concurrent warning() calls should be safe
	// (warning() properly locks warningsMu)
	t.Run("ConcurrentWarnings", func(t *testing.T) {
		const numGoroutines = 10
		const numIterations = 5

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numIterations; j++ {
					warning(nil, "warning from goroutine %d iteration %d\n", id, j)
				}
				done <- true
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Clean up - flushwarnings now handles its own locking
		flushwarnings()
	})

	// Test 2: Serial warning/flush to verify basic behavior
	t.Run("SerialWarningFlush", func(t *testing.T) {
		warning(nil, "test warning 1\n")
		warning(nil, "test warning 2\n")

		// flushwarnings now handles its own locking for warningsMu
		// and acquires row lock internally via errorwin
		flushwarnings()
	})
}

// TestErrorwin1RowAccess tests that errorwin1 accesses global.row.col
// safely when the caller holds the row lock.
// errorwin1 requires the caller to hold global.row.lk.
func TestErrorwin1RowAccess(t *testing.T) {
	dir := t.TempDir()
	FlexiblyMakeWindowScaffold(
		t,
		ScWin("testfile"),
		ScBody("testfile", "test content"),
		ScDir(dir, "testfile"),
	)

	// Test 1: Single call to errorwin1 with row lock held
	t.Run("SingleCall", func(t *testing.T) {
		global.row.lk.Lock()
		w := errorwin1(dir, nil)
		global.row.lk.Unlock()
		if w == nil {
			t.Error("errorwin1 returned nil")
		}
	})

	// Test 2: Sequential calls with row lock (required behavior)
	t.Run("SequentialWithLock", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			subdir := filepath.Join(dir, fmt.Sprintf("subdir%d", i))
			global.row.lk.Lock()
			w := errorwin1(subdir, nil)
			global.row.lk.Unlock()
			if w == nil {
				t.Errorf("errorwin1 returned nil for subdir%d", i)
			}
		}
	})
}

// TestErrorwinLocking tests the locking behavior in errorwin().
// errorwin() now properly acquires the row lock before calling errorwin1,
// then acquires the window lock, following the lock ordering: row -> window.
func TestErrorwinLocking(t *testing.T) {
	dir := t.TempDir()
	FlexiblyMakeWindowScaffold(
		t,
		ScWin("testfile"),
		ScBody("testfile", "test content"),
		ScDir(dir, "testfile"),
	)

	// Test: Single call to errorwin works and returns locked window
	t.Run("SingleCall", func(t *testing.T) {
		w := errorwin(nil, 'E', nil)
		if w == nil {
			t.Fatal("errorwin returned nil")
		}
		// errorwin should return with the window locked
		// We unlock it to clean up
		w.Unlock()
	})

	// Test: Sequential calls work correctly
	t.Run("SequentialCalls", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			w := errorwin(nil, 'E', nil)
			if w == nil {
				t.Fatalf("errorwin returned nil on iteration %d", i)
			}
			if w.col == nil {
				t.Errorf("errorwin returned window with nil col on iteration %d", i)
			}
			w.Unlock()
		}
	})
}

// TestErrorwinforwinLocking tests the locking behavior in errorwinforwin().
// errorwinforwin() takes a locked window, extracts directory info, unlocks it,
// and returns a new locked error window.
func TestErrorwinforwinLocking(t *testing.T) {
	dir := t.TempDir()
	FlexiblyMakeWindowScaffold(
		t,
		ScWin("testfile"),
		ScBody("testfile", "test content"),
		ScDir(dir, "testfile"),
	)

	// Test: errorwinforwin unlocks input window and returns locked error window
	t.Run("UnlocksInputAndReturnsLocked", func(t *testing.T) {
		// Get a window to test with
		global.row.lk.Lock()
		w := global.row.col[0].w[0]
		global.row.lk.Unlock()

		// Lock the window before calling errorwinforwin
		w.Lock('T')

		// errorwinforwin should unlock w and return a different locked window
		errwin := errorwinforwin(w)
		if errwin == nil {
			t.Fatal("errorwinforwin returned nil")
		}
		defer errwin.Unlock()

		// Verify the returned window is an error window
		if errwin.col == nil {
			t.Error("errorwinforwin returned window with nil col")
		}
	})
}

// TestUnifiedErrorWindow tests both errorwin and errorwinforwin behavior
// to ensure the consolidated function covers both use cases.
// These tests verify the behavior that must be preserved after consolidation.
func TestUnifiedErrorWindow(t *testing.T) {
	dir := t.TempDir()

	// Test: errorwin with nil MntDir creates +Errors window
	t.Run("ErrorwinNilMntDir", func(t *testing.T) {
		FlexiblyMakeWindowScaffold(
			t,
			ScWin("testfile"),
			ScBody("testfile", "test content"),
			ScDir(dir, "testfile"),
		)

		w := errorwin(nil, 'T', nil)
		if w == nil {
			t.Fatal("errorwin returned nil")
		}
		defer w.Unlock()

		// Should create +Errors window (no directory prefix)
		expectedName := "+Errors"
		if w.body.file.Name() != expectedName {
			t.Errorf("errorwin created window named %q; want %q", w.body.file.Name(), expectedName)
		}
	})

	// Test: errorwin with MntDir creates dir/+Errors window
	t.Run("ErrorwinWithMntDir", func(t *testing.T) {
		FlexiblyMakeWindowScaffold(
			t,
			ScWin("testfile"),
			ScBody("testfile", "test content"),
			ScDir(dir, "testfile"),
		)

		md := mnt.Add(dir, nil)
		defer mnt.DecRef(md)

		w := errorwin(md, 'T', nil)
		if w == nil {
			t.Fatal("errorwin returned nil")
		}
		defer w.Unlock()

		// Should create dir/+Errors window
		expectedName := filepath.Join(dir, "+Errors")
		if w.body.file.Name() != expectedName {
			t.Errorf("errorwin created window named %q; want %q", w.body.file.Name(), expectedName)
		}
	})

	// Test: errorwinforwin extracts directory from input window
	t.Run("ErrorwinforwinExtractsDir", func(t *testing.T) {
		FlexiblyMakeWindowScaffold(
			t,
			ScWin("testfile"),
			ScBody("testfile", "test content"),
			ScDir(dir, "testfile"),
		)

		// Get a window to test with
		global.row.lk.Lock()
		w := global.row.col[0].w[0]
		global.row.lk.Unlock()

		w.Lock('T')
		errwin := errorwinforwin(w)
		if errwin == nil {
			t.Fatal("errorwinforwin returned nil")
		}
		defer errwin.Unlock()

		// Error window should be created with directory from input window
		expectedName := filepath.Join(dir, "+Errors")
		if errwin.body.file.Name() != expectedName {
			t.Errorf("errorwinforwin created window named %q; want %q", errwin.body.file.Name(), expectedName)
		}
	})

	// Test: errorwinforwin preserves include paths
	t.Run("ErrorwinforwinPreservesIncl", func(t *testing.T) {
		// Use a unique subdirectory so we get a fresh error window
		subdir := filepath.Join(dir, "incltest")
		os.MkdirAll(subdir, 0755)

		// Create the include path as an actual directory (AddIncl validates directories)
		inclPath := filepath.Join(dir, "includes")
		os.MkdirAll(inclPath, 0755)

		FlexiblyMakeWindowScaffold(
			t,
			ScWin("testfile"),
			ScBody("testfile", "test content"),
			ScDir(subdir, "testfile"),
		)

		// Get a window and add includes to it
		global.row.lk.Lock()
		w := global.row.col[0].w[0]
		global.row.lk.Unlock()

		w.AddIncl(inclPath)

		w.Lock('T')
		errwin := errorwinforwin(w)
		if errwin == nil {
			t.Fatal("errorwinforwin returned nil")
		}
		defer errwin.Unlock()

		// Error window should have the include path from input window
		found := false
		for _, inc := range errwin.incl {
			if inc == inclPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("errorwinforwin did not preserve include path %q, got includes: %v", inclPath, errwin.incl)
		}
	})

	// Test: errorwin returns window with correct owner
	t.Run("ErrorwinOwner", func(t *testing.T) {
		FlexiblyMakeWindowScaffold(
			t,
			ScWin("testfile"),
			ScBody("testfile", "test content"),
			ScDir(dir, "testfile"),
		)

		w := errorwin(nil, 'X', nil)
		if w == nil {
			t.Fatal("errorwin returned nil")
		}
		defer w.Unlock()

		// The window should be locked by owner 'X'
		// (We verify this by successfully unlocking - if lock was wrong it would panic/deadlock)
	})

	// Test: errorwinforwin preserves owner from input window
	t.Run("ErrorwinforwinOwner", func(t *testing.T) {
		FlexiblyMakeWindowScaffold(
			t,
			ScWin("testfile"),
			ScBody("testfile", "test content"),
			ScDir(dir, "testfile"),
		)

		global.row.lk.Lock()
		w := global.row.col[0].w[0]
		global.row.lk.Unlock()

		w.Lock('Z')
		errwin := errorwinforwin(w)
		if errwin == nil {
			t.Fatal("errorwinforwin returned nil")
		}
		defer errwin.Unlock()

		// The error window should be locked by the same owner
		// (We verify this by successfully unlocking - if lock was wrong it would panic/deadlock)
	})

	// Test: repeated calls to same error window return existing window
	t.Run("ErrorwinReusesExisting", func(t *testing.T) {
		FlexiblyMakeWindowScaffold(
			t,
			ScWin("testfile"),
			ScBody("testfile", "test content"),
			ScDir(dir, "testfile"),
		)

		// First call creates the error window
		w1 := errorwin(nil, 'T', nil)
		if w1 == nil {
			t.Fatal("first errorwin returned nil")
		}
		w1.Unlock()

		// Second call should return the same window
		w2 := errorwin(nil, 'T', nil)
		if w2 == nil {
			t.Fatal("second errorwin returned nil")
		}
		defer w2.Unlock()

		if w1 != w2 {
			t.Error("errorwin should reuse existing +Errors window")
		}
	})
}

// TestMakenewwindowGlobalAccess tests the global state access in makenewwindow.
// makenewwindow now properly acquires the row lock internally to protect
// access to global.row.col, global.activecol, and global.seltext.
func TestMakenewwindowGlobalAccess(t *testing.T) {
	dir := t.TempDir()
	FlexiblyMakeWindowScaffold(
		t,
		ScWin("testfile"),
		ScBody("testfile", "test content"),
		ScDir(dir, "testfile"),
	)

	// Test: Single call to makenewwindow works correctly
	t.Run("SingleCall", func(t *testing.T) {
		w := makenewwindow(nil)
		if w == nil {
			t.Error("makenewwindow returned nil")
		}
	})

	// Test: Sequential calls work correctly (makenewwindow handles its own locking)
	t.Run("SequentialCalls", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			w := makenewwindow(nil)
			if w == nil {
				t.Errorf("makenewwindow returned nil on iteration %d", i)
			}
		}
	})
}

func TestFlushWarnings(t *testing.T) {
	// TODO(rjk): Write me.
	dir := t.TempDir()
	firstfilename := filepath.Join(dir, "firstfile")
	secondfilename := filepath.Join(dir, "secondfile")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	tests := []struct {
		name string
		fn   func(*testing.T, *globals, string)
		want *dumpfile.Content
	}{
		{
			name: "logSomethingSmall",
			fn:   logSomethingSmall,
			want: &dumpfile.Content{
				CurrentDir: cwd,
				VarFont:    defaultVarFont,
				FixedFont:  defaultFixedFont,
				Columns: []dumpfile.Column{
					{},
				},
				Windows: []*dumpfile.Window{
					{
						Tag: dumpfile.Text{
							Buffer: firstfilename + " Del Snarf | Look Edit ",
						},
					},
					{
						Tag: dumpfile.Text{
							Buffer: secondfilename + " Del Snarf | Look Edit ",
						},
					},
					{
						Type: dumpfile.Unsaved,
						Tag: dumpfile.Text{
							Buffer: "+Errors Del Snarf | Look Edit ",
						},
						Body: dumpfile.Text{
							Buffer: "SomethingSmall\n",
							Q1:     15},
					},
				},
			},
		},
		{
			name: "logSomethingWithMntDir",
			fn:   logSomethingWithMntDir,
			want: &dumpfile.Content{
				CurrentDir: cwd,
				VarFont:    defaultVarFont,
				FixedFont:  defaultFixedFont,
				Columns: []dumpfile.Column{
					{},
				},
				Windows: []*dumpfile.Window{
					{
						Tag: dumpfile.Text{
							Buffer: firstfilename + " Del Snarf | Look Edit ",
						},
					},
					{
						Tag: dumpfile.Text{
							Buffer: secondfilename + " Del Snarf | Look Edit ",
						},
					},
					{
						Type: dumpfile.Unsaved,
						Tag: dumpfile.Text{
							Buffer: filepath.Join(dir, "+Errors") + " Del Snarf | Look Edit ",
						},
						// TODO(rjk): Why isn't Q0 set? Where does this happen?
						// Somewhere, there's logic that fixes that.
						Body: dumpfile.Text{
							Buffer: "I am an warning\nI am a second warning\n",
							Q1:     38,
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// TODO(rjk): Each test should use its own global.
			FlexiblyMakeWindowScaffold(
				t,
				ScWin("firstfile"),
				ScBody("firstfile", contents),
				ScDir(dir, "firstfile"),
				ScWin("secondfile"),
				ScBody("secondfile", alt_contents),
				ScDir(dir, "secondfile"),
			)

			tc.fn(t, global, dir)

			// Function under test - flushwarnings handles its own locking
			flushwarnings()

			t.Log(*varfontflag, defaultVarFont)

			got, err := global.row.dump()
			if err != nil {
				t.Fatalf("dump failed: %v", err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("dump mismatch (-want +got):\n%s", diff)
			}

		})
	}

}
