package regexp

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func readLargeFile(b testing.TB, numcopies int) []rune {
	b.Helper()

	// TODO(rjk): copy the test data into this directory?
	fname := "../testdata/hello.go"

	f, err := os.ReadFile(fname)
	if err != nil {
		b.Fatalf("can't read %q: %v", fname, err)
	}

	littlefile := []rune(string(f))
	bigfile := make([]rune, 0)

	for i := 0; i < numcopies; i++ {
		bigfile = append(bigfile, littlefile...)
	}
	return bigfile
}

func makeRe(b testing.TB) *Regexp {
	b.Helper()
	re, err := CompileAcme("main")
	if err != nil {
		b.Fatalf("can't complie rexgp %q: %v", "main", err)
	}
	return re
}

func BenchmarkFindForward(b *testing.B) {
	r := readLargeFile(b, 100)
	re := makeRe(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := re.FindForward(r, 0, len(r), -1)
		if got, want := len(matches), 2*100; got != want {
			b.Errorf("wrong # of matches got %d want %d", got, want)
		}
	}
}

func BenchmarkFindBackward(b *testing.B) {
	r := readLargeFile(b, 100)
	re := makeRe(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := re.oldFindBackward(r, 0, len(r), -1)
		if got, want := len(matches), 2*100; got != want {
			b.Errorf("wrong # of matches got %d want %d", got, want)
		}
	}
}

func BenchmarkNewFindBackward(b *testing.B) {
	r := readLargeFile(b, 100)
	re := makeRe(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := re.newFindBackward(r, 0, len(r), -1)
		if got, want := len(matches), 2*100; got != want {
			b.Errorf("wrong # of matches got %d want %d", got, want)
		}
	}
}

func BenchmarkNewFindBackwardOne(b *testing.B) {
	r := readLargeFile(b, 100)
	re := makeRe(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := re.newFindBackward(r, 0, len(r), 1)
		if got, want := len(matches), 1; got != want {
			b.Errorf("wrong # of matches got %d want %d", got, want)
		}
	}
}

func BenchmarkOldFindBackwardOne(b *testing.B) {
	r := readLargeFile(b, 100)
	re := makeRe(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches := re.oldFindBackward(r, 0, len(r), 1)
		if got, want := len(matches), 1; got != want {
			b.Errorf("wrong # of matches got %d want %d", got, want)
		}
	}
}

func TestFindForward(t *testing.T) {
	r := readLargeFile(t, 10)
	re := makeRe(t)
	t.Log(string(r))

	matches := re.FindForward(r, 0, len(r), -1)
	if got, want := len(matches), 2*10; got != want {
		t.Errorf("wrong # of matches got %d want %d", got, want)
	}
	res := [][]int{
		{8, 12},
		{33, 37},
		{78, 82},
		{103, 107},
		{148, 152},
		{173, 177},
		{218, 222},
		{243, 247},
		{288, 292},
		{313, 317},
		{358, 362},
		{383, 387},
		{428, 432},
		{453, 457},
		{498, 502},
		{523, 527},
		{568, 572},
		{593, 597},
		{638, 642},
		{663, 667},
	}
	if diff := cmp.Diff(res, matches); diff != "" {
		t.Errorf("dump mismatch (-want +got):\n%s", diff)
	}

}

func TestOldFindBackward(t *testing.T) {
	r := readLargeFile(t, 10)
	re := makeRe(t)
	t.Log(string(r))
	matches := re.oldFindBackward(r, 0, len(r), -1)
	if got, want := len(matches), 2*10; got != want {
		t.Errorf("wrong # of matches got %d want %d", got, want)
	}

	res := [][]int{
		{663, 667},
		{638, 642},
		{593, 597},
		{568, 572},
		{523, 527},
		{498, 502},
		{453, 457},
		{428, 432},
		{383, 387},
		{358, 362},
		{313, 317},
		{288, 292},
		{243, 247},
		{218, 222},
		{173, 177},
		{148, 152},
		{103, 107},
		{78, 82},
		{33, 37},
		{8, 12},
	}
	if diff := cmp.Diff(res, matches); diff != "" {
		t.Errorf("dump mismatch (-want +got):\n%s", diff)
	}
}

func TestNewFindBackward(t *testing.T) {
	r := readLargeFile(t, 10)
	re := makeRe(t)
	t.Log(string(r))
	matches := re.newFindBackward(r, 0, len(r), -1)
	if got, want := len(matches), 10*2; got != want {
		t.Errorf("wrong # of matches got %d want %d", got, want)
	}

	res := [][]int{
		{663, 667},
		{638, 642},
		{593, 597},
		{568, 572},
		{523, 527},
		{498, 502},
		{453, 457},
		{428, 432},
		{383, 387},
		{358, 362},
		{313, 317},
		{288, 292},
		{243, 247},
		{218, 222},
		{173, 177},
		{148, 152},
		{103, 107},
		{78, 82},
		{33, 37},
		{8, 12},
	}
	if diff := cmp.Diff(res, matches); diff != "" {
		t.Errorf("dump mismatch (-want +got):\n%s", diff)
	}
}

func TestNewFindBackwardSlice(t *testing.T) {
	r := readLargeFile(t, 10)
	re := makeRe(t)
	t.Log(string(r))
	matches := re.newFindBackward(r, 0, len(r), 1)
	if got, want := len(matches), 1; got != want {
		t.Errorf("wrong # of matches got %d want %d", got, want)
	}

	res := [][]int{
		{663, 667},
	}
	if diff := cmp.Diff(res, matches); diff != "" {
		t.Errorf("dump mismatch (-want +got):\n%s", diff)
	}
}
