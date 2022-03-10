package file

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestOverall(t *testing.T) {
	b := NewBufferNoNr(nil)
	b.checkPiecesCnt(t, 2)
	b.checkContent("#0", t, "")

	b.insertString(0, "", t)
	b.checkPiecesCnt(t, 2)
	b.checkContent("#1", t, "")

	b.insertString(0, "All work ウクラ makes John a dull boy", t)
	b.checkPiecesCnt(t, 3)
	b.checkContent("#2", t, "All work ウクラ makes John a dull boy")

	b.insertString(9, "and no playing ", t)
	b.checkPiecesCnt(t, 6)
	b.checkContent("#3", t, "All work and no playing ウクラ makes John a dull boy")

	b.SetUndoPoint()
	// Also check that multiple change commits don't create empty changes.
	b.SetUndoPoint()
	b.deleteCreateOffsetTuple(20, 18, t)
	b.checkContent("#4", t, "All work and no play a dull boy")

	b.insertString(20, " makes Jack", t)
	b.checkContent("#5", t, "All work and no play makes Jack a dull boy")

	b.Undo(0)
	b.checkContent("#6", t, "All work and no play a dull boy")
	b.Undo(0)
	b.checkContent("#7", t, "All work and no playing ウクラ makes John a dull boy")
	b.Undo(0)
	b.checkContent("#8", t, "All work ウクラ makes John a dull boy")

	b.Redo(0)
	b.checkContent("#9", t, "All work and no playing ウクラ makes John a dull boy")
	b.Redo(0)
	b.checkContent("#10", t, "All work and no play a dull boy")
	b.Redo(0)
	b.checkContent("#11", t, "All work and no play makes Jack a dull boy")
	b.Redo(0)
	b.checkContent("#12", t, "All work and no play makes Jack a dull boy")
}

func TestCacheInsertAndDelete(t *testing.T) {
	b := NewBufferNoNr([]byte("testing insertation"))
	b.checkPiecesCnt(t, 3)
	b.checkContent("#0", t, "testing insertation")

	b.cacheInsertString(8, "caching", t)
	b.checkPiecesCnt(t, 6)
	b.checkContent("#1", t, "testing cachinginsertation")

	b.cacheInsertString(15, " ", t)
	b.checkPiecesCnt(t, 6)
	b.checkContent("#2", t, "testing caching insertation")

	b.cacheDelete(12, 3, t)
	b.checkPiecesCnt(t, 6)
	b.checkContent("#3", t, "testing cach insertation")

	b.cacheInsertString(12, "ed", t)
	b.checkPiecesCnt(t, 6)
	b.checkContent("#4", t, "testing cached insertation")
}

func TestSimulateBackspace(t *testing.T) {
	b := NewBufferNoNr([]byte("apples and oranges"))
	for i := 5; i > 0; i-- {
		b.cacheDelete(i, 1, t)
	}
	b.checkContent("#0", t, "a and oranges")
	b.Undo(0)
	b.checkContent("#1", t, "apples and oranges")
}

func TestSimulateDeleteKey(t *testing.T) {
	b := NewBufferNoNr([]byte("apples and oranges"))
	for i := 0; i < 4; i++ {
		b.cacheDelete(7, 1, t)
	}
	b.checkContent("#0", t, "apples oranges")
	b.Undo(0)
	b.checkContent("#1", t, "apples and oranges")
}

func TestDelete(t *testing.T) {
	b := NewBufferNoNr([]byte("and what is a dream?"))
	b.insertString(9, "exactly ", t)
	b.checkContent("#0", t, "and what exactly is a dream?")

	b.delete(22, 2000, t)
	b.checkContent("#1", t, "and what exactly is a ")
	b.insertString(22, "joke?", t)
	b.checkContent("#2", t, "and what exactly is a joke?")

	cases := []struct {
		off, len int
		expected string
	}{
		{9, 8, "and what is a joke?"},
		{9, 13, "and what joke?"},
		{5, 6, "and wactly is a joke?"},
		{9, 14, "and what oke?"},
		{11, 3, "and what exly is a joke?"},
	}
	for _, c := range cases {
		b.delete(c.off, c.len, t)
		b.checkContent("#3", t, c.expected)
		b.Undo(0)
		b.checkContent("#4", t, "and what exactly is a joke?")
	}
}

