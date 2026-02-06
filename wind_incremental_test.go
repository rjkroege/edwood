package main

// Tests for Phase 5.4: Integrate Incremental Updates
//
// These tests verify that UpdatePreview() produces the same rendered output
// as a full re-parse, regardless of whether the implementation uses the
// incremental path or the full re-parse path internally. Each test:
//
// 1. Sets up a Window in preview mode with initial markdown content.
// 2. Calls UpdatePreview() to establish the initial preview state.
// 3. Edits the body buffer (insert, delete, or replace).
// 4. Calls UpdatePreview() again.
// 5. Compares the preview content, source map, and link map against
//    a fresh ParseWithSourceMap() of the same body text.
//
// The tests are designed to pass with both the current full-re-parse
// UpdatePreview() and the future incremental path. If the incremental
// path produces different results from full re-parse, these tests will
// catch the discrepancy.
//
// Run with: go test -run TestIncrementalPreview ./...

import (
	"image"
	"testing"

	"github.com/rjkroege/edwood/edwoodtest"
	"github.com/rjkroege/edwood/file"
	"github.com/rjkroege/edwood/markdown"
	"github.com/rjkroege/edwood/rich"
)

// setupIncrementalTestWindow creates a Window in preview mode with the given
// markdown source. It calls UpdatePreview() once to establish the initial
// preview state. Returns the window ready for edit + re-preview testing.
func setupIncrementalTestWindow(t *testing.T, sourceMarkdown string) *Window {
	t.Helper()

	rect := image.Rect(0, 0, 800, 600)
	display := edwoodtest.NewDisplay(rect)
	global.configureGlobals(display)

	sourceRunes := []rune(sourceMarkdown)

	w := NewWindow().initHeadless(nil)
	w.display = display
	w.body = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("/test/incremental.md", sourceRunes),
	}
	w.body.all = image.Rect(0, 20, 800, 600)
	w.tag = Text{
		display: display,
		fr:      &MockFrame{},
		file:    file.MakeObservableEditableBuffer("", nil),
	}
	w.col = &Column{safe: true}
	w.r = rect

	font := edwoodtest.NewFont(10, 14)
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0xFFFFFFFF)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, 0x000000FF)

	rt := NewRichText()
	bodyRect := image.Rect(0, 20, 800, 600)
	rt.Init(display, font,
		WithRichTextBackground(bgImage),
		WithRichTextColor(textImage),
	)
	rt.Render(bodyRect)

	content, sourceMap, linkMap := markdown.ParseWithSourceMap(sourceMarkdown)
	rt.SetContent(content)

	w.richBody = rt
	w.SetPreviewSourceMap(sourceMap)
	w.SetPreviewLinkMap(linkMap)
	w.SetPreviewMode(true)

	// Run initial UpdatePreview to establish baseline state.
	w.UpdatePreview()

	return w
}

// testInsert inserts text into the window's body and records the edit for
// the incremental preview path.
func testInsert(w *Window, pos int, text []rune) {
	w.body.file.InsertAt(pos, text)
	w.recordEdit(markdown.EditRecord{Pos: pos, OldLen: 0, NewLen: len(text)})
}

// testDelete deletes a range from the window's body and records the edit for
// the incremental preview path.
func testDelete(w *Window, q0, q1 int) {
	w.body.file.DeleteAt(q0, q1)
	w.recordEdit(markdown.EditRecord{Pos: q0, OldLen: q1 - q0, NewLen: 0})
}

