// Package command provides command dispatch functionality for edwood.
package command

import (
	"testing"
)

// =============================================================================
// Tests for Preview Command Interfaces and Types
// =============================================================================
//
// These tests verify the interfaces and behaviors needed for preview commands
// (Markdeep) that will be extracted from exec.go.
//
// The actual command implementations depend on main package types (Window,
// RichText, markdown.SourceMap, etc.). These tests verify the command package
// can support preview operations through well-defined interfaces.

// =============================================================================
// Preview State Tests
// =============================================================================

// TestPreviewStateNew tests PreviewState creation.
func TestPreviewStateNew(t *testing.T) {
	ps := NewPreviewState("test.md", true, false)

	if ps.FileName() != "test.md" {
		t.Errorf("FileName() = %q, want %q", ps.FileName(), "test.md")
	}
	if !ps.HasWindow() {
		t.Error("HasWindow() = false, want true")
	}
	if ps.IsPreviewMode() {
		t.Error("IsPreviewMode() = true, want false")
	}
}

// TestPreviewStateCanPreview tests when preview can be enabled.
func TestPreviewStateCanPreview(t *testing.T) {
	tests := []struct {
		name       string
		fileName   string
		hasWindow  bool
		wantOK     bool
		wantReason string
	}{
		{
			name:       "markdown file can be previewed",
			fileName:   "readme.md",
			hasWindow:  true,
			wantOK:     true,
			wantReason: "",
		},
		{
			name:       "uppercase .MD extension",
			fileName:   "README.MD",
			hasWindow:  true,
			wantOK:     true,
			wantReason: "",
		},
		{
			name:       "mixed case extension",
			fileName:   "notes.Md",
			hasWindow:  true,
			wantOK:     true,
			wantReason: "",
		},
		{
			name:       "no window",
			fileName:   "test.md",
			hasWindow:  false,
			wantOK:     false,
			wantReason: "no window",
		},
		{
			name:       "empty filename",
			fileName:   "",
			hasWindow:  true,
			wantOK:     false,
			wantReason: "no file name",
		},
		{
			name:       "non-markdown file",
			fileName:   "main.go",
			hasWindow:  true,
			wantOK:     false,
			wantReason: "not markdown",
		},
		{
			name:       "text file",
			fileName:   "notes.txt",
			hasWindow:  true,
			wantOK:     false,
			wantReason: "not markdown",
		},
		{
			name:       "file with md in name but wrong extension",
			fileName:   "markdown_notes.txt",
			hasWindow:  true,
			wantOK:     false,
			wantReason: "not markdown",
		},
		{
			name:       "directory path with markdown file",
			fileName:   "/home/user/docs/readme.md",
			hasWindow:  true,
			wantOK:     true,
			wantReason: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ps := NewPreviewState(tc.fileName, tc.hasWindow, false)
			canPreview, reason := ps.CanPreview()
			if canPreview != tc.wantOK {
				t.Errorf("CanPreview() = %v, want %v", canPreview, tc.wantOK)
			}
			if reason != tc.wantReason {
				t.Errorf("CanPreview() reason = %q, want %q", reason, tc.wantReason)
			}
		})
	}
}