func TestDeleteAtTheEndOfCachedPiece(t *testing.T) {
	b := NewBufferNoNr([]byte("Original data."))
	b.cacheInsertString(8, ",", t)
	b.cacheDelete(9, 1, t)
	b.checkContent("#0", t, "Original,data.")
	b.Undo(0)
	b.checkContent("#1", t, "Original data.")
}

func TestGroupChanges(t *testing.T) {
	b := NewBufferNoNr([]byte("group 1, group 2, group 3"))
	b.checkPiecesCnt(t, 3)
	// b.GroupChanges()

	b.cacheDelete(0, 6, t)
	b.checkContent("#0", t, "1, group 2, group 3")

	b.cacheDelete(3, 6, t)
	b.checkContent("#1", t, "1, 2, group 3")

	b.cacheDelete(6, 6, t)
	b.checkContent("#2", t, "1, 2, 3")

	b.Undo(0)
	b.checkContent("#3", t, "group 1, group 2, group 3")
	b.Undo(0)
	b.checkContent("#4", t, "group 1, group 2, group 3")

	b.Redo(0)
	b.checkContent("#5", t, "1, 2, 3")
	b.Redo(0)
	b.checkContent("#6", t, "1, 2, 3")
}

func TestSaving(t *testing.T) {
	b := NewBufferNoNr(nil)

	b.checkModified(t, 1, false)
	b.insertString(0, "stars can frighten", t)
	b.checkModified(t, 2, true)

	b.Clean()
	b.checkModified(t, 3, false)

	b.Undo(0)
	b.checkModified(t, 4, true)
	b.Redo(0)
	b.checkModified(t, 5, false)

	b.insertString(0, "Neptun, Titan, ", t)
	b.checkModified(t, 6, true)
	b.Undo(0)
	b.checkModified(t, 7, false)

	b.Redo(0)
	b.checkModified(t, 8, true)

	b.Clean()
	b.checkModified(t, 9, false)

	b = NewBufferNoNr([]byte("my book is closed"))
	b.checkModified(t, 10, false)

	b.insertString(17, ", I read no more", t)
	b.checkModified(t, 11, true)
	b.Undo(0)
	b.checkModified(t, 12, false)

	b.Redo(0)
	b.Clean()
	b.checkModified(t, 13, false)

	b.Undo(0)
	b.Clean()
	b.checkModified(t, 14, false)
}

func TestReader(t *testing.T) {
	b := NewBufferNoNr(nil)
	b.insertString(0, "So many", t)
	b.insertString(7, " books,", t)
	b.insertString(14, " so little", t)
	b.insertString(24, " time.", t)
	b.checkContent("#0", t, "So many books, so little time.")

	cases := []struct {
		off, len int
		expected string
		err      error
	}{
		{0, 7, "So many", nil},
		{1, 11, "o many book", nil},
		{8, 4, "book", nil},
		{15, 20, "so little time.", io.EOF},
	}

	for _, c := range cases {
		data := make([]byte, c.len)
		n, err := b.ReadAt(data, int64(c.off))
		if err != c.err {
			t.Errorf("expected error %v, got %v", c.err, err)
		}
		if n != len(c.expected) {
			t.Errorf("n should be %d, got %d", len(c.expected), n)
		}
		if !bytes.Equal(data[:n], []byte(c.expected)) {
			t.Errorf("got '%s', want '%s'", data[:n], c.expected)
		}
	}
}