// comparePreviewWithFullParse verifies that the current preview state of w
// matches what ParseWithSourceMap would produce for the current body content.
// It compares:
// 1. Content spans (text + style) — exact match.
// 2. Source map behavior via ToSource/ToRendered at sampled positions.
// 3. Link map behavior via URLAt at sampled positions.
func comparePreviewWithFullParse(t *testing.T, w *Window, label string) {
	t.Helper()

	bodyContent := w.body.file.String()
	fullContent, fullSM, fullLM := markdown.ParseWithSourceMap(bodyContent)

	// Compare content: the rendered text from the preview should match.
	previewContent := w.RichBody().Content()

	if len(fullContent) != len(previewContent) {
		t.Errorf("%s: content span count mismatch: full-parse=%d, preview=%d",
			label, len(fullContent), len(previewContent))
		t.Logf("  full-parse text: %s", contentText(fullContent))
		t.Logf("  preview text:    %s", contentText(previewContent))
		return
	}

	for i := range fullContent {
		if fullContent[i].Text != previewContent[i].Text {
			t.Errorf("%s: span %d text mismatch: full=%q, preview=%q",
				label, i, fullContent[i].Text, previewContent[i].Text)
		}
		if fullContent[i].Style != previewContent[i].Style {
			t.Errorf("%s: span %d style mismatch: full=%+v, preview=%+v",
				label, i, fullContent[i].Style, previewContent[i].Style)
		}
	}

	// Compare rendered text length.
	fullLen := fullContent.Len()
	previewLen := previewContent.Len()
	if fullLen != previewLen {
		t.Errorf("%s: rendered length mismatch: full=%d, preview=%d",
			label, fullLen, previewLen)
	}

	// Compare source map behavior via ToSource at sampled rendered positions.
	// Test at start, end, and every 10th position.
	previewSM := w.PreviewSourceMap()
	if fullSM != nil && previewSM != nil && fullLen > 0 {
		positions := []int{0}
		for p := 10; p < fullLen; p += 10 {
			positions = append(positions, p)
		}
		if fullLen > 1 {
			positions = append(positions, fullLen-1)
		}

		for _, p := range positions {
			fullSrc0, fullSrc1 := fullSM.ToSource(p, p+1)
			prevSrc0, prevSrc1 := previewSM.ToSource(p, p+1)
			if fullSrc0 != prevSrc0 || fullSrc1 != prevSrc1 {
				t.Errorf("%s: ToSource(%d,%d) mismatch: full=(%d,%d), preview=(%d,%d)",
					label, p, p+1, fullSrc0, fullSrc1, prevSrc0, prevSrc1)
			}
		}
	}

	// Compare link map behavior via URLAt at sampled rendered positions.
	previewLM := w.PreviewLinkMap()
	if fullLM != nil && previewLM != nil && fullLen > 0 {
		for p := 0; p < fullLen; p++ {
			fullURL := fullLM.URLAt(p)
			prevURL := previewLM.URLAt(p)
			if fullURL != prevURL {
				t.Errorf("%s: URLAt(%d) mismatch: full=%q, preview=%q",
					label, p, fullURL, prevURL)
			}
		}
	}
}

// contentText extracts the concatenated text from a Content slice.
func contentText(c rich.Content) string {
	var s string
	for _, span := range c {
		s += span.Text
	}
	return s
}

// TestIncrementalPreviewEditInParagraph verifies that editing text within
// a paragraph produces the same preview output as a full re-parse.
func TestIncrementalPreviewEditInParagraph(t *testing.T) {
	initialSource := "# Title\n\nSome text here.\n\nAnother paragraph.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// Insert " extra" after "Some" in the first paragraph.
	insertPos := len([]rune("# Title\n\nSome"))
	insertText := []rune(" extra")
	testInsert(w,insertPos, insertText)

	w.UpdatePreview()

	comparePreviewWithFullParse(t, w, "edit-in-paragraph")
}

// TestIncrementalPreviewEditInCodeBlock verifies that editing text inside
// a fenced code block produces the same preview output as a full re-parse.
func TestIncrementalPreviewEditInCodeBlock(t *testing.T) {
	initialSource := "# Title\n\n```\nold code line\nmore code\n```\n\nAfter code.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// Replace "old" with "new" inside the code block.
	replacePos := len([]rune("# Title\n\n```\n"))
	testDelete(w,replacePos, replacePos+3)
	testInsert(w,replacePos, []rune("new"))

	w.UpdatePreview()

	comparePreviewWithFullParse(t, w, "edit-in-code-block")
}

// TestIncrementalPreviewDeleteHeading verifies that deleting a heading
// produces the same preview output as a full re-parse.
func TestIncrementalPreviewDeleteHeading(t *testing.T) {
	initialSource := "# Title\n\nFirst paragraph.\n\n## Section\n\nSecond paragraph.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// Delete "## Section\n".
	headingStart := len([]rune("# Title\n\nFirst paragraph.\n\n"))
	headingEnd := headingStart + len([]rune("## Section\n"))
	testDelete(w,headingStart, headingEnd)

	w.UpdatePreview()

	comparePreviewWithFullParse(t, w, "delete-heading")
}

