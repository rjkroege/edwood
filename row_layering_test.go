package main

import (
	"testing"

	"github.com/rjkroege/edwood/dumpfile"
	dumputil "github.com/rjkroege/edwood/internal/dump"
)

// These tests verify that the interfaces in internal/dump can be used to fix
// layering violations in row.go.
// The layering concerns are:
// 1. Line 432: Row.dump() directly accesses t.file.String() to serialize buffer contents
// 2. Line 487: Row.loadhelper() contains tag parsing logic that should be unified
//
// The production interfaces are defined in internal/dump/interfaces.go.
// These tests verify their integration with the main package.

// mockDumpableWindow is a mock implementation of dump.DumpableWindow for testing.
// It demonstrates how Window could implement the interface to fix layering.
type mockDumpableWindow struct {
	bodyContent string
	tagContent  string
	dirty       bool
	isDir       bool
	name        string
	extControl  bool
	dumpDir     string
	dumpCmd     string
	bodyQ0      int
	bodyQ1      int
	tagQ0       int
	tagQ1       int
	font        string
}

func (m *mockDumpableWindow) BodyContent() dumputil.ContentProvider {
	return dumputil.ContentProviderFunc(func() string { return m.bodyContent })
}

func (m *mockDumpableWindow) TagContent() dumputil.ContentProvider {
	return dumputil.ContentProviderFunc(func() string { return m.tagContent })
}

func (m *mockDumpableWindow) IsDirty() bool              { return m.dirty }
func (m *mockDumpableWindow) IsDir() bool                { return m.isDir }
func (m *mockDumpableWindow) Name() string               { return m.name }
func (m *mockDumpableWindow) HasExternalControl() bool   { return m.extControl }
func (m *mockDumpableWindow) DumpInfo() (string, string) { return m.dumpDir, m.dumpCmd }
func (m *mockDumpableWindow) BodySelection() (int, int)  { return m.bodyQ0, m.bodyQ1 }
func (m *mockDumpableWindow) TagSelection() (int, int)   { return m.tagQ0, m.tagQ1 }
func (m *mockDumpableWindow) Font() string               { return m.font }

// TestContentProviderInterface tests that dumputil.ContentProvider can abstract
// buffer access for dump operations.
func TestContentProviderInterface(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "empty buffer",
			content:  "",
			expected: "",
		},
		{
			name:     "simple text",
			content:  "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "multiline text",
			content:  "Line 1\nLine 2\nLine 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "unicode text",
			content:  "Hello, 世界! こんにちは",
			expected: "Hello, 世界! こんにちは",
		},
		{
			name:     "code buffer",
			content:  "func main() {\n\tfmt.Println(\"Hello\")\n}",
			expected: "func main() {\n\tfmt.Println(\"Hello\")\n}",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Test function adapter
			provider := dumputil.ContentProviderFunc(func() string {
				return tc.content
			})

			got := provider.String()
			if got != tc.expected {
				t.Errorf("ContentProvider.String() = %q, want %q", got, tc.expected)
			}
		})
	}
}