func TestBufferSize(t *testing.T) {
	b := NewBufferNoNr(nil)
	tests := []struct {
		action func()
		want   int
	}{
		0: {func() {}, 0},
		1: {func() { b.insertString(0, " Like", t) }, 5},
		2: {func() { b.insertString(0, " Colour", t) }, 12},
		3: {func() { b.insertString(7, " You", t) }, 16},
		4: {func() { b.delete(5, 1, t) }, 15},
		5: {func() { b.insertString(0, "Pink is the", t) }, 26},
		6: {func() { b.Undo(0) }, 15},
		7: {func() { b.Redo(0) }, 26},
	}

	for i, tt := range tests {
		tt.action()
		if got := b.Size(); got != tt.want {
			t.Fatalf("%d: got %d, want %d", i, got, tt.want)
		}
	}
}

func TestUndoRedoReturnedOffsets(t *testing.T) {
	b := NewBufferNoNr(nil)
	insert := func(off, len int) {
		b.insertString(off, strings.Repeat(".", len), t)
	}
	insert(0, 7)
	insert(7, 5)
	insert(12, 9)
	b.delete(8, 8, t)
	insert(3, 19)
	b.delete(0, 20, t)

	undo, redo := (*Buffer).Undo, (*Buffer).Redo
	tests := []struct {
		op      func(*Buffer, int) (int, int, bool, int)
		wantOff int
		wantN   int
	}{
		0:  {redo, -1, 0},
		1:  {undo, 0, 20},
		2:  {undo, 3, -16},
		3:  {undo, 8, 16},
		4:  {undo, 12, 3},
		5:  {undo, 7, 2},
		6:  {undo, 0, -7},
		7:  {undo, -1, 0},
		8:  {redo, 0, 7},
		9:  {redo, 7, 12},
		10: {redo, 12, 21},
		11: {redo, 8, 0},
		12: {redo, 3, 22},
		13: {redo, 0, -20},
		14: {redo, -1, 0},
	}

	for i, tt := range tests {
		off, n, _, _ := tt.op(b, 0)
		if off != tt.wantOff {
			t.Errorf("%d: got offset %d, want %d", i, off, tt.wantOff)
		}
		if n != tt.wantN {
			t.Errorf("%d: got n %d, want %d", i, n, tt.wantN)
		}
	}
}

func TestPieceNr(t *testing.T) {
	b := NewBufferNoNr(nil)
	manderianBytes := []byte("痛苦本身可能是很多痛苦, 但主要的原因是痛苦, 但我给它时间陷入这种痛苦, 以至于有些巨大的痛苦")
	eng1 := []byte("Lorem ipsum in Mandarin")
	eng2 := []byte("This is the")
	eng3 := []byte("In the midst")

	b.insertCreateOffsetTuple(0, manderianBytes, t)
	b.checkContent("TestPieceNr: First insert", t, string(manderianBytes))

	b.insertCreateOffsetTuple(b.Nr(), eng1, t)
	b.checkContent("TestPieceNr: Second insert", t, string(manderianBytes)+string(eng1))

	b.insertCreateOffsetTuple(0, eng2, t)
	buffAfterInserts := string(eng2) + string(manderianBytes) + string(eng1)
	b.checkContent("TestPieceNr: third insert", t, buffAfterInserts)

	t.Logf("Before delete: %v\n", string(b.Bytes()))

	b.deleteCreateOffsetTuple(13, 10, t) // Currently, the offset translates to 17 (should be 20). Should be deleting a total of 25 bytes

	buffAfterDelete := []rune(buffAfterInserts)
	buffAfterDelete = append(buffAfterDelete[:13], buffAfterDelete[23:]...)
	b.checkContent("TestPieceNr: after 1 delete", t, string(buffAfterDelete))

	b.insertCreateOffsetTuple(8, eng3, t)
	buffAfterDelete = append(buffAfterDelete[:8], append([]rune(string(eng3)), buffAfterDelete[8:]...)...)
	b.checkContent("TestPieceNr: after everything", t, string(buffAfterDelete))

	undo, redo := (*Buffer).Undo, (*Buffer).Redo
	tests := []struct {
		op func(*Buffer, int) (int, int, bool, int)
	}{
		0:  {redo},
		1:  {undo},
		2:  {undo},
		3:  {undo},
		4:  {undo},
		5:  {undo},
		6:  {undo},
		7:  {undo},
		8:  {redo},
		9:  {redo},
		10: {redo},
		11: {redo},
		12: {redo},
		13: {redo},
		14: {redo},
	}

	for i, tt := range tests {
		t.Run("TestPieceNr #"+fmt.Sprint(i), func(t *testing.T) {
			tt.op(b, 0)
			nr := b.Nr()
			wantNr := countRunes(t, b)
			if nr != wantNr {
				t.Errorf("%d: got n %d, want %d", i, nr, wantNr)
			}
		})
	}
}