// TestIncrementalPreviewAddListItem verifies that adding a list item
// produces the same preview output as a full re-parse.
func TestIncrementalPreviewAddListItem(t *testing.T) {
	initialSource := "# Title\n\n- item one\n- item two\n\nAfter list.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// Insert "- item three\n" after "- item two\n".
	insertPos := len([]rune("# Title\n\n- item one\n- item two\n"))
	testInsert(w,insertPos, []rune("- item three\n"))

	w.UpdatePreview()

	comparePreviewWithFullParse(t, w, "add-list-item")
}

// TestIncrementalPreviewEditWithBoldFormatting verifies that editing a
// paragraph containing bold text produces the same preview output.
func TestIncrementalPreviewEditWithBoldFormatting(t *testing.T) {
	initialSource := "# Title\n\nSome **bold** text here.\n\nAnother paragraph.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// Insert " very" before "bold" (inside the bold markers).
	insertPos := len([]rune("# Title\n\nSome **"))
	testInsert(w,insertPos, []rune("very "))

	w.UpdatePreview()

	comparePreviewWithFullParse(t, w, "edit-with-bold")
}

// TestIncrementalPreviewEditWithLink verifies that editing a paragraph
// containing a link produces the same preview output, including link map.
func TestIncrementalPreviewEditWithLink(t *testing.T) {
	initialSource := "# Title\n\nClick [here](http://example.com) please.\n\nEnd.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// Append " now" after "please".
	insertPos := len([]rune("# Title\n\nClick [here](http://example.com) please"))
	testInsert(w,insertPos, []rune(" now"))

	w.UpdatePreview()

	comparePreviewWithFullParse(t, w, "edit-with-link")
}

// TestIncrementalPreviewEditInTable verifies that editing a table cell
// produces the same preview output.
func TestIncrementalPreviewEditInTable(t *testing.T) {
	initialSource := "Text.\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\nMore text.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// Replace "1" with "10" in the table cell.
	replacePos := len([]rune("Text.\n\n| A | B |\n|---|---|\n| "))
	testDelete(w,replacePos, replacePos+1)
	testInsert(w,replacePos, []rune("10"))

	w.UpdatePreview()

	comparePreviewWithFullParse(t, w, "edit-in-table")
}

// TestIncrementalPreviewMergeParagraphs verifies that deleting the blank
// line between two paragraphs (merging them) produces the same output.
func TestIncrementalPreviewMergeParagraphs(t *testing.T) {
	initialSource := "First paragraph.\n\nSecond paragraph.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// Delete the blank line "\n" between paragraphs.
	deletePos := len([]rune("First paragraph.\n"))
	testDelete(w,deletePos, deletePos+1)

	w.UpdatePreview()

	comparePreviewWithFullParse(t, w, "merge-paragraphs")
}

// TestIncrementalPreviewSplitParagraph verifies that inserting a blank
// line to split a paragraph produces the same output.
func TestIncrementalPreviewSplitParagraph(t *testing.T) {
	initialSource := "First line.\nSecond line.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// Insert "\n" after first line to split into two paragraphs.
	insertPos := len([]rune("First line.\n"))
	testInsert(w,insertPos, []rune("\n"))

	w.UpdatePreview()

	comparePreviewWithFullParse(t, w, "split-paragraph")
}

// TestIncrementalPreviewMultipleEdits verifies that multiple sequential
// edits followed by a single UpdatePreview produce the same output.
func TestIncrementalPreviewMultipleEdits(t *testing.T) {
	initialSource := "# Title\n\nFirst paragraph.\n\nSecond paragraph.\n\nThird paragraph.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// Edit 1: Insert " updated" after "First" in the first paragraph.
	insertPos1 := len([]rune("# Title\n\nFirst"))
	testInsert(w,insertPos1, []rune(" updated"))

	// Edit 2: Insert " also" after "Third" in the third paragraph.
	// Account for the shift from the first insertion.
	insertPos2 := len([]rune("# Title\n\nFirst updated paragraph.\n\nSecond paragraph.\n\nThird"))
	testInsert(w,insertPos2, []rune(" also"))

	w.UpdatePreview()

	comparePreviewWithFullParse(t, w, "multiple-edits")
}

