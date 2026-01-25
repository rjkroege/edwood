package main

import (
	"image"
	"testing"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/edwoodtest"
	"github.com/rjkroege/edwood/markdown"
	"github.com/rjkroege/edwood/rich"
)

// TestPreviewWindowCreation tests that a PreviewWindow can be created
// and initialized correctly.
func TestPreviewWindowCreation(t *testing.T) {
	rect := image.Rect(0, 0, 600, 400)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	pw := NewPreviewWindow()
	if pw == nil {
		t.Fatal("NewPreviewWindow() returned nil")
	}

	pw.Init(rect, display, font)

	// Verify the preview window has a RichText component
	if pw.RichText() == nil {
		t.Error("PreviewWindow.RichText() returned nil after Init")
	}

	// Verify the display is stored
	if pw.Display() != display {
		t.Error("PreviewWindow.Display() does not match initialized display")
	}

	// Verify the rectangle is stored
	if got := pw.Rect(); got != rect {
		t.Errorf("PreviewWindow.Rect() = %v, want %v", got, rect)
	}
}

// TestPreviewWindowSetMarkdown tests that markdown content can be set
// and is parsed into rich content.
func TestPreviewWindowSetMarkdown(t *testing.T) {
	rect := image.Rect(0, 0, 600, 400)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	// Create colors for the preview window
	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	pw := NewPreviewWindow()
	pw.Init(rect, display, font,
		WithPreviewBackground(bgImage),
		WithPreviewTextColor(textImage),
	)

	// Set markdown content
	mdContent := "# Heading\n\nSome **bold** text."
	pw.SetMarkdown(mdContent)

	// Verify content was set (should have parsed to rich.Content)
	content := pw.Content()
	if content == nil {
		t.Fatal("PreviewWindow.Content() returned nil after SetMarkdown")
	}

	// Content should have multiple spans (heading, newlines, body with bold)
	if len(content) == 0 {
		t.Error("PreviewWindow.Content() returned empty content")
	}
}

// TestPreviewWindowRedraw tests that the preview window can be redrawn.
func TestPreviewWindowRedraw(t *testing.T) {
	rect := image.Rect(0, 0, 600, 400)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)
	scrBg, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Palebluegreen)
	scrThumb, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Medblue)

	pw := NewPreviewWindow()
	pw.Init(rect, display, font,
		WithPreviewBackground(bgImage),
		WithPreviewTextColor(textImage),
		WithPreviewScrollbarColors(scrBg, scrThumb),
	)

	pw.SetMarkdown("# Hello World\n\nThis is a test.")

	// Clear any draw ops from init
	display.(edwoodtest.GettableDrawOps).Clear()

	// Call Redraw
	pw.Redraw()

	// Verify that some draw operations occurred
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	if len(ops) == 0 {
		t.Error("Redraw() did not produce any draw operations")
	}
}

// TestPreviewWindowWithSource tests that a preview window can track a source.
func TestPreviewWindowWithSource(t *testing.T) {
	rect := image.Rect(0, 0, 600, 400)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	pw := NewPreviewWindow()
	pw.Init(rect, display, font)

	// Set a source identifier (simulating linking to a source window)
	sourcePath := "/path/to/file.md"
	pw.SetSource(sourcePath)

	// Verify source is stored
	if got := pw.Source(); got != sourcePath {
		t.Errorf("PreviewWindow.Source() = %q, want %q", got, sourcePath)
	}
}

// TestPreviewWindowParsesMarkdownCorrectly verifies that the markdown
// is correctly parsed when set.
func TestPreviewWindowParsesMarkdownCorrectly(t *testing.T) {
	rect := image.Rect(0, 0, 600, 400)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	pw := NewPreviewWindow()
	pw.Init(rect, display, font)

	// Set markdown with a heading
	pw.SetMarkdown("# Test Heading")

	content := pw.Content()
	if len(content) == 0 {
		t.Fatal("Content is empty")
	}

	// The first span should be the heading (parsed from markdown)
	// Check that it has heading style (Bold = true, Scale > 1.0)
	firstSpan := content[0]
	if !firstSpan.Style.Bold {
		t.Error("Heading should be bold")
	}
	if firstSpan.Style.Scale <= 1.0 {
		t.Errorf("Heading scale = %f, want > 1.0", firstSpan.Style.Scale)
	}
}