// TestDumpableWindowInterface tests that dumputil.DumpableWindow can abstract
// window properties needed for dump operations.
func TestDumpableWindowInterface(t *testing.T) {
	// Create a mock implementation
	mock := &mockDumpableWindow{
		bodyContent: "body text",
		tagContent:  "/path/file.go Del Snarf",
		dirty:       true,
		isDir:       false,
		name:        "/path/file.go",
		extControl:  false,
		dumpDir:     "",
		dumpCmd:     "",
		bodyQ0:      0,
		bodyQ1:      10,
		tagQ0:       0,
		tagQ1:       14,
		font:        "/lib/font/go/Go-Regular.ttf",
	}

	// Test that we can extract all required properties
	if got := mock.BodyContent().String(); got != "body text" {
		t.Errorf("BodyContent().String() = %q, want %q", got, "body text")
	}

	if got := mock.TagContent().String(); got != "/path/file.go Del Snarf" {
		t.Errorf("TagContent().String() = %q, want %q", got, "/path/file.go Del Snarf")
	}

	if !mock.IsDirty() {
		t.Error("IsDirty() = false, want true")
	}

	if mock.IsDir() {
		t.Error("IsDir() = true, want false")
	}

	if got := mock.Name(); got != "/path/file.go" {
		t.Errorf("Name() = %q, want %q", got, "/path/file.go")
	}

	if mock.HasExternalControl() {
		t.Error("HasExternalControl() = true, want false")
	}

	dir, cmd := mock.DumpInfo()
	if dir != "" || cmd != "" {
		t.Errorf("DumpInfo() = (%q, %q), want (\"\", \"\")", dir, cmd)
	}

	q0, q1 := mock.BodySelection()
	if q0 != 0 || q1 != 10 {
		t.Errorf("BodySelection() = (%d, %d), want (0, 10)", q0, q1)
	}

	tq0, tq1 := mock.TagSelection()
	if tq0 != 0 || tq1 != 14 {
		t.Errorf("TagSelection() = (%d, %d), want (0, 14)", tq0, tq1)
	}

	if got := mock.Font(); got != "/lib/font/go/Go-Regular.ttf" {
		t.Errorf("Font() = %q, want %q", got, "/lib/font/go/Go-Regular.ttf")
	}
}

// TestDumpableWindowExecType tests the Exec window type abstraction.
func TestDumpableWindowExecType(t *testing.T) {
	mock := &mockDumpableWindow{
		bodyContent: "+Errors output",
		tagContent:  "/home/user/project/+Errors Del Snarf",
		dirty:       false,
		isDir:       false,
		name:        "/home/user/project/+Errors",
		extControl:  true,
		dumpDir:     "/home/user/project",
		dumpCmd:     "go test ./...",
		bodyQ0:      0,
		bodyQ1:      0,
		tagQ0:       0,
		tagQ1:       0,
		font:        "",
	}

	if !mock.HasExternalControl() {
		t.Error("Exec window should have external control")
	}

	dir, cmd := mock.DumpInfo()
	if dir != "/home/user/project" {
		t.Errorf("DumpInfo() dir = %q, want %q", dir, "/home/user/project")
	}
	if cmd != "go test ./..." {
		t.Errorf("DumpInfo() cmd = %q, want %q", cmd, "go test ./...")
	}
}

// TestTagParserInterface tests that dumputil.TagParser can handle tag parsing operations.
func TestTagParserInterface(t *testing.T) {
	parser := &dumputil.DefaultTagParser{}

	tests := []struct {
		name      string
		tagBuffer string
		wantName  string
		wantAfter string
		hasBar    bool
	}{
		{
			name:      "simple file tag",
			tagBuffer: "/path/to/file.go Del Snarf | Look",
			wantName:  "/path/to/file.go",
			wantAfter: " Look",
			hasBar:    true,
		},
		{
			name:      "directory tag",
			tagBuffer: "/home/user/ Del Snarf Get | Look",
			wantName:  "/home/user/",
			wantAfter: " Look",
			hasBar:    true,
		},
		{
			name:      "no bar tag",
			tagBuffer: "/path/to/file.go Del Snarf",
			wantName:  "/path/to/file.go",
			wantAfter: "",
			hasBar:    false,
		},
		{
			name:      "tag with commands",
			tagBuffer: "+Errors Del Snarf | mk",
			wantName:  "+Errors",
			wantAfter: " mk",
			hasBar:    true,
		},
		{
			name:      "empty tag",
			tagBuffer: "",
			wantName:  "",
			wantAfter: "",
			hasBar:    false,
		},
		{
			name:      "windows path",
			tagBuffer: "C:\\Users\\test\\file.go Del Snarf | Look",
			wantName:  "C:\\Users\\test\\file.go",
			wantAfter: " Look",
			hasBar:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotName := parser.ParseName(tc.tagBuffer)
			if gotName != tc.wantName {
				t.Errorf("ParseName(%q) = %q, want %q", tc.tagBuffer, gotName, tc.wantName)
			}

			gotAfter, gotHasBar := parser.ParseAfterBar(tc.tagBuffer)
			if gotHasBar != tc.hasBar {
				t.Errorf("ParseAfterBar(%q) hasBar = %v, want %v", tc.tagBuffer, gotHasBar, tc.hasBar)
			}
			if gotHasBar && gotAfter != tc.wantAfter {
				t.Errorf("ParseAfterBar(%q) afterBar = %q, want %q", tc.tagBuffer, gotAfter, tc.wantAfter)
			}
		})
	}
}