// TestPreviewStateIsMarkdown tests the IsMarkdown helper.
func TestPreviewStateIsMarkdown(t *testing.T) {
	tests := []struct {
		fileName string
		want     bool
	}{
		{"readme.md", true},
		{"README.MD", true},
		{"notes.Md", true},
		{"file.mD", true},
		{"main.go", false},
		{"notes.txt", false},
		{"", false},
		{"md", false},           // not a file extension
		{".md", true},           // hidden file with .md extension
		{"foo.markdown", false}, // only .md supported, not .markdown
	}

	for _, tc := range tests {
		t.Run(tc.fileName, func(t *testing.T) {
			ps := NewPreviewState(tc.fileName, true, false)
			if got := ps.IsMarkdown(); got != tc.want {
				t.Errorf("IsMarkdown() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestPreviewStateToggle tests preview mode toggling logic.
func TestPreviewStateToggle(t *testing.T) {
	// Start in normal mode
	ps := NewPreviewState("test.md", true, false)
	if ps.IsPreviewMode() {
		t.Error("should start in normal mode")
	}

	// Check that we want to enter preview mode
	action := ps.ToggleAction()
	if action != PreviewEnter {
		t.Errorf("ToggleAction() = %v, want PreviewEnter", action)
	}

	// Simulate entering preview mode
	ps = NewPreviewState("test.md", true, true)
	if !ps.IsPreviewMode() {
		t.Error("should be in preview mode")
	}

	// Check that we want to exit preview mode
	action = ps.ToggleAction()
	if action != PreviewExit {
		t.Errorf("ToggleAction() = %v, want PreviewExit", action)
	}
}

// =============================================================================
// Preview Operation Tests
// =============================================================================

// TestPreviewOperationNew tests PreviewOperation creation.
func TestPreviewOperationNew(t *testing.T) {
	op := NewPreviewOperation()

	if !op.RequiresWindow() {
		t.Error("preview should require a window")
	}
	if !op.RequiresMarkdown() {
		t.Error("preview should require markdown file")
	}
	if !op.IsToggle() {
		t.Error("Markdeep should be a toggle command")
	}
}

// TestPreviewOperationName tests the operation name.
func TestPreviewOperationName(t *testing.T) {
	op := NewPreviewOperation()

	// The command is called "Markdeep" in the UI
	if op.Name() != "Markdeep" {
		t.Errorf("Name() = %q, want %q", op.Name(), "Markdeep")
	}
}

// =============================================================================
// Preview Resources Tests
// =============================================================================

// TestPreviewResourcesNew tests PreviewResources creation and cleanup.
func TestPreviewResourcesNew(t *testing.T) {
	pr := NewPreviewResources()

	if pr.HasSourceMap() {
		t.Error("new PreviewResources should not have source map")
	}
	if pr.HasLinkMap() {
		t.Error("new PreviewResources should not have link map")
	}
	if pr.HasImageCache() {
		t.Error("new PreviewResources should not have image cache")
	}
}

// TestPreviewResourcesClear tests clearing resources.
func TestPreviewResourcesClear(t *testing.T) {
	pr := NewPreviewResources()

	// Set dummy values
	pr.SetSourceMap("dummy-source-map")
	pr.SetLinkMap("dummy-link-map")
	pr.SetImageCache("dummy-image-cache")

	if !pr.HasSourceMap() {
		t.Error("should have source map after setting")
	}

	// Clear all resources
	pr.Clear()

	if pr.HasSourceMap() {
		t.Error("source map should be cleared")
	}
	if pr.HasLinkMap() {
		t.Error("link map should be cleared")
	}
	if pr.HasImageCache() {
		t.Error("image cache should be cleared")
	}
}

// TestPreviewResourcesNeedsClear tests when resources need clearing.
func TestPreviewResourcesNeedsClear(t *testing.T) {
	tests := []struct {
		name      string
		setSource bool
		setLink   bool
		setImage  bool
		want      bool
	}{
		{"empty", false, false, false, false},
		{"source map only", true, false, false, true},
		{"link map only", false, true, false, true},
		{"image cache only", false, false, true, true},
		{"all set", true, true, true, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pr := NewPreviewResources()
			if tc.setSource {
				pr.SetSourceMap("dummy")
			}
			if tc.setLink {
				pr.SetLinkMap("dummy")
			}
			if tc.setImage {
				pr.SetImageCache("dummy")
			}
			if got := pr.NeedsClear(); got != tc.want {
				t.Errorf("NeedsClear() = %v, want %v", got, tc.want)
			}
		})
	}
}

// =============================================================================
// Preview Command Entry Tests
// =============================================================================

// TestPreviewCommandEntry tests that Markdeep command has correct properties.
func TestPreviewCommandEntry(t *testing.T) {
	d := NewDispatcher()
	reg := NewPreviewCommandRegistry()
	reg.RegisterPreviewCommands(d)

	// Verify Markdeep is registered
	cmd := d.LookupCommand("Markdeep")
	if cmd == nil {
		t.Fatal("Markdeep command not found")
	}

	// Markdeep is not undoable (it's a view toggle, not a text modification)
	if cmd.Mark() {
		t.Error("Markdeep should not be undoable (mark=false)")
	}

	// Verify the name
	if cmd.Name() != "Markdeep" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "Markdeep")
	}
}

// TestPreviewCommandDispatch tests preview command lookup.
func TestPreviewCommandDispatch(t *testing.T) {
	d := NewDispatcher()
	reg := NewPreviewCommandRegistry()
	reg.RegisterPreviewCommands(d)

	tests := []struct {
		input    string
		wantName string
		found    bool
	}{
		{"Markdeep", "Markdeep", true},
		{"markdeep", "", false}, // Case sensitive
		{"MARKDEEP", "", false}, // Case sensitive
		{"Preview", "", false},  // Not the command name
		{"Markdown", "", false}, // Not the command name
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			cmd := d.LookupCommand(tc.input)
			if tc.found {
				if cmd == nil {
					t.Errorf("expected to find command for %q", tc.input)
					return
				}
				if cmd.Name() != tc.wantName {
					t.Errorf("expected name %q, got %q", tc.wantName, cmd.Name())
				}
			} else {
				if cmd != nil {
					t.Errorf("expected nil for %q, got %v", tc.input, cmd)
				}
			}
		})
	}
}

// TestPreviewCommandRegistryIntegration tests the full registration flow.
func TestPreviewCommandRegistryIntegration(t *testing.T) {
	d := NewDispatcher()
	reg := NewPreviewCommandRegistry()
	reg.RegisterPreviewCommands(d)

	// Should have exactly 1 preview command
	cmds := d.Commands()
	if len(cmds) != 1 {
		t.Errorf("expected 1 command, got %d", len(cmds))
	}

	// Verify the command is Markdeep
	if len(cmds) > 0 && cmds[0].Name() != "Markdeep" {
		t.Errorf("expected Markdeep command, got %s", cmds[0].Name())
	}
}

// =============================================================================
// Combined Command Registry Tests
// =============================================================================

// TestAllCommandTypesWithPreview tests that all command types coexist.
func TestAllCommandTypesWithPreview(t *testing.T) {
	d := NewDispatcher()

	fileReg := NewFileCommandRegistry()
	fileReg.RegisterFileCommands(d)

	editReg := NewEditCommandRegistry()
	editReg.RegisterEditCommands(d)

	windowReg := NewWindowCommandRegistry()
	windowReg.RegisterWindowCommands(d)

	previewReg := NewPreviewCommandRegistry()
	previewReg.RegisterPreviewCommands(d)

	// Should have 16 total commands (6 file + 5 edit + 4 window + 1 preview)
	cmds := d.Commands()
	if len(cmds) != 16 {
		t.Errorf("expected 16 commands, got %d", len(cmds))
	}

	// Spot check each category
	if d.LookupCommand("Put") == nil {
		t.Error("Put (file command) should be registered")
	}
	if d.LookupCommand("Cut") == nil {
		t.Error("Cut (edit command) should be registered")
	}
	if d.LookupCommand("Zerox") == nil {
		t.Error("Zerox (window command) should be registered")
	}
	if d.LookupCommand("Markdeep") == nil {
		t.Error("Markdeep (preview command) should be registered")
	}
}

// =============================================================================
// Font Loading Validation Tests
// =============================================================================
//
// These tests verify the validation helpers for font loading during preview.
// The actual font loading depends on the display package, but we can validate
// the logic for determining what fonts should be loaded.

// TestFontScaleFactors tests the standard font scale factors.
func TestFontScaleFactors(t *testing.T) {
	scales := PreviewFontScales()

	// Verify H1, H2, H3 scale factors
	expected := map[string]float64{
		"H1": 2.0,
		"H2": 1.5,
		"H3": 1.25,
	}

	for name, want := range expected {
		if got, ok := scales[name]; !ok {
			t.Errorf("missing scale factor for %s", name)
		} else if got != want {
			t.Errorf("scale factor for %s = %v, want %v", name, got, want)
		}
	}
}

// TestFontVariants tests the font variants needed for preview.
func TestFontVariants(t *testing.T) {
	variants := PreviewFontVariants()

	// Should include bold, italic, bold-italic, and code
	expected := []string{"bold", "italic", "bolditalic", "code"}

	for _, want := range expected {
		found := false
		for _, v := range variants {
			if v == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing font variant: %s", want)
		}
	}
}