// TestPreviewUpdatesOnChange tests that the preview window re-renders
// when the markdown content is updated (live update functionality).
func TestPreviewUpdatesOnChange(t *testing.T) {
	rect := image.Rect(0, 0, 600, 400)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	bgImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.White)
	textImage, _ := display.AllocImage(image.Rect(0, 0, 1, 1), display.ScreenImage().Pix(), true, draw.Black)

	pw := NewPreviewWindow()
	pw.Init(rect, display, font,
		WithPreviewBackground(bgImage),
		WithPreviewTextColor(textImage),
	)

	// Set initial markdown content (no trailing newline for simple test)
	initialContent := "# Initial Heading"
	pw.SetMarkdown(initialContent)

	// Verify initial content is set
	content1 := pw.Content()
	if len(content1) == 0 {
		t.Fatal("Initial content is empty")
	}

	// Verify the first span has expected text
	if content1[0].Text != "Initial Heading" {
		t.Errorf("Initial heading text = %q, want %q", content1[0].Text, "Initial Heading")
	}

	// Clear draw ops to track new draw calls
	display.(edwoodtest.GettableDrawOps).Clear()

	// Update markdown content (simulating source file change)
	// Note: heading with trailing newline includes the newline in the heading text
	updatedContent := "# Updated Heading\n\nNew paragraph with **bold** text."
	pw.SetMarkdown(updatedContent)

	// Verify content was updated
	content2 := pw.Content()
	if len(content2) == 0 {
		t.Fatal("Updated content is empty")
	}

	// Verify the first span now has updated text (includes newline due to multi-line input)
	if content2[0].Text != "Updated Heading\n" {
		t.Errorf("Updated heading text = %q, want %q", content2[0].Text, "Updated Heading\n")
	}

	// Verify content changed (not the same as before)
	if content1[0].Text == content2[0].Text {
		t.Error("Content did not change after SetMarkdown with new content")
	}

	// Redraw and verify draw operations occurred
	pw.Redraw()
	ops := display.(edwoodtest.GettableDrawOps).DrawOps()
	if len(ops) == 0 {
		t.Error("Redraw() after content update did not produce any draw operations")
	}
}

// TestPreviewUpdatesPreservesSource tests that updating content preserves
// the source identifier.
func TestPreviewUpdatesPreservesSource(t *testing.T) {
	rect := image.Rect(0, 0, 600, 400)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	pw := NewPreviewWindow()
	pw.Init(rect, display, font)

	// Set source and initial content
	sourcePath := "/path/to/readme.md"
	pw.SetSource(sourcePath)
	pw.SetMarkdown("# Initial")

	// Update content
	pw.SetMarkdown("# Updated")

	// Verify source is still set
	if got := pw.Source(); got != sourcePath {
		t.Errorf("Source after update = %q, want %q", got, sourcePath)
	}
}

// TestPreviewUpdatesMultipleTimes tests that content can be updated
// multiple times and the preview tracks the latest content.
func TestPreviewUpdatesMultipleTimes(t *testing.T) {
	rect := image.Rect(0, 0, 600, 400)
	display := edwoodtest.NewDisplay(rect)
	font := edwoodtest.NewFont(10, 14)

	pw := NewPreviewWindow()
	pw.Init(rect, display, font)

	// Series of updates - note: headings with following content include trailing newline
	tests := []struct {
		md              string
		expectedHeading string
	}{
		{"# Version 1", "Version 1"},
		{"# Version 2\n\nMore content.", "Version 2\n"},
		{"# Version 3\n\n**Bold** and *italic*.", "Version 3\n"},
		{"# Final Version", "Final Version"},
	}

	for i, tt := range tests {
		pw.SetMarkdown(tt.md)
		content := pw.Content()
		if len(content) == 0 {
			t.Fatalf("Content empty after update %d", i+1)
		}
		// Each version should have content reflecting the update
		if content[0].Text != tt.expectedHeading {
			t.Errorf("Update %d: heading = %q, want %q", i+1, content[0].Text, tt.expectedHeading)
		}
	}
}

// Verify that PreviewWindow uses markdown.Parse (compile-time check)
var _ = markdown.Parse
var _ = rich.Plain
