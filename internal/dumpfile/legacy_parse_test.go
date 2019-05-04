package dumpfile

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sanity-io/litter"
)

type testvector struct {
	filename   string
	tc         *Content
	parseerror string
}

var (
	missingpth string
	tests      []testvector
)

func TestLegacyLoad(t *testing.T) {
	// Open each of the test files
	tests = append(tests, []testvector{
		{
			// short line
			filename:   filepath.Join("testdata", "legacy", "bad1.dump"),
			tc:         nil,
			parseerror: "EOF",
		},
		{
			// short line
			filename:   filepath.Join("testdata", "legacy", "bad2.dump"),
			tc:         nil,
			parseerror: "EOF",
		},
		{
			// short line
			filename:   filepath.Join("testdata", "legacy", "bad3.dump"),
			tc:         nil,
			parseerror: "EOF",
		},
		{
			// short line
			filename:   filepath.Join("testdata", "legacy", "bad4.dump"),
			tc:         nil,
			parseerror: "EOF",
		},
		{
			// too many columns
			filename:   filepath.Join("testdata", "legacy", "bad5.dump"),
			tc:         nil,
			parseerror: "bad number of column widths 11 in \"  0.0000000  59.9609375 0.0000000  59.9609375 0.0000000  59.9609375 0.0000000  59.9609375 0.0000000  59.9609375 59.9609375\"",
		},
		{
			// invalid column width
			filename:   filepath.Join("testdata", "legacy", "bad6.dump"),
			tc:         nil,
			parseerror: "parsing column width in \"  0.0000000  a\" had error strconv.ParseFloat: parsing \"a\": invalid syntax",
		},
		{
			// short line, w line
			filename:   filepath.Join("testdata", "legacy", "bad7.dump"),
			tc:         nil,
			parseerror: "EOF",
		},
		{
			// bad column identifier.
			filename:   filepath.Join("testdata", "legacy", "bad8.dump"),
			tc:         nil,
			parseerror: "parsing column id in \"c          a New Cut Paste Snarf Sort Zerox Delcol \" had error strconv.ParseInt: parsing \"a\": invalid syntax",
		},

		{
			filename: filepath.Join("testdata", "legacy", "nowin.dump"),
			tc: &Content{
				CurrentDir: "/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood",
				VarFont:    "/lib/font/bit/lucsans/euro.8.font",
				FixedFont:  "/lib/font/bit/lucm/unicode.9.font",
				RowTag: Text{
					Buffer: "Newcol Kill Putall Dump Exit ",
					Q0:     0, Q1: 0},
				Columns: []Column{
					{
						Position: 0,
						Tag: Text{
							Buffer: "New Cut Paste Snarf Sort Zerox Delcol ",
							Q0:     0, Q1: 0}},
					{
						Position: 59.9609375,
						Tag: Text{
							Buffer: "New Cut Paste Snarf Sort Zerox Delcol ",
							Q0:     0, Q1: 0}}},
				Windows: []*Window{}},
			parseerror: "",
		},
		{
			filename: filepath.Join("testdata", "legacy", "basic.dump"),
			tc: &Content{
				CurrentDir: "/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood",
				VarFont:    "/lib/font/bit/lucsans/euro.8.font",
				FixedFont:  "/lib/font/bit/lucm/unicode.9.font",
				RowTag: Text{
					Buffer: "Newcol Kill Putall Dump Exit ",
					Q0:     0,
					Q1:     0,
				},
				Columns: []Column{
					{
						Position: 0,
						Tag: Text{
							Buffer: "New Cut Paste Snarf Sort Zerox Delcol ",
							Q0:     0,
							Q1:     0,
						},
					},
					{
						Position: 59.9609375,
						Tag: Text{
							Buffer: "New Cut Paste Snarf Sort Zerox Delcol ",
							Q0:     0,
							Q1:     0,
						},
					},
				},
				Windows: []*Window{
					{
						Type:     Saved,
						Column:   1,
						Position: 2.2618232,
						Font:     "/lib/font/bit/lucsans/euro.8.font",
						Tag: Text{
							Buffer: "/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood/ Del Snarf Get | Look Edit ",
							Q0:     0,
							Q1:     0,
						},
						Body: Text{
							Buffer: "",
							Q0:     0,
							Q1:     0,
						},
						ExecDir:     "",
						ExecCommand: "",
					},
				},
			},
			parseerror: "",
		},
		{
			filename: filepath.Join("testdata", "legacy", "onecol.dump"),
			tc: &Content{
				CurrentDir: "/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood",
				VarFont:    "/lib/font/bit/lucsans/euro.8.font",
				FixedFont:  "/lib/font/bit/lucm/unicode.9.font",
				RowTag: Text{
					Buffer: "win Newcol Kill Putall Dump Exit win",
					Q0:     0,
					Q1:     0,
				},
				Columns: []Column{
					{
						Position: 0,
						Tag: Text{
							Buffer: "New Cut Paste Snarf Sort Zerox Delcol ",
							Q0:     0,
							Q1:     0,
						},
					},
				},
				Windows: []*Window{
					{
						Type:     Unsaved,
						Column:   0,
						Position: 2.2618232,
						Font:     "/lib/font/bit/lucsans/euro.8.font",
						Tag: Text{
							Buffer: "Del Snarf Undo | Look Edit ",
							Q0:     0,
							Q1:     0,
						},
						Body: Text{
							Buffer: "hello\n",
							Q0:     6,
							Q1:     6,
						},
						ExecDir:     "",
						ExecCommand: "",
					},
					{
						Type:     Exec,
						Column:   0,
						Position: 0,
						Font:     "",
						Tag: Text{
							Buffer: "",
							Q0:     0,
							Q1:     0,
						},
						Body: Text{
							Buffer: "",
							Q0:     0,
							Q1:     0,
						},
						ExecDir:     "/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood/",
						ExecCommand: "win",
					},
					{
						Type:     Saved,
						Column:   0,
						Position: 60.3838245,
						Font:     "/lib/font/bit/lucsans/euro.8.font",
						Tag: Text{
							Buffer: "/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood/README.md Del Snarf | Look Edit ",
							Q0:     0,
							Q1:     0,
						},
						Body: Text{
							Buffer: "",
							Q0:     0,
							Q1:     0,
						},
						ExecDir:     "",
						ExecCommand: "",
					},
				},
			},
			parseerror: "",
		},

		{
			filename: filepath.Join("testdata", "legacy", "zerox.dump"),
			tc: &Content{
				CurrentDir: "/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood",
				VarFont:    "/lib/font/bit/lucsans/euro.8.font", FixedFont: "/lib/font/bit/lucm/unicode.9.font",
				RowTag: Text{
					Buffer: "win Newcol Kill Putall Dump Exit win echo hi",
					Q0:     0, Q1: 0},
				Columns: []Column{
					{Position: 0, Tag: Text{
						Buffer: "New Cut Paste Snarf Sort Zerox Delcol ", Q0: 0, Q1: 0}},
					{Position: 59.9609375, Tag: Text{
						Buffer: "New Cut Paste Snarf Sort Zerox Delcol ", Q0: 0, Q1: 0}}},
				Windows: []*Window{
					{Type: 0, Column: 0, Position: 2.2618232,
						Font: "/lib/font/bit/lucsans/euro.8.font",
						Tag: Text{
							Buffer: "/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood/acme.go Del Snarf | Look Edit ",
							Q0:     0, Q1: 0}, Body: Text{Buffer: "", Q0: 0, Q1: 0}, ExecDir: "", ExecCommand: ""},
					{Type: 2, Column: 0, Position: 45.716244, Font: "/lib/font/bit/lucsans/euro.8.font", Tag: Text{Buffer: "/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood/acme.go Del Snarf | Look Edit ", Q0: 0, Q1: 0}, Body: Text{Buffer: "", Q0: 0, Q1: 0}, ExecDir: "", ExecCommand: ""},
					{Type: 2, Column: 0, Position: 87.114462, Font: "/lib/font/bit/lucsans/euro.8.font", Tag: Text{Buffer: "/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood/acme.go Del Snarf | Look Edit ", Q0: 0, Q1: 0}, Body: Text{Buffer: "", Q0: 0, Q1: 0}, ExecDir: "", ExecCommand: ""},
					{Type: 0, Column: 1, Position: 2.2618232, Font: "/lib/font/bit/lucsans/euro.8.font", Tag: Text{Buffer: "/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood/ Del Snarf Get | Look Edit ", Q0: 0, Q1: 0}, Body: Text{Buffer: "", Q0: 0, Q1: 0}, ExecDir: "", ExecCommand: ""},
					{Type: 3, Column: 1, Position: 0, Font: "", Tag: Text{Buffer: "", Q0: 0, Q1: 0}, Body: Text{Buffer: "", Q0: 0, Q1: 0}, ExecDir: "/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood/", ExecCommand: "win"},
					{Type: 1, Column: 1, Position: 68.6086361, Font: "/lib/font/bit/lucsans/euro.8.font",
						Tag: Text{
							Buffer: "/+Errors Del Snarf | Look Edit ",
							Q0:     0, Q1: 0},
						Body: Text{
							Buffer: "hi\n/Users/rjkroege/tools/gopkg/src/github.com/rjkroege/edwood/-Gubaidulina modified\n",
							Q0:     3, Q1: 84},
						ExecDir: "", ExecCommand: ""}}},
			parseerror: "",
		},

		// TODO(rjk): Insert some error handling test cases.

	}...)

	for _, v := range tests {
		c, err := LoadLegacy(v.filename, "/home/gopher")

		if v.parseerror == "" && err != nil {
			t.Errorf("%s: unexepcted error %#v\n", v.filename, err)
		}
		if err == nil && v.parseerror != "" {
			t.Errorf("%s: expected error %#v but got none.\n", v.filename, v.parseerror)
		}
		if err != nil && v.parseerror != err.Error() {
			t.Errorf("%s: error is %#v; expected %#v\n", v.filename, err.Error(), v.parseerror)
		}

		if !reflect.DeepEqual(v.tc, c) {
			t.Errorf("%s: content is %s; expected %s\n", v.filename, litter.Sdump(c), litter.Sdump(v.tc))
		}
	}
}