// TestIncrementalPreviewReplaceEntireContent verifies that replacing the
// entire body content produces the same output as a full re-parse.
func TestIncrementalPreviewReplaceEntireContent(t *testing.T) {
	initialSource := "# Old Title\n\nOld content here.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// Replace the entire content.
	newSource := "# New Title\n\n## Section\n\nCompletely different content.\n\n- list item\n"
	testDelete(w,0, w.body.file.Nr())
	testInsert(w,0, []rune(newSource))

	w.UpdatePreview()

	comparePreviewWithFullParse(t, w, "replace-entire-content")
}

// TestIncrementalPreviewNonASCII verifies correctness when the document
// contains multi-byte Unicode characters.
func TestIncrementalPreviewNonASCII(t *testing.T) {
	initialSource := "# Über\n\nText with émojis here.\n\nAnother paragraph.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// Insert " more" after "émojis".
	insertPos := len([]rune("# Über\n\nText with émojis"))
	testInsert(w,insertPos, []rune(" more"))

	w.UpdatePreview()

	comparePreviewWithFullParse(t, w, "non-ascii")
}

// TestIncrementalPreviewEditFirstBlock verifies editing the very first
// block (heading) produces the same output.
func TestIncrementalPreviewEditFirstBlock(t *testing.T) {
	initialSource := "# Title\n\nParagraph.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// Replace "Title" with "New Title".
	replacePos := len([]rune("# "))
	testDelete(w,replacePos, replacePos+5) // delete "Title"
	testInsert(w,replacePos, []rune("New Title"))

	w.UpdatePreview()

	comparePreviewWithFullParse(t, w, "edit-first-block")
}

// TestIncrementalPreviewEditLastBlock verifies editing the very last
// block produces the same output.
func TestIncrementalPreviewEditLastBlock(t *testing.T) {
	initialSource := "# Title\n\nOnly paragraph.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// Append " extra" before the trailing newline.
	insertPos := len([]rune("# Title\n\nOnly paragraph"))
	testInsert(w,insertPos, []rune(" extra"))

	w.UpdatePreview()

	comparePreviewWithFullParse(t, w, "edit-last-block")
}

// TestIncrementalPreviewAddFencedCodeBlock verifies that adding a fenced
// code block produces the same output (this is a case that may trigger
// full re-parse in the incremental path, which is acceptable).
func TestIncrementalPreviewAddFencedCodeBlock(t *testing.T) {
	initialSource := "# Title\n\nSome text.\n\nMore text.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// Insert a fenced code block between paragraphs.
	insertPos := len([]rune("# Title\n\nSome text.\n\n"))
	codeBlock := "```\nnew code\n```\n\n"
	testInsert(w,insertPos, []rune(codeBlock))

	w.UpdatePreview()

	comparePreviewWithFullParse(t, w, "add-fenced-code-block")
}

// TestIncrementalPreviewSequentialUpdates verifies that multiple
// UpdatePreview calls in sequence (simulating repeated edits with
// debounce firing between each) all produce correct output.
func TestIncrementalPreviewSequentialUpdates(t *testing.T) {
	initialSource := "# Title\n\nParagraph one.\n\nParagraph two.\n"
	w := setupIncrementalTestWindow(t, initialSource)

	// First edit + update.
	insertPos := len([]rune("# Title\n\nParagraph one"))
	testInsert(w,insertPos, []rune(" edited"))
	w.UpdatePreview()
	comparePreviewWithFullParse(t, w, "sequential-update-1")

	// Second edit + update.
	insertPos2 := len([]rune("# Title\n\nParagraph one edited.\n\nParagraph two"))
	testInsert(w,insertPos2, []rune(" also"))
	w.UpdatePreview()
	comparePreviewWithFullParse(t, w, "sequential-update-2")

	// Third edit: delete heading + update.
	testDelete(w,0, len([]rune("# Title\n\n")))
	w.UpdatePreview()
	comparePreviewWithFullParse(t, w, "sequential-update-3")
}
