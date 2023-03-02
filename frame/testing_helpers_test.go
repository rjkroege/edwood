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
// the baseline. f
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