func (b *Buffer) checkPiecesCnt(t *testing.T, expected int) {
	t.Helper()
	if b.piecesCnt != expected {
		t.Errorf("got %d pieces, want %d", b.piecesCnt, expected)
	}
}

func (b *Buffer) checkContent(name string, t *testing.T, expected string) {
	t.Helper()
	c := string(b.Bytes())
	if c != expected {
		t.Errorf("%s: got '%s', want '%s'", name, c, expected)
	}

	actualNr := b.Nr()
	expectedNr := (utf8.RuneCountInString(expected))
	if actualNr != expectedNr {
		t.Errorf("%v: got '%v' runes, expected '%v' runes", name, actualNr, expectedNr)
	}

	if got, want := b.Size(), len(expected); got != want {
		t.Errorf("%v: b.Size() got %d, want %d", name, got, want)
	}
}

func (b *Buffer) insertString(off int, data string, t *testing.T) {
	b.SetUndoPoint()
	b.cacheInsertString(off, data, t)
}

func (b *Buffer) cacheInsertString(off int, data string, t *testing.T) {
	err := b.insertCreateOffsetTuple((off), []byte(data), t)
	if err != nil {
		panic(err)
	}
}

func (b *Buffer) delete(off, length int, t *testing.T) {
	b.SetUndoPoint()
	b.cacheDelete(off, length, t)
}

func (b *Buffer) cacheDelete(off, length int, t *testing.T) {
	b.deleteCreateOffsetTuple(off, length, t)
}

