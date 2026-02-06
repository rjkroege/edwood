// Package wind provides the Window type and related components for edwood.
package wind

import (
	"github.com/rjkroege/edwood/markdown"
	"github.com/rjkroege/edwood/rich"
)

// PreviewState holds preview mode fields for rich text rendering.
// This encapsulates fields that were previously in the Window struct
// related to markdown preview functionality.
type PreviewState struct {
	previewMode bool   // true when showing rendered markdown preview
	clickPos    int    // rune position of last B1 null-click
	clickMsec   uint32 // timestamp of last B1 null-click

	sourceMap  *markdown.SourceMap  // maps rendered positions to source positions
	linkMap    *markdown.LinkMap    // maps rendered positions to link URLs
	imageCache *rich.ImageCache     // cache for loaded images in preview mode
}

// NewPreviewState creates a new PreviewState with default values.
func NewPreviewState() *PreviewState {
	return &PreviewState{}
}

// IsPreviewMode returns true if the window is in preview mode.
func (ps *PreviewState) IsPreviewMode() bool {
	return ps.previewMode
}

// SetPreviewMode enables or disables preview mode.
func (ps *PreviewState) SetPreviewMode(mode bool) {
	ps.previewMode = mode
}

// SourceMap returns the source map for mapping preview positions to source.
// Returns nil if no source map is set.
func (ps *PreviewState) SourceMap() *markdown.SourceMap {
	return ps.sourceMap
}

// SetSourceMap sets the source map.
func (ps *PreviewState) SetSourceMap(sm *markdown.SourceMap) {
	ps.sourceMap = sm
}

// LinkMap returns the link map for mapping preview positions to URLs.
// Returns nil if no link map is set.
func (ps *PreviewState) LinkMap() *markdown.LinkMap {
	return ps.linkMap
}

// SetLinkMap sets the link map.
func (ps *PreviewState) SetLinkMap(lm *markdown.LinkMap) {
	ps.linkMap = lm
}

// ImageCache returns the image cache for preview mode.
// Returns nil if no image cache is set.
func (ps *PreviewState) ImageCache() *rich.ImageCache {
	return ps.imageCache
}

// SetImageCache sets the image cache.
func (ps *PreviewState) SetImageCache(ic *rich.ImageCache) {
	ps.imageCache = ic
}

// ClearCache clears cached preview data (source map, link map, image cache).
// This does not affect the preview mode state itself.
func (ps *PreviewState) ClearCache() {
	ps.sourceMap = nil
	ps.linkMap = nil
	ps.imageCache = nil
}

// ClickState returns the last click position and timestamp for double-click detection.
func (ps *PreviewState) ClickState() (pos int, msec uint32) {
	return ps.clickPos, ps.clickMsec
}

// SetClickState updates the click state for double-click detection.
func (ps *PreviewState) SetClickState(pos int, msec uint32) {
	ps.clickPos = pos
	ps.clickMsec = msec
}
