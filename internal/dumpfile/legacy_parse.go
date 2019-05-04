package dumpfile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// readtrim returns a string read from the file or an error.
func readtrim(rd *bufio.Reader) (string, error) {
	l, err := rd.ReadString('\n')
	if err == io.EOF && l == "" {
		// We've run out of content.
		return "", nil
	} else if err != nil {
		return "", err
	}
	l = strings.TrimRight(l, "\r\n")
	return l, nil
}

var splittingregexp *regexp.Regexp

func init() {
	splittingregexp = regexp.MustCompile("[ \t]+")
}

// splitline splits the line based on a regexp and returns an array with not more than
// count elements.
func splitline(l string, count int) []string {
	splits := splittingregexp.Split(strings.TrimLeft(l, "\t "), count)
	// log.Printf("splitting %#v ➜ %#v", l, splits)
	return splits
}

// loadhelper breaks out common load file parsing functionality for selected row
// types.
func loadhelper(rd *bufio.Reader, subl []string, fontname string, numcol, ndumped int, wintype WindowType) (*Window, error) {
	// log.Printf("loadhelper start subl=%#v fontname=%s ndumped=%d dumpid=%d", subl, fontname, ndumped, dumpid)
	// defer log.Println("loadhelper done")
	// Column for this window.
	oi, err := strconv.ParseInt(subl[1], 10, 64)
	if err != nil || oi < 0 || oi > 10 {
		return nil, fmt.Errorf("cant't parse column id %s: %v", subl[1], err)
	}
	i := int(oi)

	// Bound i by number of columns.
	if i > numcol { // Didn't we already make sure that we have a column?
		i = numcol
	}

	// Window id is unused.

	// Get q0
	oq0, err := strconv.ParseInt(subl[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cant't parse q0 %s because %v", subl[3], err)
	}
	q0 := int(oq0)

	// Get q1
	oq1, err := strconv.ParseInt(subl[4], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("cant't parse q1 %s because %v", subl[4], err)
	}
	q1 := int(oq1)

	// Row size
	percent, err := strconv.ParseFloat(subl[5], 64)
	if err != nil {
		return nil, fmt.Errorf("cant't parse percent %s because %v", subl[5], err)
	}

	// Read the follow-on line for tag value.
	nextline, err := readtrim(rd)
	if err != nil {
		return nil, err
	}
	subl = splitline(nextline, 6)
	tag := subl[5]

	// Additional content if file was dirty.
	buffer := make([]byte, ndumped)
	// Read from the file into a string. Any amount missing
	// is considered a fatal error.
	if n, err := rd.Read(buffer); err != nil || n != ndumped {
		return nil, fmt.Errorf("can't load dumped file contents %v", err)
	}

	// TODO(rjk): set from new variable.
	return &Window{
		Type:     wintype,
		Column:   i,
		Position: percent,
		Font:     fontname,
		Tag: Text{
			Buffer: tag,
			Q0:     0,
			Q1:     0,
		},
		Body: Text{
			Buffer: string(buffer),
			Q0:     q0,
			Q1:     q1,
		},
		ExecDir:     "",
		ExecCommand: "",
	}, nil
}

// TODO(rjk): split this apart into smaller functions and files.
func LoadLegacy(file, home string) (*Content, error) {
	// log.Println("LoadLegacy start", file)

	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("loading old dumpfile file %s failed: %v", file, err)
	}
	defer f.Close()
	b := bufio.NewReader(f)

	dc := new(Content)

	// Current directory.
	l, err := readtrim(b)
	if err != nil {
		return nil, err
	}
	dc.CurrentDir = l

	// variable width font
	l, err = readtrim(b)
	if err != nil {
		return nil, err
	}
	dc.VarFont = l

	// fixed width font
	l, err = readtrim(b)
	if err != nil {
		return nil, err
	}
	dc.FixedFont = l

	// Column widths
	l, err = readtrim(b)
	if err != nil {
		return nil, err
	}
	subl := splitline(l, -1)

	if len(subl) > 10 {
		return nil, fmt.Errorf("bad number of column widths %d in %#v", len(subl), l)
	}
	dc.Columns = make([]Column, len(subl))

	for i, cwidth := range subl {
		percent, err := strconv.ParseFloat(cwidth, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing column width in %#v had error %v", l, err)
		}
		dc.Columns[i].Position = percent
	}

	dc.Windows = make([]*Window, 0, 10)

	// Read the window entries. There will be an entry for each Window. A Window may be
	// 1 or 2 lines except for Window records that correspond to each file. In which case,the
	// unsaved file contents will also be present.
	for {
		l, err = readtrim(b)
		if err != nil {
			return nil, err
		}

		// log.Printf("read line: %#v\n", l)
		// log.Printf("current dc: %#v\n", *dc)

		switch {
		case l == "":
			// We've reached the end.
			return dc, nil
		case l[0] == 'c':
			// Column header.
			subl := splitline(l, 3)
			bi, err := strconv.ParseInt(subl[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("parsing column id in %#v had error %v", l, err)
			}
			dc.Columns[bi].Tag = Text{Buffer: subl[2], Q0: 0, Q1: 0}
		case l[0] == 'w':
			subl := strings.TrimLeft(l[1:], " \t")
			dc.RowTag = Text{Buffer: subl, Q0: 0, Q1: 0}
		case l[0] == 'e': // command block
			if len(l) < 1+5*12+1 {
				return nil, fmt.Errorf("bad line %#v in dumpfile", l)
			}
			// We discard a line
			_, err = readtrim(b) // ctl line; ignored
			if err != nil {
				return nil, err
			}
			dirline, err := readtrim(b) // directory
			if err != nil {
				return nil, err
			}
			if dirline == "" {
				dirline = home
			}
			cmdline, err := readtrim(b) // command
			if err != nil {
				return nil, err
			}

			// TODO(rjk): We don't restore external commands very well.
			// This is something that I've long been unhappy about. Make it better.
			// TODO(rjk): Confirm that this will actually work.
			dc.Windows = append(dc.Windows, &Window{
				Type:     Exec,
				Column:   len(dc.Columns) - 1,
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
				ExecDir:     dirline,
				ExecCommand: cmdline,
			})
			// log.Println("cmdline", cmdline, "dirline", dirline)
		case l[0] == 'f':
			if len(l) < 1+5*12+1 {
				return nil, fmt.Errorf("bad line %#v in dumpfile", l)
			}
			spl := splitline(l, 7)
			win, err := loadhelper(b, spl, spl[6], len(dc.Columns), 0, Saved)
			if err != nil {
				return nil, err
			}
			dc.Windows = append(dc.Windows, win)
		case l[0] == 'F':
			if len(l) < 1+6*12+1 {
				return nil, fmt.Errorf("bad line %#v in dumpfile", l)
			}
			spl := splitline(l, 8)
			on, err := strconv.ParseInt(spl[6], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("bad count of unsaved text from line %#v in dumpfile", l)
			}
			ndumped := int(on)

			win, err := loadhelper(b, spl, spl[7], len(dc.Columns), ndumped, Unsaved)
			if err != nil {
				return nil, err
			}
			dc.Windows = append(dc.Windows, win)
		case l[0] == 'x':
			if len(l) < 1+5*12+1 {
				return nil, fmt.Errorf("bad line %#v in dumpfile", l)
			}
			spl := splitline(l, 7)
			win, err := loadhelper(b, spl, spl[6], len(dc.Columns), 0, Zerox)
			if err != nil {
				return nil, err
			}
			dc.Windows = append(dc.Windows, win)
		default:
			return nil, fmt.Errorf("default bad line %#v in dumpfile", l)
		}
	}
}
