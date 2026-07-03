package frame

import (
	"image"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func deleteSingleCharacterAtLineEnd(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("0ab"), 0)
	gdo(t, fr).Clear()

	s := fr.Delete(2, 3)

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func deleteSingleCharacterInMiddle(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("0ab"), 0)
	gdo(t, fr).Clear()

	s := fr.Delete(1, 2)

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func deleteMultipleCharacterInMiddle(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	// Three logical lines; middle word of line 1 is bounded by spaces.
	fr.Insert([]rune("hello world and foo\nmore text here stuff\nbaz end done here"), 0)
	gdo(t, fr).Clear()

	// Delete "hello world and foo\nmore t" (10 chars bounded by spaces).
	p0 := len("hello world and foo\nmore ")
	s := fr.Delete(p0, p0+len("text"))

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func deleteNewlineTocreateWrappedLine(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("0ab\n1cd\n2ef"), 0)
	gdo(t, fr).Clear()

	s := fr.Delete(len("0ab"), len("0ab\n"))

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func rippleUpDeletedChar(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	// gdo(t, fr).Clear()
	fr.Insert([]rune("0ab1cd2ef"), 0)
	gdo(t, fr).Clear()

	s := fr.Delete(1, 2) // a

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func rippleUpDeletedCharWithNewline(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("0ab1cd\n2ef"), 0)
	gdo(t, fr).Clear()

	s := fr.Delete(1, 2) // a

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func deleteTab(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	t.Log(fr.GetMaxtab())

	// gdo(t, fr).Clear()
	fr.Insert([]rune("0	ab1cd2ef"), 0)
	gdo(t, fr).Clear()

	s := fr.Delete(1, 2) // the tab

	if got, want := s, 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func deleteCharBeforeTab(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	t.Log(fr.GetMaxtab())

	// gdo(t, fr).Clear()
	fr.Insert([]rune("0a	b1cd2ef"), 0)
	gdo(t, fr).Clear()

	s := fr.Delete(1, 2) // a

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func rippleUpMultiLine(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	// gdo(t, fr).Clear()
	fr.Insert([]rune("0a\nb1\ncd2\nef"), 0)
	gdo(t, fr).Clear()

	s := fr.Delete(0, 6) // 0a\nb1\n

	if got, want := s, 2; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// deleteOnlyCharFromLine removes "c", the only visible character on line 2 of
// "ab\nc\nef". The trailing '\n' is kept, so line 2 becomes empty but no
// visual line disappears and the return value is 0.
func deleteOnlyCharFromLine(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("ab\nc\nef"), 0)
	gdo(t, fr).Clear()

	// Delete 'c' at position 3; '\n' at position 4 stays, keeping line 2
	// as a distinct (empty) visual line.
	s := fr.Delete(3, 4)

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// deleteDefBetweenEmptyLines removes "def" from "abc\n\ndef\n\nghi\n".
// The preceding and following blank lines both stay; "def"'s visual row
// becomes empty and "ghi" ripples up, but no visual line count changes.
func deleteDefBetweenEmptyLines(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	// 6-row frame holds all of "abc\n\ndef\n\nghi\n" (14 chars).
	//   row 1: "abc\n"
	//   row 2: "\n"       (blank line)
	//   row 3: "def\n"
	//   row 4: "\n"       (blank line)
	//   row 5: "ghi\n"
	fr.Insert([]rune("abc\n\ndef\n\nghi\n"), 0)
	gdo(t, fr).Clear()

	// Delete "def" (positions 5–7); the surrounding newlines stay intact.
	s := fr.Delete(5, 8)

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// deleteLeadingNewlineAndDef removes "\ndef" from "abc\n\ndef\n\nghi\n".
// The blank line ('\n' at pos 4) and "def" (pos 5-7) are deleted; the '\n'
// that terminated "def" and the remaining "\nghi\n" stay, so "ghi" ripples
// up and one visual line disappears.
func deleteLeadingNewlineAndDef(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("abc\n\ndef\n\nghi\n"), 0)
	gdo(t, fr).Clear()

	// Delete "\ndef" (positions 4–7); keeps the '\n' at position 8 onward.
	s := fr.Delete(4, 8)

	if got, want := s, 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// deleteDefAndTrailingNewline removes "def\n" from "abc\n\ndef\n\nghi\n".
// "def" (pos 5-7) and its '\n' (pos 8) are deleted; the preceding blank line
// and "\nghi\n" remain, so "ghi" ripples up and one visual line disappears.
func deleteDefAndTrailingNewline(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("abc\n\ndef\n\nghi\n"), 0)
	gdo(t, fr).Clear()

	// Delete "def\n" (positions 5–8); the '\n' at position 9 and "ghi\n" stay.
	s := fr.Delete(5, 9)

	if got, want := s, 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// deleteEfFromBetweenBlanks removes "ef" from "abc\n\ndef\n\nghi\n".
// "def" shrinks to "d" on row 3; surrounding blank lines stay and no
// visual line disappears.
func deleteEfFromBetweenBlanks(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	// 6-row frame: rows are "abc\n", "\n", "def\n", "\n", "ghi\n".
	fr.Insert([]rune("abc\n\ndef\n\nghi\n"), 0)
	gdo(t, fr).Clear()

	// 'e' is at pos 6, 'f' at pos 7. Delete(6, 8).
	s := fr.Delete(6, 8)

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// deleteFFromBetweenBlanks removes "f" from "abc\n\ndef\n\nghi\n".
// "def" shrinks to "de" on row 3; surrounding blank lines stay and no
// visual line disappears.
func deleteFFromBetweenBlanks(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("abc\n\ndef\n\nghi\n"), 0)
	gdo(t, fr).Clear()

	// 'f' is at pos 7. Delete(7, 8).
	s := fr.Delete(7, 8)

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// deleteEfFromDefBlankGhi removes "ef" from "abc\ndef\n\nghi\n".
// "def" shrinks to "d" on row 2; trailing blank and "ghi" stay, no
// visual line disappears.
func deleteEfFromDefBlankGhi(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	// 5-row frame holds all of "abc\ndef\n\nghi\n" (13 chars).
	//   row 1: "abc\n"
	//   row 2: "def\n"
	//   row 3: "\n"       (blank line)
	//   row 4: "ghi\n"
	fr.Insert([]rune("abc\ndef\n\nghi\n"), 0)
	gdo(t, fr).Clear()

	// 'e' is at pos 5, 'f' at pos 6. Delete(5, 7).
	s := fr.Delete(5, 7)

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// deleteFFromDefBlankGhi removes "f" from "abc\ndef\n\nghi\n".
// "def" shrinks to "de" on row 2; trailing blank and "ghi" stay, no
// visual line disappears.
func deleteFFromDefBlankGhi(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("abc\ndef\n\nghi\n"), 0)
	gdo(t, fr).Clear()

	// 'f' is at pos 6. Delete(6, 7).
	s := fr.Delete(6, 7)

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// deleteEfFromDefGhi removes "ef" from "abc\ndef\nghi\n".
// "def" shrinks to "d" on row 2; "ghi" stays on row 3, no line disappears.
func deleteEfFromDefGhi(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	// 4-row frame holds all of "abc\ndef\nghi\n" (12 chars).
	//   row 1: "abc\n"
	//   row 2: "def\n"
	//   row 3: "ghi\n"
	fr.Insert([]rune("abc\ndef\nghi\n"), 0)
	gdo(t, fr).Clear()

	// 'e' is at pos 5, 'f' at pos 6. Delete(5, 7).
	s := fr.Delete(5, 7)

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// deleteFFromDefGhi removes "f" from "abc\ndef\nghi\n".
// "def" shrinks to "de" on row 2; "ghi" stays on row 3, no line disappears.
func deleteFFromDefGhi(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("abc\ndef\nghi\n"), 0)
	gdo(t, fr).Clear()

	// 'f' is at pos 6. Delete(6, 7).
	s := fr.Delete(6, 7)

	if got, want := s, 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// deleteFirstLineBeforeBlank removes "abc\n" from "abc\n\ndef\n\nghi\n".
// The blank line ripples up to row 1 and one visual line disappears.
func deleteFirstLineBeforeBlank(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("abc\n\ndef\n\nghi\n"), 0)
	gdo(t, fr).Clear()

	// Delete "abc\n" (positions 0–3); '\n' at pos 4 becomes the new row 1.
	s := fr.Delete(0, 4)

	if got, want := s, 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// deleteMiddleLine removes the middle of three newline-terminated lines.
// After the delete "ghi" should ripple up from line 3 to line 2 and line 3
// should be cleared.
func deleteMiddleLine(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	// Three lines: "abc\n" on line 1, "def\n" on line 2, "ghi" on line 3.
	fr.Insert([]rune("abc\ndef\nghi"), 0)
	gdo(t, fr).Clear()

	// Delete "def\n" (positions 4–8); one visual line should disappear.
	s := fr.Delete(4, 8)

	if got, want := s, 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// deleteEmptyMiddleLine removes the blank line between two non-empty lines.
// "abc\n\nghi" has an empty line 2; deleting the '\n' at position 4 should
// cause "ghi" to ripple up to line 2 and line 3 to be cleared.
func deleteEmptyMiddleLine(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	fr.Insert([]rune("abc\n\nghi"), 0)
	gdo(t, fr).Clear()

	// Delete the blank line's newline at position 4; one visual line disappears.
	s := fr.Delete(4, 5)

	if got, want := s, 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// deleteEliminatesSoftWrap deletes the character that was causing a soft wrap,
// so the first logical line now fits on one visual line. "1cd" ripples up from
// visual line 3 to visual line 2 and a blank line appears at the bottom.
func deleteEliminatesSoftWrap(t *testing.T, fr Frame, iv *invariants) {
	t.Helper()

	// In a narrow frame "0abX\n1cd" lays out as:
	//   line 1: "0ab"  (soft wrap – X doesn't fit in the 1-px remainder)
	//   line 2: "X\n"
	//   line 3: "1cd"
	fr.Insert([]rune("0abX\n1cd"), 0)
	gdo(t, fr).Clear()

	// Delete X (position 3). Soft wrap disappears: "0ab\n" fits on line 1,
	// "1cd" ripples up to line 2, line 3 becomes empty.
	s := fr.Delete(3, 4)

	if got, want := s, 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestDelete is a high-level Delete test
func TestDelete(t *testing.T) {
	iv := &invariants{
		topcorner: image.Pt(20, 10),
	}

	*validate = true

	tests := []struct {
		name        string
		fn          func(t *testing.T, fr Frame, iv *invariants)
		want        []string
		textarea    image.Rectangle
		knowntofail bool
	}{
		{
			// Delete a single character at line end as we'd see with a backspace
			// key press.
			name: "deleteSingleCharacterAtLineEnd",
			fn:   deleteSingleCharacterAtLineEnd,
			want: []string{
				"fill (46,10)-(59,20)",
			},
			textarea: image.Rect(20, 10, 60, 40),
		},
		{
			// Delete a single character in the middle of a terminal line.
			name: "deleteSingleCharacterInMiddle",
			fn:   deleteSingleCharacterInMiddle,
			want: []string{
				"blit (46,10)-(59,20), to (33,10)-(46,20)",
				"fill (46,10)-(46,20)",
				"fill (46,10)-(59,20)",
			},
			textarea: image.Rect(20, 10, 60, 40),
		},
		{
			// Delete multiple characters in the middle of a line where the deleted
			// run is bounded by space characters on both sides.
			name: "deleteMultipleCharacterInMiddle",
			fn:   deleteMultipleCharacterInMiddle,
			want: []string{
				"blit (137,20)-(280,30), to (85,20)-(228,30)",
				"fill (228,20)-(228,30)",
				"fill (228,20)-(400,30)",
				// Redundant blit?
				"blit (20,30)-(241,40), to (20,30)-(241,40)",
				"fill (241,30)-(241,40)",
			},
			textarea: image.Rect(20, 10, 400, 100),
		},
		{
			// Delete a newline to create a wrapped line. TODO(rjk): This op blits a
			// line to itself. This is visually fine but is wasted work. None of the
			// drawops generated here are necessary for a correct screen update.
			name: "deleteNewlineTocreateWrappedLine",
			fn:   deleteNewlineTocreateWrappedLine,
			want: []string{
				"blit (20,20)-(59,30), to (20,20)-(59,30)",
				"fill (59,20)-(59,30)",
			},
			textarea: image.Rect(20, 10, 60, 40),
		},

		{
			// Ripple up a single deleted character.
			name: "rippleUpDeletedChar",
			fn:   rippleUpDeletedChar,
			want: []string{
				"blit (46,10)-(59,20), to (33,10)-(46,20)",
				"fill (46,10)-(46,20)",
				"blit (20,20)-(33,30), to (46,10)-(59,20)",
				"fill (59,10)-(60,20)",
				"blit (33,20)-(59,30), to (20,20)-(46,30)",
				"fill (46,20)-(46,30)",
				"blit (20,30)-(33,40), to (46,20)-(59,30)",
				"fill (59,20)-(60,30)",
				"blit (33,30)-(59,40), to (20,30)-(46,40)",
				"fill (46,30)-(46,40)",
				"fill (46,30)-(59,40)",
			},
			textarea: image.Rect(20, 10, 60, 40),
		},
		{
			// Like rippleUpDeletedChar but the init string contains an explicit
			// newline after the soft-wrapped content.
			name: "rippleUpDeletedCharWithNewline",
			fn:   rippleUpDeletedCharWithNewline,
			want: []string{
				"blit (46,10)-(59,20), to (33,10)-(46,20)",
				"fill (46,10)-(46,20)",
				"blit (20,20)-(33,30), to (46,10)-(59,20)",
				"fill (59,10)-(60,20)",
				"blit (33,20)-(59,30), to (20,20)-(46,30)",
				"fill (46,20)-(46,30)",
				"fill (46,20)-(60,30)",
				"blit (20,30)-(59,40), to (20,30)-(59,40)",
				"fill (59,30)-(59,40)",
			},
			textarea: image.Rect(20, 10, 60, 40),
		},
		{
			// character followed by tab where character after tab shouldn't move, delete the tab
			// have to make this wide enough for tabs to work.
			name: "deleteTab",
			fn:   deleteTab,
			want: []string{
				"blit (124,10)-(137,20), to (33,10)-(46,20)",
				"fill (46,10)-(46,20)",
				"blit (20,20)-(111,30), to (46,10)-(137,20)",
				"fill (137,10)-(137,20)",
				"fill (137,10)-(140,20)",
				"fill (20,20)-(111,30)",
				"fill (137,10)-(140,20)",
				"fill (20,20)-(111,30)",
			},
			// Has to be wide enough to accommodate a tab. Tab is 8 * 13 charwidths = 104.
			textarea: image.Rect(20, 10, 140, 40),
		},
		{
			// Character followed by tab where character after tab shouldn't move,
			// delete the character, tab should stretch.
			name: "deleteCharBeforeTab",
			fn:   deleteCharBeforeTab,
			want: []string{
				"fill (33,10)-(124,20)",
			},
			// Has to be wide enough to accommodate a tab. Tab is 8 * 13 charwidths = 104.
			textarea: image.Rect(20, 10, 140, 40),
		},
		{
			// Ripple up a multiline deletion, text off the bottom.
			name: "rippleUpMultiLine",
			fn:   rippleUpMultiLine,
			want: []string{
				"blit (20,30)-(60,40), to (20,10)-(60,20)",
				"blit (20,40)-(60,40), to (20,20)-(60,20)",
				"fill (20,20)-(60,30)",
				"fill (20,30)-(60,40)",
				"fill (20,40)-(20,50)",
			},
			textarea: image.Rect(20, 10, 60, 40),
		},
		{
			// Delete the only visible character on line 2 of "ab\nc\nef"; the
			// trailing '\n' stays so no visual line disappears.
			name:     "deleteOnlyCharFromLine",
			fn:       deleteOnlyCharFromLine,
			textarea: image.Rect(20, 10, 60, 40),
			want: []string{
				"fill (20,20)-(60,30)",
			},
		},
		{
			// Delete "def" (pos 5-7) from "abc\n\ndef\n\nghi\n"; row 3 becomes
			// empty, surrounding blank lines absorb the change, no line disappears.
			name:     "deleteDefBetweenEmptyLines",
			fn:       deleteDefBetweenEmptyLines,
			textarea: image.Rect(20, 10, 60, 70),
			want: []string{
				"fill (20,30)-(60,40)",
			},
		},
		{
			// Delete "\ndef" (pos 4-7) from "abc\n\ndef\n\nghi\n"; blank line and
			// "def" removed, "ghi" ripples up — one visual line disappears.
			name:     "deleteLeadingNewlineAndDef",
			fn:       deleteLeadingNewlineAndDef,
			textarea: image.Rect(20, 10, 60, 70),
			want: []string{
				"fill (20,20)-(60,30)",
				"blit (20,40)-(60,50), to (20,30)-(60,40)",
				"blit (20,50)-(60,70), to (20,40)-(60,60)",
				"fill (20,50)-(60,60)",
				"fill (20,60)-(20,70)",
			},
		},
		{
			// Delete "def\n" (pos 5-8) from "abc\n\ndef\n\nghi\n"; "def" and its
			// newline removed, "ghi" ripples up — one visual line disappears.
			name:     "deleteDefAndTrailingNewline",
			fn:       deleteDefAndTrailingNewline,
			textarea: image.Rect(20, 10, 60, 70),
			want: []string{
				"blit (20,40)-(60,50), to (20,30)-(60,40)",
				"blit (20,50)-(60,70), to (20,40)-(60,60)",
				"fill (20,50)-(60,60)",
				"fill (20,60)-(20,70)",
			},
		},
		{
			// Delete "ef" (pos 6-7) from "abc\n\ndef\n\nghi\n"; "def"→"d", no line disappears.
			name:     "deleteEfFromBetweenBlanks",
			fn:       deleteEfFromBetweenBlanks,
			textarea: image.Rect(20, 10, 60, 70),
			want: []string{
				"fill (33,30)-(60,40)",
				"fill (20,40)-(60,50)",
			},
		},
		{
			// Delete "f" (pos 7) from "abc\n\ndef\n\nghi\n"; "def"→"de", no line disappears.
			name:     "deleteFFromBetweenBlanks",
			fn:       deleteFFromBetweenBlanks,
			textarea: image.Rect(20, 10, 60, 70),
			want: []string{
				"fill (46,30)-(60,40)",
				"fill (20,40)-(60,50)",
			},
		},
		{
			// Delete "ef" (pos 5-6) from "abc\ndef\n\nghi\n"; "def"→"d", no line disappears.
			name:     "deleteEfFromDefBlankGhi",
			fn:       deleteEfFromDefBlankGhi,
			textarea: image.Rect(20, 10, 60, 60),
			want: []string{
				"fill (33,20)-(60,30)",
				"fill (20,30)-(60,40)",
			},
		},
		{
			// Delete "f" (pos 6) from "abc\ndef\n\nghi\n"; "def"→"de", no line disappears.
			name:     "deleteFFromDefBlankGhi",
			fn:       deleteFFromDefBlankGhi,
			textarea: image.Rect(20, 10, 60, 60),
			want: []string{
				"fill (46,20)-(60,30)",
				"fill (20,30)-(60,40)",
			},
		},
		{
			// Delete "ef" (pos 5-6) from "abc\ndef\nghi\n"; "def"→"d", no line disappears.
			name:     "deleteEfFromDefGhi",
			fn:       deleteEfFromDefGhi,
			textarea: image.Rect(20, 10, 60, 50),
			want: []string{
				"fill (33,20)-(60,30)",
				"blit (20,30)-(59,40), to (20,30)-(59,40)",
				"fill (59,30)-(59,40)",
			},
		},
		{
			// Delete "f" (pos 6) from "abc\ndef\nghi\n"; "def"→"de", no line disappears.
			name:     "deleteFFromDefGhi",
			fn:       deleteFFromDefGhi,
			textarea: image.Rect(20, 10, 60, 50),
			want: []string{
				"fill (46,20)-(60,30)",
				"blit (20,30)-(59,40), to (20,30)-(59,40)",
				"fill (59,30)-(59,40)",
			},
		},
		{
			// Delete "abc\n" (pos 0-3) from "abc\n\ndef\n\nghi\n"; blank ripples to
			// row 1, one visual line disappears.
			name:     "deleteFirstLineBeforeBlank",
			fn:       deleteFirstLineBeforeBlank,
			textarea: image.Rect(20, 10, 60, 70),
			want: []string{
				"blit (20,20)-(60,30), to (20,10)-(60,20)",
				"blit (20,30)-(60,70), to (20,20)-(60,60)",
				"fill (20,50)-(60,60)",
				"fill (20,60)-(20,70)",
			},
		},
		{
			// Delete the middle of three lines: "ghi" ripples up to line 2,
			// line 3 is cleared.
			name: "deleteMiddleLine",
			fn:   deleteMiddleLine,
			want: []string{
				"blit (20,30)-(60,40), to (20,20)-(60,30)",
				"blit (20,40)-(60,40), to (20,30)-(60,30)",
				"fill (59,20)-(60,30)",
				"fill (20,30)-(59,40)",
			},
			textarea: image.Rect(20, 10, 60, 40),
		},
		{
			// Delete the blank middle line: "ghi" ripples up to line 2, line 3 clears.
			name: "deleteEmptyMiddleLine",
			fn:   deleteEmptyMiddleLine,
			want: []string{
				"blit (20,30)-(60,40), to (20,20)-(60,30)",
				"blit (20,40)-(60,40), to (20,30)-(60,30)",
				"fill (59,20)-(60,30)",
				"fill (20,30)-(59,40)",
			},
			textarea: image.Rect(20, 10, 60, 40),
		},
		{
			// Delete the character that causes a soft wrap; the wrap disappears
			// and text ripples up to fill the freed visual line.
			name: "deleteEliminatesSoftWrap",
			fn:   deleteEliminatesSoftWrap,
			want: []string{
				"fill (20,20)-(60,30)",
				"blit (20,30)-(60,40), to (20,20)-(60,30)",
				"blit (20,40)-(60,40), to (20,30)-(60,30)",
				"fill (59,20)-(60,30)",
				"fill (20,30)-(59,40)",
			},
			textarea: image.Rect(20, 10, 60, 40),
		},
		// Rippling tabs
		// Tabs in narrow columns (what are they even suppose to do?)
		// Need Tab insertion tests too (At beginning of document, into a narrow Window, forcing ripple)
		// character followed by tab where character after tab shouldn't move, delete the character
		// chunk, range, chunk (what does that mean?)
		// blank line rippling
		// delete whole line
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			iv.textarea = tc.textarea
			fr := setupFrame(t, iv)

			if tc.knowntofail {
				tc.fn(t, fr, iv)
				generateVisualizedOutput(t, fr)
				t.Log("known failing: bug not yet fixed")
				return
			}

			// TODO(rjk): validate here

			tc.fn(t, fr, iv)

			// TODO(rjk): validate here

			// Peek inside.
			got := gdo(t, fr).DrawOps()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("dump mismatch (-want +got):\n%s", diff)
			}

			visualizedoutputtest(t, fr)
		})
	}
}
