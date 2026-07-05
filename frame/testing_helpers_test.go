package frame

import (
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rjkroege/edwood/edwoodtest"
)

var rebase = flag.Bool("rebase", false, "overwrite SVG baselines with the current trial output")

// Code needed to help write tests.

// testName creates the correct name for the visualized test output.
func testName(t *testing.T, suffix string) string {
	return filepath.Join("testdata", t.Name()) + suffix + ".html"
}

func makeVisualizedOutputTestPath(t *testing.T) string {
	t.Helper()

	tp := testName(t, "_trial")
	dir := filepath.Dir(tp)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("can't make makeVisualizedOutputTestPath %s: %v", dir, err)
	}

	return tp
}

// compareVisualizedOutputTestToBaseline compares the generated SVG to
// the baseline. The generated file is <blah>_trial.html in the testdata directory.
func compareVisualizedOutputTestToBaseline(t *testing.T) {
	t.Helper()

	// load the base
	baselinename := testName(t, "")
	want := ""
	if b, err := os.ReadFile(baselinename); err != nil {
		t.Errorf("baseline unreadable for %s", baselinename)
		return
	} else {
		want = string(b)
	}

	testoutname := testName(t, "_trial")
	got := ""
	if b, err := os.ReadFile(testoutname); err != nil {
		t.Errorf("test result unreadable for %s", testoutname)
		return
	} else {
		got = string(b)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("visualized output mismatch (-want +got):\n%s", diff)
	} else {

		if err := os.RemoveAll(testoutname); err != nil {
			t.Errorf("can't remove valid output %s: %v", testoutname, err)
		}
	}
}

func gdo(t *testing.T, fr Frame) edwoodtest.GettableDrawOps {
	t.Helper()
	frimpl := fr.(*frameimpl)
	gdo := frimpl.display.(edwoodtest.GettableDrawOps)
	return gdo
}

// findModuleRoot walks up from dir until it finds a go.mod file.
func findModuleRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// sourceRelFile returns the repo-relative slash path of the file at the
// given runtime.Caller skip depth, relative to the module root.
func sourceRelFile(skip int) string {
	_, absFile, _, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	root := findModuleRoot(filepath.Dir(absFile))
	if root == "" {
		return absFile
	}
	rel, err := filepath.Rel(root, absFile)
	if err != nil {
		return absFile
	}
	return filepath.ToSlash(rel)
}

// generateVisualizedOutput writes the SVG trial file without comparing
// to a baseline. Used for known-failing tests that document a bug.
func generateVisualizedOutput(t *testing.T, fr Frame) {
	t.Helper()
	oname := makeVisualizedOutputTestPath(t)
	sf, err := os.Create(oname)
	if err != nil {
		t.Fatalf("can't make a file for the test output %s: %v", oname, err)
	}
	if err := gdo(t, fr).SVGDrawOps(sf, t.Name(), sourceRelFile(2)); err != nil {
		t.Fatalf("can't write a file for the test output %s: %v", oname, err)
	}
	sf.Close()
}

// visualizedoutputtest generates SVG-based graphical output
func visualizedoutputtest(t *testing.T, fr Frame) {
	t.Helper()
	oname := makeVisualizedOutputTestPath(t)
	sf, err := os.Create(oname)
	if err != nil {
		t.Fatalf("can't make a file for the test output %s: %v", oname, err)
	}
	if err := gdo(t, fr).SVGDrawOps(sf, t.Name(), sourceRelFile(2)); err != nil {
		t.Fatalf("can't write a file for the test output %s: %v", oname, err)
	}
	sf.Close()

	if *rebase {
		baseline := testName(t, "")
		if err := os.Rename(oname, baseline); err != nil {
			t.Fatalf("rebase: can't promote %s to %s: %v", oname, baseline, err)
		}
		return
	}

	// Compare the generated SVG to the baseline.
	compareVisualizedOutputTestToBaseline(t)
}
