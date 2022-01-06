// Package dumpfile implements encoding and decoding of Edwood dump file.
//
// A dump file stores the state of Edwood so that it can be restored
// when Edwood is restarted.
package dumpfile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const version = 1

// WindowType defines the type of window.
type WindowType int

const (
	Saved   WindowType = iota // Saved is a File and directory stored on disk
	Unsaved                   // Unsaved contains buffer that's not stored on disk
	Zerox                     // Zerox is a copy of a Saved or Unsaved window
	Exec                      // Exec is a window controlled by an outside process
)

// Content stores the state of Edwood.
type Content struct {
	CurrentDir string    // Edwood's current working directory
	VarFont    string    // Variable width font
	FixedFont  string    // Fixed width font
	RowTag     Text      // Top-most tag (usually "Newcol ... Exit")
	Columns    []Column  // List of columns
	Windows    []*Window // List of windows across all columns
}

// Column stores the state of a column in Edwood.
type Column struct {
	Position float64 // Position within the row (in percentage)
	Tag      Text    // Tag above the column (usually "New ... Delcol")
}

// Window stores the state of a window in Edwood.
type Window struct {
	Type WindowType // Type of window

	Column   int     // Column index where the window will be placed
	Position float64 // Position within the column (in percentage)
	Font     string  `json:",omitempty"` // Font name or path

	// ctl line has these but there is no point storing them:
	//ID	int	// we regenerate window IDs when loading
	//TagLen	int	// redundant
	//BodyLen int	// redundant
	//IsDir bool	// redundant
	//Dirty bool	// WindowType == Unsaved

	Tag Text // Tag above this window (usually "/path/to/file Del ...")

	// Text buffer and selection of body.
	// Body.Buffer is empty if Type == Unsaved.
	Body Text

	// Used for Type == Exec
	ExecDir     string `json:",omitempty"` // Execute command in this directory
	ExecCommand string `json:",omitempty"` // Command to execute
}

// Text is a UTF-8 encoded text with a substring selected
// using rune-indexing (instead of byte-indexing).
type Text struct {
	Buffer string `json:",omitempty"` // UTF-8 encoded text
	Q0     int    // Selection starts at this rune position
	Q1     int    // Selection ends before this rune position
}

type versionedContent struct {
	Version int // Dump file format version
	*Content
}

// Load parses the dump file and returns its content.
func Load(file string) (*Content, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return decode(bufio.NewReader(f))
}

func decode(r io.Reader) (*Content, error) {
	var vc versionedContent

	dec := json.NewDecoder(r)
	err := dec.Decode(&vc)
	if err != nil {
		return nil, err
	}
	if vc.Version != version {
		return nil, fmt.Errorf("dump file format %v; expected %v", vc.Version, version)
	}
	return vc.Content, nil
}

// Save encodes the dump file content and writes it to file.
func (c *Content) Save(file string) error {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return c.encode(f)
}

func (c *Content) encode(w io.Writer) error {
	vc := versionedContent{
		Version: version,
		Content: c,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	return enc.Encode(&vc)
}
