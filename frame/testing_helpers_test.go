package frame

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rjkroege/edwood/edwoodtest"
)

var updatePNGs = flag.Bool("updatepngs", false, "Overwrite PNG golden baselines with current output")

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

// pixelPNGPath returns a path inside testdata/<TestName>/ for a PNG file
// with the given suffix, e.g. "before_trial" or "after_golden".
func pixelPNGPath(t *testing.T, name, suffix string) string {
	t.Helper()
	dir := filepath.Join("testdata", filepath.Dir(t.Name()))
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("pixelPNGPath mkdir %s: %v", dir, err)
	}
	return filepath.Join(dir, name+"_"+suffix+".png")
}

// snapBeforePNG writes the current pixel state as a trial "before" PNG,
// compares it to the committed golden, and fails the test on mismatch.
// Call it before gdo.Clear() and the operation under test.
func snapBeforePNG(t *testing.T, fr Frame, name string) {
	t.Helper()
	writeAndComparePNG(t, fr,
		pixelPNGPath(t, name, "before_trial"),
		pixelPNGPath(t, name, "before_golden"),
	)
}

// snapAfterPNG writes the current pixel state as a trial "after" PNG,
// compares it to the committed golden, and fails the test on mismatch.
func snapAfterPNG(t *testing.T, fr Frame, name string) {
	t.Helper()
	writeAndComparePNG(t, fr,
		pixelPNGPath(t, name, "after_trial"),
		pixelPNGPath(t, name, "after_golden"),
	)
}

// writeAndComparePNG writes the current screen image to trialPath, then
// compares it byte-for-byte with goldenPath.  On mismatch the test fails
// and the trial file is left on disk for inspection.  On match the trial
// file is removed.  When -updatepngs is set the golden is overwritten with
// the trial instead of being compared.
func writeAndComparePNG(t *testing.T, fr Frame, trialPath, goldenPath string) {
	t.Helper()

	f, err := os.Create(trialPath)
	if err != nil {
		t.Fatalf("writeAndComparePNG create %s: %v", trialPath, err)
	}
	if err := gdo(t, fr).ScreenImageAsPNG(f); err != nil {
		f.Close()
		t.Fatalf("writeAndComparePNG write %s: %v", trialPath, err)
	}
	f.Close()

	if *updatePNGs {
		trial, err := os.ReadFile(trialPath)
		if err != nil {
			t.Fatalf("writeAndComparePNG read trial %s: %v", trialPath, err)
		}
		if err := os.WriteFile(goldenPath, trial, 0644); err != nil {
			t.Fatalf("writeAndComparePNG write golden %s: %v", goldenPath, err)
		}
		os.Remove(trialPath)
		return
	}

	golden, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		t.Logf("no PNG golden at %s; run with -updatepngs to create", goldenPath)
		return
	}
	if err != nil {
		t.Fatalf("writeAndComparePNG read golden %s: %v", goldenPath, err)
	}

	trial, err := os.ReadFile(trialPath)
	if err != nil {
		t.Fatalf("writeAndComparePNG read trial %s: %v", trialPath, err)
	}

	if bytes.Equal(golden, trial) {
		os.Remove(trialPath)
		return
	}

	// Images differ.  Skip the failure when the CJK fallback font is
	// unavailable: goldens committed from macOS use Arial Unicode for
	// CJK chars; systems without it produce .notdef boxes and will always
	// differ from those goldens.
	if !edwoodtest.HasCJKFallback() {
		t.Logf("skipping PNG comparison for %s: no system CJK font (golden requires Arial Unicode)", goldenPath)
		os.Remove(trialPath)
		return
	}

	t.Errorf("PNG mismatch: %s differs from golden %s; run with -updatepngs to regenerate",
		trialPath, goldenPath)
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