// TestTagParserBuildTag tests tag construction using dumputil.DefaultTagParser.
func TestTagParserBuildTag(t *testing.T) {
	parser := &dumputil.DefaultTagParser{}

	tests := []struct {
		name     string
		filename string
		afterBar string
		want     string
	}{
		{
			name:     "simple file",
			filename: "/path/file.go",
			afterBar: " Look",
			want:     "/path/file.go | Look",
		},
		{
			name:     "with commands",
			filename: "/path/file.go",
			afterBar: " mk all",
			want:     "/path/file.go | mk all",
		},
		{
			name:     "empty after bar",
			filename: "/path/file.go",
			afterBar: "",
			want:     "/path/file.go |",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parser.BuildTag(tc.filename, tc.afterBar)
			if got != tc.want {
				t.Errorf("BuildTag(%q, %q) = %q, want %q", tc.filename, tc.afterBar, got, tc.want)
			}
		})
	}
}

// TestTagParserRoundTrip tests that parsing and rebuilding produces valid results.
func TestTagParserRoundTrip(t *testing.T) {
	parser := &dumputil.DefaultTagParser{}

	// Tags should be parseable after BuildTag
	originalName := "/path/to/file.go"
	originalAfter := " Look mk"

	tag := parser.BuildTag(originalName, originalAfter)

	gotName := parser.ParseName(tag)
	if gotName != originalName {
		t.Errorf("ParseName after BuildTag: got %q, want %q", gotName, originalName)
	}

	gotAfter, ok := parser.ParseAfterBar(tag)
	if !ok {
		t.Error("ParseAfterBar after BuildTag: expected bar to be present")
	}
	if gotAfter != originalAfter {
		t.Errorf("ParseAfterBar after BuildTag: got %q, want %q", gotAfter, originalAfter)
	}
}

// TestWindowDumperCreation tests dumputil.WindowDumper can create valid dumpfile entries.
func TestWindowDumperCreation(t *testing.T) {
	dumper := dumputil.NewWindowDumper(&dumputil.DefaultTagParser{})

	tests := []struct {
		name     string
		window   *mockDumpableWindow
		colIdx   int
		position float64
		wantType dumpfile.WindowType
		wantBody bool
	}{
		{
			name: "saved window",
			window: &mockDumpableWindow{
				bodyContent: "saved content",
				tagContent:  "/path/file.go Del",
				dirty:       false,
				isDir:       false,
				name:        "/path/file.go",
				extControl:  false,
			},
			colIdx:   0,
			position: 25.0,
			wantType: dumpfile.Saved,
			wantBody: false,
		},
		{
			name: "unsaved window",
			window: &mockDumpableWindow{
				bodyContent: "unsaved content",
				tagContent:  "/path/file.go Del",
				dirty:       true,
				isDir:       false,
				name:        "/path/file.go",
				extControl:  false,
			},
			colIdx:   1,
			position: 50.0,
			wantType: dumpfile.Unsaved,
			wantBody: true,
		},
		{
			name: "exec window",
			window: &mockDumpableWindow{
				bodyContent: "output",
				tagContent:  "+Errors Del",
				dirty:       false,
				isDir:       false,
				name:        "+Errors",
				extControl:  true,
				dumpDir:     "/project",
				dumpCmd:     "make",
			},
			colIdx:   0,
			position: 75.0,
			wantType: dumpfile.Exec,
			wantBody: false,
		},
		{
			name: "directory window",
			window: &mockDumpableWindow{
				bodyContent: "file1.go\nfile2.go\n",
				tagContent:  "/path/ Del",
				dirty:       true, // directories can be "dirty" but shouldn't save body
				isDir:       true,
				name:        "/path/",
				extControl:  false,
			},
			colIdx:   0,
			position: 0.0,
			wantType: dumpfile.Saved, // Directories are treated as saved
			wantBody: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dw := dumper.DumpWindow(tc.window, tc.colIdx, tc.position)

			if dw.Type != tc.wantType {
				t.Errorf("Type = %v, want %v", dw.Type, tc.wantType)
			}

			if dw.Column != tc.colIdx {
				t.Errorf("Column = %d, want %d", dw.Column, tc.colIdx)
			}

			if dw.Position != tc.position {
				t.Errorf("Position = %f, want %f", dw.Position, tc.position)
			}

			hasBody := dw.Body.Buffer != ""
			if hasBody != tc.wantBody {
				t.Errorf("has body = %v, want %v", hasBody, tc.wantBody)
			}

			// Verify tag content is preserved
			if dw.Tag.Buffer != tc.window.tagContent {
				t.Errorf("Tag.Buffer = %q, want %q", dw.Tag.Buffer, tc.window.tagContent)
			}
		})
	}
}

