// Package dump provides interfaces and types for window dump/load operations.
// These interfaces help decouple row.go from Window/Text internals, fixing
// layering violations noted at row.go:432 and row.go:487.
package dump

import (
	"strings"

	"github.com/rjkroege/edwood/dumpfile"
)

// ContentProvider abstracts access to buffer contents for serialization.
// This interface helps fix the layering violation at row.go:432 where
// Row.dump() directly accesses t.file.String().
type ContentProvider interface {
	// String returns the full buffer contents as a string.
	String() string
}

// ContentProviderFunc is a function adapter for ContentProvider.
type ContentProviderFunc func() string

func (f ContentProviderFunc) String() string {
	return f()
}

// DumpableWindow abstracts the window properties needed for dump operations.
// This helps decouple row.go from the internal structure of Window.
type DumpableWindow interface {
	// BodyContent returns an interface to get the body content.
	BodyContent() ContentProvider

	// TagContent returns an interface to get the tag content.
	TagContent() ContentProvider

	// IsDirty returns true if the buffer has unsaved changes.
	IsDirty() bool

	// IsDir returns true if this is a directory listing.
	IsDir() bool

	// Name returns the file/buffer name.
	Name() string

	// HasExternalControl returns true if window is controlled externally (nopen[QWevent] > 0).
	HasExternalControl() bool

	// DumpInfo returns directory and command string for Exec windows.
	DumpInfo() (dir, cmd string)

	// BodySelection returns the body selection range.
	BodySelection() (q0, q1 int)

	// TagSelection returns the tag selection range.
	TagSelection() (q0, q1 int)

	// Font returns the body font name.
	Font() string
}

// TagParser provides methods for parsing and manipulating window tags.
// This interface helps fix the layering violation at row.go:487 where
// tag parsing logic is embedded in loadhelper().
type TagParser interface {
	// ParseName extracts the filename from a tag buffer.
	ParseName(tagBuffer string) string

	// ParseAfterBar extracts the content after the "|" delimiter.
	ParseAfterBar(tagBuffer string) (string, bool)

	// BuildTag constructs a tag string from components.
	BuildTag(name string, afterBar string) string
}

// DefaultTagParser implements TagParser with standard tag parsing rules.
type DefaultTagParser struct{}

// ParseName extracts the filename (first space-separated word) from a tag buffer.
func (p *DefaultTagParser) ParseName(tagBuffer string) string {
	parts := strings.SplitN(tagBuffer, " ", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// ParseAfterBar extracts the content after the "|" delimiter.
func (p *DefaultTagParser) ParseAfterBar(tagBuffer string) (string, bool) {
	parts := strings.SplitN(tagBuffer, "|", 2)
	if len(parts) != 2 {
		return "", false
	}
	return parts[1], true
}

// BuildTag constructs a tag string from name and afterBar content.
func (p *DefaultTagParser) BuildTag(name string, afterBar string) string {
	// Standard tag format: "name |afterBar"
	return name + " |" + afterBar
}

// TagRestorer handles tag restoration during window loading.
// This is the counterpart to TagParser for the load side of row.go:487.
type TagRestorer interface {
	// RestoreTag takes a dumpfile.Text tag and extracts its components.
	// Returns the filename extracted from the tag, content after the bar, and any error.
	RestoreTag(tagText dumpfile.Text) (filename string, afterBar string, err error)

	// ValidateTag checks if a tag has the expected format.
	ValidateTag(tagBuffer string) error
}

// DefaultTagRestorer implements TagRestorer with standard tag restoration rules.
type DefaultTagRestorer struct {
	parser TagParser
}

// NewTagRestorer creates a new DefaultTagRestorer.
func NewTagRestorer(parser TagParser) *DefaultTagRestorer {
	return &DefaultTagRestorer{parser: parser}
}

// RestoreTag extracts tag components for window restoration.
func (r *DefaultTagRestorer) RestoreTag(tagText dumpfile.Text) (string, string, error) {
	// First split: "filename rest" -> ["filename", "rest"]
	parts := strings.SplitN(tagText.Buffer, " ", 2)
	if len(parts) != 2 {
		return "", "", &TagParseError{Tag: tagText.Buffer, Reason: "missing space after filename"}
	}
	filename := parts[0]

	// Find the part after "|"
	afterBar, ok := r.parser.ParseAfterBar(tagText.Buffer)
	if !ok {
		return "", "", &TagParseError{Tag: tagText.Buffer, Reason: "missing | delimiter"}
	}

	return filename, afterBar, nil
}

// ValidateTag checks tag format.
func (r *DefaultTagRestorer) ValidateTag(tagBuffer string) error {
	parts := strings.SplitN(tagBuffer, " ", 2)
	if len(parts) != 2 {
		return &TagParseError{Tag: tagBuffer, Reason: "missing space after filename"}
	}

	_, ok := r.parser.ParseAfterBar(tagBuffer)
	if !ok {
		return &TagParseError{Tag: tagBuffer, Reason: "missing | delimiter"}
	}

	return nil
}

// TagParseError represents an error parsing a tag.
type TagParseError struct {
	Tag    string
	Reason string
}

func (e *TagParseError) Error() string {
	return "bad window tag: " + e.Reason + ": " + e.Tag
}

// WindowDumper uses DumpableWindow to create dump file entries.
// This demonstrates how the interface could be used to fix the layering.
type WindowDumper struct {
	parser TagParser
}

// NewWindowDumper creates a WindowDumper with the given tag parser.
func NewWindowDumper(parser TagParser) *WindowDumper {
	return &WindowDumper{parser: parser}
}

// DumpWindow creates a dumpfile.Window from a DumpableWindow.
// This method helps fix the layering violation by moving window
// serialization logic out of Row.dump().
func (d *WindowDumper) DumpWindow(w DumpableWindow, colIdx int, position float64) *dumpfile.Window {
	dw := &dumpfile.Window{
		Column:   colIdx,
		Position: position,
		Font:     w.Font(),
		Tag: dumpfile.Text{
			Buffer: w.TagContent().String(),
		},
		Body: dumpfile.Text{
			Buffer: "", // Will be filled for unsaved windows
		},
	}

	// Set selection ranges
	dw.Body.Q0, dw.Body.Q1 = w.BodySelection()
	dw.Tag.Q0, dw.Tag.Q1 = w.TagSelection()

	// Determine window type and handle special cases
	switch {
	case w.HasExternalControl():
		dir, cmd := w.DumpInfo()
		if cmd != "" {
			dw.Type = dumpfile.Exec
			dw.ExecDir = dir
			dw.ExecCommand = cmd
		}
		// Skip windows with external control but no dump command

	case w.IsDirty() && !w.IsDir():
		// Unsaved window - include body content
		dw.Type = dumpfile.Unsaved
		dw.Body.Buffer = w.BodyContent().String()

	default:
		// Saved window
		dw.Type = dumpfile.Saved
	}

	return dw
}