func TestRuneTuple(t *testing.T) {
	tt := []struct {
		name    string
		buf     []string
		roff    int
		bwant   int
		preops  []int
		bviewed int
	}{
		{
			name:    "zero buf, start",
			buf:     []string{},
			roff:    0,
			bwant:   0,
			bviewed: 1,
		},
		{
			name:    "one buf, start",
			buf:     []string{"foo"},
			roff:    0,
			bwant:   0,
			bviewed: 1,
		},
		{
			name:    "one buf, middle",
			buf:     []string{"foo"},
			roff:    1,
			bwant:   1,
			bviewed: 3,
		},
		{
			name:    "one buf, end",
			buf:     []string{"foo"},
			roff:    2,
			bwant:   2,
			bviewed: 3,
		},
		{
			name:    "one buf, not-ASCII, mid",
			buf:     []string{"a痛苦本身"},
			roff:    2,
			bwant:   len("a痛"),
			bviewed: 3,
		},
		{
			name:    "one buf, not-ASCII, end",
			buf:     []string{"痛苦本身"},
			roff:    3,
			bwant:   len("痛苦本"),
			bviewed: 3,
		},
		{
			name:    "one buf, not-ASCII, past-end",
			buf:     []string{"痛苦本身"},
			roff:    4,
			bwant:   len("痛苦本身"),
			bviewed: 3,
		},
		{
			name:    "one buf, not-ASCII, far past-end",
			buf:     []string{"痛苦本身"},
			roff:    1000,
			bwant:   len("痛苦本身"),
			bviewed: 2,
		},
		{
			name:    "three bufs, not-ASCII, start of middle piece",
			buf:     []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:    5,
			bwant:   len("痛苦本身痛"),
			bviewed: 4,
		},
		{
			name:    "three bufs, not-ASCII, end of middle piece",
			buf:     []string{"痛苦本身", "痛苦本", "痛苦痛苦本身"},
			roff:    7,
			bwant:   len("痛苦本身痛苦本"),
			bviewed: 4,
		},
		{
			name:    "three bufs, not-ASCII, start of end piece",
			buf:     []string{"痛苦本身", "痛苦本", "痛苦痛苦本身"},
			roff:    8,
			bwant:   len("痛苦本身痛苦本痛"),
			bviewed: 5,
		},
		{
			name:    "three bufs, not-ASCII, end of end piece",
			buf:     []string{"痛苦本身", "痛苦本", "痛苦痛苦本身"},
			roff:    13,
			bwant:   len("痛苦本身痛苦本痛苦痛苦本身"),
			bviewed: 5,
		},
		{
			// Count runes in xyz: echo -n xyz| u wc -m
			name:    "failing case in TestPieceNr",
			buf:     []string{"This is the", "痛苦本身可能是很多痛苦, 但主要的原因是痛苦, 但我给它时间陷入这种痛苦, 以至于有些巨大的痛苦Lorem ipsum in Mandarin"},
			roff:    23,
			bwant:   len("This is the痛苦本身可能是很多痛苦,"),
			bviewed: 4,
		},
		{
			name:    "one buf, not-ASCII, mid, preop 0",
			buf:     []string{"a痛苦本身"},
			roff:    2,
			bwant:   len("a痛"),
			preops:  []int{0},
			bviewed: 3,
		},
		{
			name:    "one buf, not-ASCII, mid, preop way past end",
			buf:     []string{"a痛苦本身"},
			roff:    2,
			bwant:   len("a痛"),
			preops:  []int{20},
			bviewed: 3,
		},
		{
			name:    "one buf, not-ASCII, mid, preop 1",
			buf:     []string{"a痛苦本身"},
			roff:    2,
			bwant:   len("a痛"),
			preops:  []int{1},
			bviewed: 3,
		},
		{
			name:    "one buf, not-ASCII, mid, preop 3",
			buf:     []string{"a痛苦本身"},
			roff:    2,
			bwant:   len("a痛"),
			preops:  []int{3},
			bviewed: 3,
		},
		{
			name:    "three bufs, not-ASCII, preop 3rd piece 8",
			buf:     []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:    1,
			bwant:   len("痛"),
			preops:  []int{8},
			bviewed: 3,
		},
		{
			name:    "three bufs, not-ASCII, preop 3rd piece 8, back to 5",
			buf:     []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:    5,
			bwant:   len("痛苦本身痛"),
			bviewed: 4,
			preops:  []int{8},
		},
		{
			name:    "three bufs, not-ASCII, preop 3rd piece 8, back to 7",
			buf:     []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:    7,
			bwant:   len("痛苦本身痛ö本"),
			preops:  []int{8},
			bviewed: 4,
		},
		{
			name:    "three bufs, not-ASCII, preop 3rd piece 8, back to 6",
			buf:     []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:    6,
			bwant:   len("痛苦本身痛ö"),
			preops:  []int{8},
			bviewed: 4,
		},
		{
			name:    "three bufs, not-ASCII, preop 3rd piece 8, back to 2",
			buf:     []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:    2,
			bwant:   len("痛苦"),
			preops:  []int{8},
			bviewed: 3,
		},
		{
			name:    "three bufs, not-ASCII, preop 3rd piece pastend",
			buf:     []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:    1,
			bwant:   len("痛"),
			preops:  []int{50},
			bviewed: 3,
		},
		{
			name:    "three bufs, not-ASCII, preop 3rd piece pastend, back to 5",
			buf:     []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:    5,
			bwant:   len("痛苦本身痛"),
			bviewed: 4,
			preops:  []int{100},
		},
		{
			name: "three bufs, not-ASCII, preop 3rd piece pastend, back to 7",
			//
			buf:     []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:    7,
			bwant:   len("痛苦本身痛ö本"),
			preops:  []int{50},
			bviewed: 4,
		},
		{
			name:    "three bufs, not-ASCII, preop 3rd piece pastend, back to 6",
			buf:     []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:    6,
			bwant:   len("痛苦本身痛ö"),
			preops:  []int{50},
			bviewed: 4,
		},
		{
			name:    "three bufs, not-ASCII, preop 3rd piece pastend, back to 2",
			buf:     []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:    2,
			bwant:   len("痛苦"),
			preops:  []int{50},
			bviewed: 3,
		},
		{
			// NB: b.viewed is the piece that we would append an insert to.
			// Character 9 is in the next piece.
			name:    "failing case with single character structures",
			buf:     []string{"hello\n1 2", "X", " 3 4\nfoo"},
			roff:    9,
			bwant:   9,
			bviewed: 3,
			preops:  []int{8},
		},
	}
	for _, tv := range tt {
		t.Run(tv.name, func(t *testing.T) {
			// Test RuneTuple
			b := NewBufferNoNr(nil)
			for _, s := range tv.buf {
				b.insertString(b.Nr(), s, t)
			}
			b.checkPiecesCnt(t, 2+len(tv.buf))

			// New bi-directional RuneTuple depends on the state of previous
			// RuneTuple invocations to speed up operation. Run additional RuneTuple
			// invocations here for their side-effects.
			for _, d := range tv.preops {
				b.RuneTuple(d)
			}

			// t.Logf("test RuneTuple %d, %s", tv.roff, b.viewedState())
			gt := b.RuneTuple(tv.roff)
			// t.Logf("test RuneTuple result %v, %s", gt, b.viewedState())

			if got, want := gt.B, tv.bwant; got != want {
				t.Errorf("%s got %d != want %d", "byte", got, want)
			}

			if got, want := gt.R, tv.roff; got != want {
				t.Errorf("%s got %d != want %d", "rune", got, want)
			}

			if got, want := b.viewed.id, tv.bviewed; got != want {
				t.Errorf("%s got %d != want %d", "piece id", got, want)
			}
			// Test ByteTuple
			b = NewBufferNoNr(nil)
			for _, s := range tv.buf {
				b.insertString(b.Nr(), s, t)
			}
			b.checkPiecesCnt(t, 2+len(tv.buf))

			// ByteTuple depends on the state of previous
			// RuneTuple/ByteTuple invocations. Run some for their side-effects.
			for _, d := range tv.preops {
				b.RuneTuple(d)
			}

			// Skip the case where we are past the end.
			if tv.roff > tv.bwant {
				// This case is impossible for ByteTuple
				return
			}

			t.Logf("test ByteTuple %d, %s", tv.bwant, b.viewedState())
			gt = b.ByteTuple(tv.bwant)
			t.Logf("test ByteTuple result %v, %s", gt, b.viewedState())

			if got, want := gt.B, tv.bwant; got != want {
				t.Errorf("%s got %d != want %d", "byte", got, want)
			}

			if got, want := gt.R, tv.roff; got != want {
				t.Errorf("%s got %d != want %d", "rune", got, want)
			}

			if got, want := b.viewed.id, tv.bviewed; got != want {
				t.Errorf("%s got %d != want %d", "piece id", got, want)
			}
		})
	}
}

