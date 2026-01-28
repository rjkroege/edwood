// Package command provides command dispatch functionality for edwood.
// This file contains types and helpers for preview commands (Markdeep).
package command

import (
	"strings"
)

// =============================================================================
// Preview State Types
// =============================================================================

// PreviewAction represents whether to enter or exit preview mode.
type PreviewAction int

const (
	// PreviewEnter indicates preview mode should be enabled.
	PreviewEnter PreviewAction = iota
	// PreviewExit indicates preview mode should be disabled.
	PreviewExit
)

// PreviewState represents the state of a window for preview commands.
// This type captures the state needed to make decisions about whether
// the Markdeep command can be executed.
type PreviewState struct {
	fileName    string
	hasWindow   bool
	previewMode bool
}

// NewPreviewState creates a new PreviewState.
func NewPreviewState(fileName string, hasWindow, previewMode bool) *PreviewState {
	return &PreviewState{
		fileName:    fileName,
		hasWindow:   hasWindow,
		previewMode: previewMode,
	}
}

// FileName returns the file name.
func (ps *PreviewState) FileName() string {
	return ps.fileName
}

// HasWindow returns true if there is a window.
func (ps *PreviewState) HasWindow() bool {
	return ps.hasWindow
}

// IsPreviewMode returns true if currently in preview mode.
func (ps *PreviewState) IsPreviewMode() bool {
	return ps.previewMode
}

// IsMarkdown returns true if the file has a .md extension (case insensitive).
func (ps *PreviewState) IsMarkdown() bool {
	if ps.fileName == "" {
		return false
	}
	return strings.HasSuffix(strings.ToLower(ps.fileName), ".md")
}

// CanPreview determines if preview can be enabled.
// Returns (canPreview, reason) where reason is non-empty if canPreview is false.
func (ps *PreviewState) CanPreview() (bool, string) {
	if !ps.hasWindow {
		return false, "no window"
	}
	if ps.fileName == "" {
		return false, "no file name"
	}
	if !ps.IsMarkdown() {
		return false, "not markdown"
	}
	return true, ""
}

// ToggleAction returns what action should be taken when toggling preview.
func (ps *PreviewState) ToggleAction() PreviewAction {
	if ps.previewMode {
		return PreviewExit
	}
	return PreviewEnter
}

// =============================================================================
// Preview Operation Types
// =============================================================================

// PreviewOperation represents the parameters for a preview (Markdeep) operation.
// Markdeep toggles between showing raw markdown and rendered preview.
type PreviewOperation struct{}

// NewPreviewOperation creates a new PreviewOperation.
func NewPreviewOperation() *PreviewOperation {
	return &PreviewOperation{}
}

// Name returns the command name.
func (p *PreviewOperation) Name() string {
	return "Markdeep"
}

// RequiresWindow returns true because preview operates on a window.
func (p *PreviewOperation) RequiresWindow() bool {
	return true
}

// RequiresMarkdown returns true because preview only works on .md files.
func (p *PreviewOperation) RequiresMarkdown() bool {
	return true
}

// IsToggle returns true because Markdeep toggles preview mode on/off.
func (p *PreviewOperation) IsToggle() bool {
	return true
}

// =============================================================================
// Preview Resources Types
// =============================================================================

// PreviewResources tracks the resources needed for preview mode.
// When exiting preview mode, these resources need to be cleaned up.
type PreviewResources struct {
	sourceMap  interface{}
	linkMap    interface{}
	imageCache interface{}
}

// NewPreviewResources creates a new PreviewResources.
func NewPreviewResources() *PreviewResources {
	return &PreviewResources{}
}

// HasSourceMap returns true if a source map is set.
func (pr *PreviewResources) HasSourceMap() bool {
	return pr.sourceMap != nil
}

// SetSourceMap sets the source map.
func (pr *PreviewResources) SetSourceMap(sm interface{}) {
	pr.sourceMap = sm
}

// HasLinkMap returns true if a link map is set.
func (pr *PreviewResources) HasLinkMap() bool {
	return pr.linkMap != nil
}

// SetLinkMap sets the link map.
func (pr *PreviewResources) SetLinkMap(lm interface{}) {
	pr.linkMap = lm
}

// HasImageCache returns true if an image cache is set.
func (pr *PreviewResources) HasImageCache() bool {
	return pr.imageCache != nil
}

// SetImageCache sets the image cache.
func (pr *PreviewResources) SetImageCache(ic interface{}) {
	pr.imageCache = ic
}

// NeedsClear returns true if any resources are set and need clearing.
func (pr *PreviewResources) NeedsClear() bool {
	return pr.sourceMap != nil || pr.linkMap != nil || pr.imageCache != nil
}

// Clear clears all preview resources.
func (pr *PreviewResources) Clear() {
	pr.sourceMap = nil
	pr.linkMap = nil
	pr.imageCache = nil
}

// =============================================================================
// Font Configuration Helpers
// =============================================================================

// PreviewFontScales returns the font scale factors for heading levels.
// H1 = 2.0x, H2 = 1.5x, H3 = 1.25x
func PreviewFontScales() map[string]float64 {
	return map[string]float64{
		"H1": 2.0,
		"H2": 1.5,
		"H3": 1.25,
	}
}

// PreviewFontVariants returns the font variants needed for preview mode.
func PreviewFontVariants() []string {
	return []string{"bold", "italic", "bolditalic", "code"}
}

// =============================================================================
// Preview Command Registry
// =============================================================================

// PreviewCommandRegistry provides standard preview command entries for registration.
type PreviewCommandRegistry struct{}

// NewPreviewCommandRegistry creates a new PreviewCommandRegistry.
func NewPreviewCommandRegistry() *PreviewCommandRegistry {
	return &PreviewCommandRegistry{}
}

// RegisterPreviewCommands registers all preview commands with the dispatcher.
// The commands registered are: Markdeep
func (r *PreviewCommandRegistry) RegisterPreviewCommands(d *Dispatcher) {
	// Markdeep is not undoable (it's a view toggle, not text modification)
	// Flags are unused for Markdeep
	d.RegisterCommand(NewCommandEntry("Markdeep", false, true, true))
}
