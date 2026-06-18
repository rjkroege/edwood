package frame

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rjkroege/edwood/edwoodtest"
)

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

// generateVisualizedOutput writes the SVG trial file without comparing
// to a baseline. Used for known-failing tests that document a bug.
func generateVisualizedOutput(t *testing.T, fr Frame) {
	t.Helper()
	oname := makeVisualizedOutputTestPath(t)
	sf, err := os.Create(oname)
	if err != nil {
		t.Fatalf("can't make a file for the test output %s: %v", oname, err)
	}
	if err := gdo(t, fr).SVGDrawOps(sf); err != nil {
		t.Fatalf("can't write a file for the test output %s: %v", oname, err)
	}
	sf.Close()
}

// pixelPNGPath returns the path for a before/after PNG file.
// Within a subtest "TestInsert/simpleInsertShortString", name="simpleInsertShortString"
// and suffix="before" yields "testdata/TestInsert/simpleInsertShortString_before.png".
func pixelPNGPath(t *testing.T, name, suffix string) string {
	t.Helper()
	dir := filepath.Join("testdata", filepath.Dir(t.Name()))
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("pixelPNGPath mkdir %s: %v", dir, err)
	}
	return filepath.Join(dir, name+"_"+suffix+".png")
}

// snapBeforePNG writes the current pixel state of the screen as the "before"
// PNG. Call it before gdo.Clear() and the operation under test.
func snapBeforePNG(t *testing.T, fr Frame, name string) {
	t.Helper()
	path := pixelPNGPath(t, name, "before")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("snapBeforePNG create %s: %v", path, err)
	}
	defer f.Close()
	if err := gdo(t, fr).ScreenImageAsPNG(f); err != nil {
		t.Fatalf("snapBeforePNG write %s: %v", path, err)
	}
}

// snapAfterPNG writes the current pixel state of the screen (the "after" image)
// as a PNG file.
func snapAfterPNG(t *testing.T, fr Frame, name string) {
	t.Helper()
	path := pixelPNGPath(t, name, "after")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("snapAfterPNG create %s: %v", path, err)
	}
	defer f.Close()
	if err := gdo(t, fr).ScreenImageAsPNG(f); err != nil {
		t.Fatalf("snapAfterPNG write %s: %v", path, err)
	}
}

// visualizedoutputtest generates SVG-based graphical output
func visualizedoutputtest(t *testing.T, fr Frame) {
	t.Helper()
	oname := makeVisualizedOutputTestPath(t)
	sf, err := os.Create(oname)
	if err != nil {
		t.Fatalf("can't make a file for the test output %s: %v", oname, err)
	}
	if err := gdo(t, fr).SVGDrawOps(sf); err != nil {
		t.Fatalf("can't write a file for the test output %s: %v", oname, err)
	}
	sf.Close()

	// Compare the generated SVG to the baseline.
	compareVisualizedOutputTestToBaseline(t)

}