func (b *Buffer) checkModified(t *testing.T, id int, expected bool) {
	t.Helper()
	if b.Dirty() != expected {
		if expected {
			t.Errorf("#%d should be modified", id)
		} else {
			t.Errorf("#%d should not be modified", id)
		}
	}
}

func countRunes(t *testing.T, b *Buffer) int {
	t.Helper()
	return (utf8.RuneCount(b.Bytes()))
}

func NewBufferNoNr(content []byte) *Buffer {
	return NewBuffer(content, utf8.RuneCount(content))
}

func (b *Buffer) insertCreateOffsetTuple(off int, content []byte, t *testing.T) error {
	t.Helper()
	// t.Logf("insertCreateOffsetTuple before RuneTuple %d state %s", off, b.viewedState())

	start := b.RuneTuple(off)

	// t.Logf("insertCreateOffsetTuple after RuneTuple state %s", b.viewedState())

	err := b.Insert(start, content, utf8.RuneCount(content), 1)

	// t.Logf("insertCreateOffsetTuple after Insert state %s", b.viewedState())

	return err
}

func (b *Buffer) deleteCreateOffsetTuple(off, length int, t *testing.T) error {
	t.Helper()
	t.Logf("deleteCreateOffsetTuple before first RuneTuple %d state %s", off, b.viewedState())
	start := b.RuneTuple(off)
	t.Logf("deleteCreateOffsetTuple before second RuneTuple %d state %s", off+length, b.viewedState())
	end := b.RuneTuple(off + length)
	t.Logf("before calling Delete start %v, end %v, state %s", start, end, b.viewedState())

	return b.Delete(start, end, 1)
}