// TestTagRestorerInterface tests tag restoration functionality using dumputil.DefaultTagRestorer.
func TestTagRestorerInterface(t *testing.T) {
	restorer := dumputil.NewTagRestorer(&dumputil.DefaultTagParser{})

	tests := []struct {
		name         string
		tag          dumpfile.Text
		wantFilename string
		wantAfterBar string
		wantErr      bool
	}{
		{
			name: "valid tag",
			tag: dumpfile.Text{
				Buffer: "/path/file.go Del Snarf | Look",
				Q0:     0,
				Q1:     0,
			},
			wantFilename: "/path/file.go",
			wantAfterBar: " Look",
			wantErr:      false,
		},
		{
			name: "directory tag",
			tag: dumpfile.Text{
				Buffer: "/home/user/ Del Snarf Get | Look mk",
				Q0:     5,
				Q1:     10,
			},
			wantFilename: "/home/user/",
			wantAfterBar: " Look mk",
			wantErr:      false,
		},
		{
			name: "no space after filename",
			tag: dumpfile.Text{
				Buffer: "/pathfile.go",
			},
			wantErr: true,
		},
		{
			name: "no bar delimiter",
			tag: dumpfile.Text{
				Buffer: "/path/file.go Del Snarf",
			},
			wantErr: true,
		},
		{
			name: "empty tag",
			tag: dumpfile.Text{
				Buffer: "",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filename, afterBar, err := restorer.RestoreTag(tc.tag)

			if tc.wantErr {
				if err == nil {
					t.Error("RestoreTag() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("RestoreTag() unexpected error: %v", err)
				return
			}

			if filename != tc.wantFilename {
				t.Errorf("filename = %q, want %q", filename, tc.wantFilename)
			}

			if afterBar != tc.wantAfterBar {
				t.Errorf("afterBar = %q, want %q", afterBar, tc.wantAfterBar)
			}
		})
	}
}

// TestTagRestorerValidation tests tag validation using dumputil.DefaultTagRestorer.
func TestTagRestorerValidation(t *testing.T) {
	restorer := dumputil.NewTagRestorer(&dumputil.DefaultTagParser{})

	validTags := []string{
		"/path/file.go Del Snarf | Look",
		"/home/ Del | mk",
		"+Errors Del Snarf | go test",
	}

	for _, tag := range validTags {
		if err := restorer.ValidateTag(tag); err != nil {
			t.Errorf("ValidateTag(%q) = %v, want nil", tag, err)
		}
	}

	invalidTags := []string{
		"",
		"filename",
		"/path/file Del Snarf", // missing |
	}

	for _, tag := range invalidTags {
		if err := restorer.ValidateTag(tag); err == nil {
			t.Errorf("ValidateTag(%q) = nil, want error", tag)
		}
	}
}