// RuneTuple creates a byte, rune offset pair (i.e. OffsetTuple) for a
// given offset in runes.
// TODO(rjk): Consider using the cached piece to speed this up.
func (b *Buffer) OldRuneTuple(off int) OffsetTuple {
	b.validateInvariant()

	tr, tb := 0, 0
	p := b.begin

	// Find piece
	for ; p != b.end && tr+p.nr < off; p = p.next {
		tr += p.nr
		tb += len(p.data)
	}

	// Find the byte offset in piece p
	for i := 0; tr < off; tr++ {
		_, sz := utf8.DecodeRune(p.data[i:])
		tb += sz
		i += sz
	}

	return OffsetTuple{
		B: tb,
		R: tr,
	}
}

func TestFindPiece(t *testing.T) {
	tt := []struct {
		name   string
		buf    []string
		roff   int
		preops []int
	}{
		{
			name: "zero buf, start",
			buf:  []string{},
			roff: 0,
		},
		{
			name: "one buf, start",
			buf:  []string{"foo"},
			roff: 0,
		},
		{
			name: "one buf, middle",
			buf:  []string{"foo"},
			roff: 1,
		},
		{
			name: "one buf, end",
			buf:  []string{"foo"},
			roff: 2,
		},
		{
			name: "one buf, not-ASCII, mid",
			buf:  []string{"a痛苦本身"},
			roff: 2,
		},
		{
			name: "one buf, not-ASCII, end",
			buf:  []string{"痛苦本身"},
			roff: 3,
		},
		{
			name: "one buf, not-ASCII, past-end",
			buf:  []string{"痛苦本身"},
			roff: 4,
		},
		{
			name: "one buf, not-ASCII, far past-end",
			buf:  []string{"痛苦本身"},
			roff: 1000,
		},
		{
			name: "three bufs, not-ASCII, start of middle piece",
			buf:  []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff: 5,
		},
		{
			name: "three bufs, not-ASCII, end of middle piece",
			buf:  []string{"痛苦本身", "痛苦本", "痛苦痛苦本身"},
			roff: 7,
		},
		{
			name: "three bufs, not-ASCII, start of end piece",
			buf:  []string{"痛苦本身", "痛苦本", "痛苦痛苦本身"},
			roff: 8,
		},
		{
			name: "three bufs, not-ASCII, end of end piece",
			buf:  []string{"痛苦本身", "痛苦本", "痛苦痛苦本身"},
			roff: 13,
		},
		{
			// Count runes in xyz: echo -n xyz| u wc -m
			name: "failing case in TestPieceNr",
			buf:  []string{"This is the", "痛苦本身可能是很多痛苦, 但主要的原因是痛苦, 但我给它时间陷入这种痛苦, 以至于有些巨大的痛苦Lorem ipsum in Mandarin"},
			roff: 23,
		},
		{
			name:   "one buf, not-ASCII, mid, preop 0",
			buf:    []string{"a痛苦本身"},
			roff:   2,
			preops: []int{0},
		},
		{
			name:   "one buf, not-ASCII, mid, preop way past end",
			buf:    []string{"a痛苦本身"},
			roff:   2,
			preops: []int{20},
		},
		{
			name:   "one buf, not-ASCII, mid, preop 1",
			buf:    []string{"a痛苦本身"},
			roff:   2,
			preops: []int{1},
		},
		{
			name:   "one buf, not-ASCII, mid, preop 3",
			buf:    []string{"a痛苦本身"},
			roff:   2,
			preops: []int{3},
		},
		{
			name:   "three bufs, not-ASCII, preop 3rd piece 8",
			buf:    []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:   1,
			preops: []int{8},
		},
		{
			name:   "three bufs, not-ASCII, preop 3rd piece 8, back to 5",
			buf:    []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:   5,
			preops: []int{8},
		},
		{
			name:   "three bufs, not-ASCII, preop 3rd piece 8, back to 7",
			buf:    []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:   7,
			preops: []int{8},
		},
		{
			name:   "three bufs, not-ASCII, preop 3rd piece 8, back to 6",
			buf:    []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:   6,
			preops: []int{8},
		},
		{
			name:   "three bufs, not-ASCII, preop 3rd piece 8, back to 2",
			buf:    []string{"痛苦本身", "痛ö本", "a苦痛苦本b"},
			roff:   2,
			preops: []int{8},
		},
		{
			name:   "failing case from TestOverall",
			buf:    []string{"All work ", "and no playing ", "ウクラ makes John a dull boy"},
			roff:   20,
			preops: []int{20, 20 + 18},
		},
	}
	for _, tv := range tt {
		t.Run(tv.name, func(t *testing.T) {
			b := NewBufferNoNr(nil)
			for _, s := range tv.buf {
				b.insertString(b.Nr(), s, t)
			}
			b.checkPiecesCnt(t, 2+len(tv.buf))

			// We need the targetoff that will be an argument to findPiece. Make
			// it here.
			targetoff := b.RuneTuple(tv.roff)

			// Invalidate the cached data to make sure that everything is consistent.
			b.invalidateCachedData()

			// New bi-directional findPiece depends on the state of previous findPiece and
			// RuneTuple invocations to speed up operation. Run additional RuneTuple
			// invocations here for their side-effects.
			for _, d := range tv.preops {
				b.RuneTuple(d)
			}

			t.Logf("test findPiece %v, %s", targetoff, b.viewedState())
			gotp, gotb, gotr := b.findPiece(targetoff)
			t.Logf("test findPiece after %s", b.viewedState())

			wantp, wantb, wantr := b.old_findPiece(targetoff)

			if got, want := gotp, wantp; got != want {
				t.Errorf("%s got %+v != want %+v", "piece", got, want)
			}

			if got, want := gotb, wantb; got != want {
				t.Errorf("%s got %d != want %d", "byte", got, want)
			}

			if got, want := gotr, wantr; got != want {
				t.Errorf("%s got %d != want %d", "rune", got, want)
			}
		})
	}
}

// old_findPiece returns the piece holding the text at the byte offset, the
// byte offset into piece and similarly for the rune offset. If off
// happens to be at a piece boundary (i.e. the first byte of a piece)
// then the previous piece to the left is returned with an offset of the
// piece's length.
//
// If off is zero, the beginning sentinel piece is returned.
func (b *Buffer) old_findPiece(off OffsetTuple) (*piece, int, int) {
	tr, tb := 0, 0
	for p := b.begin; p.next != nil; p = p.next {
		if tb <= off.B && off.B <= tb+p.len() {
			return p, off.B - tb, off.R - tr
		}
		tb += p.len()
		tr += p.nr
	}
	return nil, 0, 0
}

// invalidateCachedData clears cached data.
func (b *Buffer) invalidateCachedData() {
	b.viewed = nil
	b.pend = Ot(-1, -1)
}
