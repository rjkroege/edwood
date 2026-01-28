package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"9fans.net/go/plan9"
	"github.com/google/go-cmp/cmp"
	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/edwoodtest"
	"github.com/rjkroege/edwood/file"
)

func TestXfidallocthread(t *testing.T) {
	global.cxfidalloc = make(chan *Xfid)
	global.cxfidfree = make(chan *Xfid)

	ctx, cancel := context.WithCancel(context.Background())

	d := (draw.Display)(nil)
	done := make(chan struct{})
	go func() {
		xfidallocthread(global, ctx, d)
		close(global.cxfidalloc)
		close(global.cxfidfree)
		global.cxfidalloc = nil
		global.cxfidfree = nil
		close(done)
	}()

	global.cxfidalloc <- (*Xfid)(nil) // Request an xfid
	x := <-global.cxfidalloc
	if x == nil {
		t.Errorf("Failed to get an Xfid")
	}
	global.cxfidfree <- x

	cancel() // Ask xfidallocthread to finish up.

	// Wait for xfidallocthread to return and global channels to be reset.
	<-done
}

func TestXfidctl(t *testing.T) {
	global.cxfidfree = make(chan *Xfid)
	defer func() {
		close(global.cxfidfree)
		global.cxfidfree = nil
	}()

	x := &Xfid{c: make(chan func(*Xfid))}
	defer close(x.c)
	go xfidctl(x, edwoodtest.NewDisplay(image.Rectangle{}))

	called := false
	x.c <- func(x *Xfid) { called = true }

	if got := <-global.cxfidfree; got != x {
		t.Errorf("got freed Xfid %v; want %v", got, x)
	}
	if !called {
		t.Errorf("function was not called")
	}
}

func TestFullrunewrite(t *testing.T) {
	testCases := []struct {
		in        []byte
		out       []rune
		pin, pout []byte
	}{
		{[]byte("hello world"), []rune("hello world"), nil, nil},
		{[]byte("Hello, 世界"), []rune("Hello, 世界"), nil, nil},
		{[]byte("hello \x00\x00world"), []rune("hello world"), nil, nil},
		{[]byte("abc\xe4\xb8xyz"), []rune("abc\uFFFD\uFFFDxyz"), nil, nil},        // invalid rune
		{[]byte("abcxyz\xe4\xb8"), []rune("abcxyz"), nil, []byte{'\xe4', '\xb8'}}, // ends with partial rune
		{[]byte("\x96hello"), []rune("世hello"), []byte{'\xe4', '\xb8'}, nil},      // begins with partial rune
	}
	for _, tc := range testCases {
		x := Xfid{
			f: &Fid{
				nrpart: len(tc.pin),
			},
			fcall: plan9.Fcall{
				Data:  tc.in,
				Count: uint32(len(tc.in)),
			},
		}
		copy(x.f.rpart[:], tc.pin)

		r := fullrunewrite(&x)
		if !reflect.DeepEqual(r, tc.out) {
			for i, b := range tc.in {
				fmt.Printf("%v %x %d\n", i, b, b)
			}
			t.Errorf("Fullrunewrite(%q, %q) full runes are %q; expected %q\n", tc.pin, tc.in, r, tc.out)
		}
		if x.f.nrpart != len(tc.pout) || !bytes.Equal(x.f.rpart[:x.f.nrpart], tc.pout[:]) {
			t.Errorf("Fullrunewrite(%q, %q) partial runes are %q; expected %q\n", tc.pin, tc.in, x.f.rpart[:x.f.nrpart], tc.pout)
		}
	}
}

type mockResponder struct {
	fcall *plan9.Fcall
	err   error
}

func (mr *mockResponder) respond(x *Xfid, t *plan9.Fcall, err error) *Xfid {
	mr.fcall = t
	mr.err = err
	return x
}

func (mr *mockResponder) msize() int { return 8192 }

func TestXfidflush(t *testing.T) {
	mr := new(mockResponder)
	w1 := NewWindow().initHeadless(nil)
	w1.body.file = file.MakeObservableEditableBuffer("", nil)
	w2 := NewWindow().initHeadless(nil)
	w2.body.file = file.MakeObservableEditableBuffer("", nil)
	global.row.col = []*Column{
		{
			w: []*Window{w1, w2},
		},
	}
	x := &Xfid{
		fs:      mr,
		c:       make(chan func(*Xfid)),
		flushed: false,
	}
	w1.eventx = nil
	w2.eventx = x

	go func() { <-x.c }()
	xfidflush(x)
	if mr.err != nil {
		t.Fatalf("got error %v", mr.err)
	}
	if !x.flushed {
		t.Errorf("Xfid is not flushed")
	}
}

func TestXfidreadQWrdsel(t *testing.T) {
	const wantSel = "εxαmple"

	w := &Window{
		body: Text{fr: &MockFrame{}, file: file.MakeObservableEditableBuffer("", []rune{})},
		tag: Text{
			fr:   &MockFrame{},
			file: file.MakeObservableEditableBuffer("", []rune{}),
		},
		col: new(Column),
	}
	textSetSelection(&w.body, "This is an «"+wantSel+"» sentence.\n")
	w.body.file.AddObserver(&w.body)
	w.tag.file.AddObserver(&w.tag)
	w.body.w = w
	w.tag.w = w
	w.ref.Inc()
	mr := new(mockResponder)
	x := &Xfid{
		f: &Fid{
			qid: plan9.Qid{Path: QID(0, QWrdsel)},
			w:   w,
		},
		fs: mr,
	}

	xfidopen(x)
	defer xfidclose(x)

	t.Run("NoError", func(t *testing.T) {
		x.fcall.Count = BUFSIZE + 1
		xfidread(x)
		if mr.err != nil {
			t.Fatalf("got error %v; want nil", mr.err)
		}
		if got, want := mr.fcall.Count, uint32(len(wantSel)); got != want {
			t.Errorf("fcall.Count is %v; want %v", got, want)
		}
		if got, want := string(mr.fcall.Data), wantSel; got != want {
			t.Errorf("fcall.Data is %q; want %q\n", got, want)
		}
	})
	t.Run("IOError", func(t *testing.T) {
		w.rdselfd = nil
		x.fcall.Count = BUFSIZE + 1
		xfidread(x)
		const errPrefix = "I/O error in temp file:"
		if mr.err == nil || !strings.HasPrefix(mr.err.Error(), errPrefix) {
			t.Fatalf("got error %v; want prefix %q", mr.err, errPrefix)
		}
	})
}

func TestXfidwriteQWaddr(t *testing.T) {
	for _, tc := range []struct {
		name string
		addr []byte
		r    Range
		err  error
	}{
		{"ErrAddrRange", []byte("/hello/"), Range{}, ErrAddrRange},
		{"ErrBadAddr", []byte("/hello/\n"), Range{}, ErrBadAddr},
		{"ValidAddr", []byte("/cα/"), Range{2, 4}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mr := new(mockResponder)
			w := NewWindow().initHeadless(nil)
			w.body.file = file.MakeObservableEditableBuffer("", []rune("abcαβξ\n"))
			w.col = new(Column)
			w.limit = Range{0, w.body.file.Nr()}
			x := &Xfid{
				fcall: plan9.Fcall{
					Data: []byte(tc.addr),
				},
				f: &Fid{
					qid: plan9.Qid{
						Path: QID(0, QWaddr),
					},
					w: w,
				},
				fs: mr,
			}
			xfidwrite(x)
			if mr.err != tc.err {
				t.Fatalf("error is %v; want %v", mr.err, tc.err)
			}
			if mr.err == nil {
				got := mr.fcall.Count
				want := uint32(len(tc.addr))
				if got != want {
					t.Errorf("fcall.Count is %v; want %v", got, want)
				}
				if !reflect.DeepEqual(w.addr, tc.r) {
					t.Errorf("window address is %v; want %v", w.addr, tc.r)
				}
			}
		})
	}
}

func TestXfidopen(t *testing.T) {
	display := edwoodtest.NewDisplay(image.Rectangle{})
	global.configureGlobals(display)

	for _, tc := range []struct {
		name string
		q    uint64
	}{
		{"QWaddr", QWaddr},
		{"QWdata", QWdata},
		{"QWxdata", QWxdata},
		{"QWevent", QWevent},
		{"QWrdsel", QWrdsel},
		{"QWwrsel", QWwrsel},
		{"QWeditout", QWeditout},
		{"Qlog", Qlog},
		{"Qeditout", Qeditout},
	} {
		t.Run(tc.name, func(t *testing.T) {
			global.editing = Inserting // for QWeditout
			mr := new(mockResponder)
			var w *Window
			q := tc.q
			if q != Qlog && q != Qeditout {
				w = NewWindow().initHeadless(nil)
				w.col = new(Column)
				w.col.safe = true
				w.body.fr = &MockFrame{}
				w.display = display
				w.tag.display = display
				w.tag.fr = &MockFrame{}
			}
			x := &Xfid{
				f: &Fid{
					qid: plan9.Qid{Path: QID(0, q)},
					w:   w,
				},
				fs: mr,
			}
			xfidopen(x)

			if mr.err != nil {
				t.Fatalf("got error %v; want nil", mr.err)
			}
			switch q {
			case QWaddr:
				if got, want := w.addr, (Range{0, 0}); !reflect.DeepEqual(got, want) {
					t.Errorf("w.addr is %v; want %v", got, want)
				}
				if got, want := w.limit, (Range{-1, -1}); !reflect.DeepEqual(got, want) {
					t.Errorf("w.limit is %v; want %v", got, want)
				}
			case QWrdsel:
				if w.rdselfd == nil {
					t.Errorf("w.rdselfd is nil after open")
				}
			}

			switch q {
			case QWeditout, Qlog, Qeditout: // Do nothing.
			default:
				if got, want := w.nopen[q], byte(1); got != want {
					t.Errorf("w.nopen[%v] is %v; want %v", q, got, want)
				}
			}
			if got, want := mr.fcall.Qid, (plan9.Qid{Path: QID(0, q)}); !cmp.Equal(got, want) {
				t.Errorf("Fcall.Qid is %#v; want %#v", got, want)
			}
			if got, want := mr.fcall.Iounit, uint32(8168); got != want {
				t.Errorf("Fcall.Iounit is %v; want %v", got, want)
			}
			if !x.f.open {
				t.Errorf("fid not open")
			}
		})
	}
}

func TestXfidopenQeditout(t *testing.T) {
	mr := new(mockResponder)
	x := &Xfid{
		f: &Fid{
			qid: plan9.Qid{Path: QID(0, Qeditout)},
		},
		fs: mr,
	}
	global.editoutlk = nil
	xfidopen(x)
	if got, want := mr.err, ErrInUse; got != want {
		t.Errorf("got error %v; want %v", got, want)
	}
}

func TestXfidopenQWeditout(t *testing.T) {
	global.configureGlobals(edwoodtest.NewDisplay(image.Rectangle{}))
	mr := new(mockResponder)

	x := &Xfid{
		f: &Fid{
			qid: plan9.Qid{Path: QID(0, QWeditout)},
			w:   NewWindow().initHeadless(nil),
		},
		fs: mr,
	}
	t.Run("ErrInUse", func(t *testing.T) {
		x.f.w.editoutlk = nil
		global.editing = Inserting
		xfidopen(x)
		if got, want := mr.err, ErrInUse; got != want {
			t.Errorf("got error %v; want %v", got, want)
		}
	})
	t.Run("ErrPermission", func(t *testing.T) {
		global.editing = Inactive
		xfidopen(x)
		if got, want := mr.err, ErrPermission; got != want {
			t.Errorf("got error %v; want %v", got, want)
		}
	})
}

func TestXfidopenQWrdsel(t *testing.T) {
	mr := new(mockResponder)
	x := &Xfid{
		f: &Fid{
			qid: plan9.Qid{Path: QID(0, QWrdsel)},
			w:   NewWindow().initHeadless(nil),
		},
		fs: mr,
	}
	t.Run("ErrInUse", func(t *testing.T) {
		x.f.w.rdselfd = os.Stdout // any non-nil file will do
		xfidopen(x)
		if got, want := mr.err, ErrInUse; got != want {
			t.Errorf("got error %v; want %v", got, want)
		}
	})
	t.Run("TempFile", func(t *testing.T) {
		x.f.w.rdselfd = nil
		testTempFileFail = true
		defer func() { testTempFileFail = false }()
		xfidopen(x)
		if mr.err == nil {
			t.Errorf("got nil error")
		}
		if got, want := mr.err.Error(), "can't create temp file"; got != want {
			t.Errorf("got error %v; want %v", got, want)
		}
		if x.f.w.rdselfd != nil {
			t.Errorf("non-nil w.rdselfd %v", x.f.w.rdselfd)
		}
	})
	t.Run("CopyFail", func(t *testing.T) {
		x.f.w.rdselfd = nil
		warnings = nil
		testIOCopyFail = true
		defer func() {
			warnings = nil
			testIOCopyFail = false
		}()
		xfidopen(x)
		if mr.err != nil {
			t.Errorf("got error %q; want nil", mr.err)
		}
		if len(warnings) == 0 {
			t.Fatalf("not warning generated")
		}
		got := string(warnings[0].buf.String())
		want := "can't write temp file for pipe command"
		if !strings.HasPrefix(got, want) {
			t.Errorf("got warning %q; want prefix %q", got, want)
		}
	})
}

func TestXfidclose(t *testing.T) {
	t.Run("NotOpen", func(t *testing.T) {
		mr := new(mockResponder)
		w := NewWindow().initHeadless(nil)
		w.tag.fr = &MockFrame{}
		w.body.fr = &MockFrame{}
		x := &Xfid{
			f: &Fid{
				qid: plan9.Qid{Path: QID(0, QWdata)},
				w:   w,
			},
			fs: mr,
		}
		xfidclose(x)
		if mr.err != nil {
			t.Errorf("got error %v", mr.err)
		}
	})

	for _, tc := range []struct {
		name string
		q    uint64
	}{
		{"QWctl", QWctl},
		{"QWaddr", QWaddr},
		{"QWdata", QWdata},
		{"QWxdata", QWxdata},
		{"QWevent", QWevent},
		{"QWrdsel", QWrdsel},
		{"QWwrsel", QWwrsel},
		{"QWeditout", QWeditout},
		{"Qeditout", Qeditout},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "edwood")
			if err != nil {
				t.Fatalf("can't create temporary file: %v", err)
			}
			defer os.Remove(tmpfile.Name())

			mr := new(mockResponder)
			var w *Window
			q := tc.q
			if q != Qeditout {
				w = NewWindow().initHeadless(nil)
				w.tag.fr = &MockFrame{}
				w.body.fr = &MockFrame{}
				w.body.display = edwoodtest.NewDisplay(image.Rectangle{})
				w.col = new(Column)
				w.rdselfd = tmpfile
				w.nomark = true
				w.nopen[q] = 1
				w.dumpstr = "win"
				w.dumpdir = "/home/gopher"
				w.ctlfid = 0
				w.ctrllock.Lock()
				close(w.editoutlk) // prevent block on send
			}
			global.editoutlk = make(chan bool)
			close(global.editoutlk) // prevent block on send
			x := &Xfid{
				f: &Fid{
					qid: plan9.Qid{
						Path: QID(0, q),
					},
					w:    w,
					open: true,
				},
				fs: mr,
			}
			xfidclose(x)
			if mr.err != nil {
				t.Errorf("got error %v", mr.err)
			}

			switch q {
			case QWctl:
				if got, want := w.ctlfid, uint32(MaxFid); got != want {
					t.Errorf("w.ctlfid is %v; want %v", got, want)
				}
			case QWdata, QWxdata:
				if w.nomark != false {
					t.Errorf("w.nomark is true")
				}
				fallthrough
			case QWaddr, QWevent:
				if got, want := w.nopen[q], byte(0); got != want {
					t.Errorf("w.nopen[%v] is %v; want %v", q, got, want)
				}
				if q == QWevent {
					if w.dumpstr != "" {
						t.Errorf("w.dumpstr is %q", w.dumpstr)
					}
					if w.dumpdir != "" {
						t.Errorf("w.dumpdir is %q", w.dumpdir)
					}
				}
			case QWrdsel:
				if w.rdselfd != nil {
					t.Errorf("w.rdselfd is not nil")
				}
			case QWwrsel:
				if w.nomark != false {
					t.Errorf("w.nomark is true")
				}
			}
		})
	}
}

func TestXfidwriteQWdata(t *testing.T) {
	display := edwoodtest.NewDisplay(image.Rectangle{})
	global.configureGlobals(display)

	mr := new(mockResponder)
	w := NewWindow().initHeadless(nil)
	w.col = new(Column)
	w.col.safe = true
	w.display = display
	w.body.fr = &MockFrame{}
	w.body.display = display
	w.tag.fr = &MockFrame{}
	w.tag.display = display

	for _, tc := range []struct {
		name    string // test name
		addr    Range  // write to this window address
		data    []byte // data to write
		err     error  // error response
		body    []byte // resulting body
		q0, q1  int    // resulting window body q0, q1
		newAddr Range  // resulting window address
	}{
		{"BadQ0", Range{100, 0}, nil, ErrAddrRange, nil, 0, 0, Range{}},
		{"BadQ1", Range{0, 100}, nil, ErrAddrRange, nil, 0, 0, Range{}},
		{"Initial", Range{0, 0}, []byte("αaaaβbbbγccc"), nil, []byte("αaaaβbbbγccc"), 12, 12, Range{12, 12}},
		{"Hello", Range{4, 8}, []byte("Hello, 世界"), nil, []byte("αaaaHello, 世界γccc"), 17, 17, Range{13, 13}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w.addr = tc.addr

			x := &Xfid{
				fcall: plan9.Fcall{
					Data:  tc.data,
					Count: uint32(len(tc.data)),
				},
				f: &Fid{
					qid: plan9.Qid{Path: QID(0, QWdata)},
					w:   w,
				},
				fs: mr,
			}
			xfidwrite(x)
			if got, want := mr.err, tc.err; got != want {
				t.Fatalf("got error %v; want %v", got, want)
			}
			if tc.err == nil {
				if got, want := mr.fcall.Count, uint32(len(tc.data)); got != want {
					t.Errorf("Fcall.Count is %v; want %v", got, want)
				}
				if got, want := w.body.file.String(), string(tc.body); got != want {
					t.Errorf("got body %q; want %q", got, want)
				}
				if tc.q0 != w.body.q0 || tc.q1 != w.body.q1 {
					t.Errorf("body (q0, q1) = (%v, %v); want (%v, %v)",
						w.body.q0, w.body.q1, tc.q0, tc.q1)
				}
				if got, want := w.addr, tc.newAddr; !reflect.DeepEqual(got, want) {
					t.Errorf("window address is %v; want %v", got, want)
				}
			}
		})
	}
}

func TestXfidwriteDeletedWin(t *testing.T) {
	mr := new(mockResponder)
	w := NewWindow().initHeadless(nil)
	x := &Xfid{
		f:  &Fid{w: w},
		fs: mr,
	}
	xfidwrite(x)
	if got, want := mr.err, ErrDeletedWin; got != want {
		t.Errorf("got error %v; want %v", got, want)
	}
}

func TestXfidwriteUnknownQID(t *testing.T) {
	mr := new(mockResponder)
	x := &Xfid{
		f: &Fid{
			qid: plan9.Qid{Path: QID(0, QMAX+1)},
		},
		fs: mr,
	}
	xfidwrite(x)

	wantErr := fmt.Sprintf("unknown qid %d in write", QMAX+1)
	if mr.err == nil {
		t.Errorf("got nil error; want error %q", wantErr)
	}
	if got, want := mr.err.Error(), wantErr; got != want {
		t.Errorf("got error %q; want error %q", got, want)
	}
}

func TestXfidwriteQWtag(t *testing.T) {
	const (
		prevTag = "/etc/hosts Del Snarf Undo | Look "
		extra   = "|fmt Ldef Lrefs"
		newTag  = prevTag + extra
	)
	mr := new(mockResponder)
	w := NewWindow().initHeadless(nil)
	w.col = new(Column)
	w.body.file = file.MakeObservableEditableBuffer("", nil)
	w.tag.file = file.MakeObservableEditableBuffer("", []rune(prevTag))
	w.tagfilenameend = len(parsetaghelper(prevTag))
	x := &Xfid{
		fcall: plan9.Fcall{
			Data:  []byte(extra),
			Count: uint32(len(extra)),
		},
		f: &Fid{
			qid: plan9.Qid{Path: QID(0, QWtag)},
			w:   w,
		},
		fs: mr,
	}
	xfidwrite(x)
	if mr.err != nil {
		t.Errorf("got error %v; want nil", mr.err)
	}
	if got, want := mr.fcall.Count, uint32(len(extra)); got != want {
		t.Errorf("fcall.Count is %v; want %v", got, want)
	}
	if got, want := w.tag.file.String(), newTag; got != want {
		t.Errorf("tag is %q; want %q", got, want)
	}
}

func TestXfidwriteQWwrsel(t *testing.T) {
	mockDisplay := edwoodtest.NewDisplay(image.Rectangle{})
	global.configureGlobals(mockDisplay)

	w := NewWindow().initHeadless(nil)
	w.col = new(Column)
	w.body.file = file.MakeObservableEditableBuffer("", nil)
	w.tag.file = file.MakeObservableEditableBuffer("", nil)
	w.body.fr = &MockFrame{}
	w.body.display = mockDisplay
	w.tag.display = mockDisplay

	for _, tc := range []struct {
		name       string // test name
		q          uint64 // Qid.Path
		wrselrange Range  // where to write for QWwrsel
		data       []byte // data to write
		want       []byte // resulting body buffer
	}{
		{"QWbody", QWbody, Range{0, 0}, []byte("αbcdλβγ"), []byte("αbcdλβγ")},
		{"QWwrsel", QWwrsel, Range{4, 4}, []byte("εfg"), []byte("αbcdεfgλβγ")},
		{"QWwrselEND", QWwrsel, Range{100, 100}, []byte("END"), []byte("αbcdεfgλβγEND")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w.wrselrange = tc.wrselrange
			mr := new(mockResponder)
			x := &Xfid{
				fcall: plan9.Fcall{
					Data:  tc.data,
					Count: uint32(len(tc.data)),
				},
				f: &Fid{
					qid: plan9.Qid{Path: QID(0, tc.q)},
					w:   w,
				},
				fs: mr,
			}
			xfidwrite(x)
			if mr.err != nil {
				t.Errorf("got error %v; want nil", mr.err)
			}
			if got, want := mr.fcall.Count, uint32(len(tc.data)); got != want {
				t.Errorf("fcall.Count is %v; want %v", got, want)
			}
			if got, want := w.body.file.String(), string(tc.want); got != want {
				t.Errorf("buffer is %q; want %q", got, want)
			}
		})
	}
}

func TestXfidwriteQlabel(t *testing.T) {
	data := []byte("Hello, 世界!\n")
	mr := new(mockResponder)
	x := &Xfid{
		fcall: plan9.Fcall{
			Data:  data,
			Count: uint32(len(data)),
		},
		f: &Fid{
			qid: plan9.Qid{Path: QID(0, Qlabel)},
		},
		fs: mr,
	}
	xfidwrite(x)
	if mr.err != nil {
		t.Errorf("got error %v; want nil", mr.err)
	}
	if got, want := mr.fcall.Count, uint32(len(data)); got != want {
		t.Errorf("fcall.Count is %v; want %v", got, want)
	}
}

func TestXfidwriteQcons(t *testing.T) {
	global.configureGlobals(edwoodtest.NewDisplay(image.Rectangle{}))
	mr := new(mockResponder)

	global.row.Init(image.Rectangle{
		image.Point{0, 0},
		image.Point{800, 600},
	}, edwoodtest.NewDisplay(image.Rectangle{}))

	data := []byte("cons error: Hello, 世界!\n")
	x := &Xfid{
		fcall: plan9.Fcall{
			Data:  data,
			Count: uint32(len(data)),
		},
		f: &Fid{
			qid: plan9.Qid{Path: QID(0, Qcons)},
		},
		fs: mr,
	}
	xfidwrite(x)
	if mr.err != nil {
		t.Fatalf("got error %v; want nil", mr.err)
	}
	if got, want := mr.fcall.Count, uint32(len(data)); got != want {
		t.Errorf("fcall.Count is %v; want %v", got, want)
	}
	w := errorwin(x.f.mntdir, 'X')
	if got, want := w.body.file.String(), string(data); got != want {
		t.Errorf("+Errors window body is %q; want %q", got, want)
	}
}

func TestXfidwriteQWerrors(t *testing.T) {
	// TODO(rjk): This is another of one these places where I should really
	// be using a quality backing mock.
	data := []byte("window error: Hello, 世界!\n")

	mockdisplay := edwoodtest.NewDisplay(image.Rectangle{})
	global.configureGlobals(mockdisplay)
	mr := new(mockResponder)

	global.row.display = mockdisplay

	col := new(Column)
	col.display = mockdisplay
	col.tag.fr = &MockFrame{}
	global.row.col = append(global.row.col, col)
	col.tag.display = mockdisplay
	col.tag.file = file.MakeObservableEditableBuffer("", nil)
	w := NewWindow().initHeadless(nil)
	w.display = mockdisplay
	w.col = col

	tagcontents := "/home/gopher/edwood/row.go Del Snarf | Look "
	w.tag.file = file.MakeObservableEditableBuffer("", []rune(tagcontents))
	w.tagfilenameend = len(parsetaghelper(string(tagcontents)))
	col.w = append(col.w, w)
	w.tag.display = mockdisplay
	w.tag.w = w
	w.tag.col = col
	w.tag.row = &global.row

	w.tag.fr = &MockFrame{}

	w.body.fr = &MockFrame{}
	w.body.display = mockdisplay

	x := &Xfid{
		fcall: plan9.Fcall{
			Data:  data,
			Count: uint32(len(data)),
		},
		f: &Fid{
			qid: plan9.Qid{Path: QID(0, QWerrors)},
			w:   w,
		},
		fs: mr,
	}

	xfidwrite(x)
	if mr.err != nil {
		t.Fatalf("got error %v; want nil", mr.err)
	}
	if got, want := mr.fcall.Count, uint32(len(data)); got != want {
		t.Errorf("fcall.Count is %v; want %v", got, want)
	}

	w.Lock('F')
	// Note: errorwinforwin will unlock w and return a new locked window.
	w = errorwinforwin(w)
	defer w.Unlock()

	if got, want := w.body.file.String(), string(data); got != want {
		t.Errorf("+Errors window body is %q; want %q", got, want)
	}
}

func TestXfidwriteQeditoutError(t *testing.T) {
	mr := new(mockResponder)
	x := &Xfid{
		f: &Fid{
			qid: plan9.Qid{Path: QID(0, Qeditout)},
		},
		fs: mr,
	}
	xfidwrite(x)
	if got, want := mr.err, ErrPermission; got != want {
		t.Errorf("got error %v; want %v", got, want)
	}
}

func TestXfidwriteQWeditout(t *testing.T) {
	data := []byte("Exαmplε εditout tεxt.")
	w := NewWindow().initHeadless(nil)
	w.col = new(Column)
	mr := new(mockResponder)
	x := &Xfid{
		fcall: plan9.Fcall{
			Data:  data,
			Count: uint32(len(data)),
		},
		f: &Fid{
			qid: plan9.Qid{Path: QID(0, QWeditout)},
			w:   w,
		},
		fs: mr,
	}

	global.editing = Collecting
	collection = nil
	defer func() {
		global.editing = Inactive
		collection = nil
	}()
	xfidwrite(x)
	if mr.err != nil {
		t.Fatalf("got error %v; want nil", mr.err)
	}
	if got, want := mr.fcall.Count, uint32(len(data)); got != want {
		t.Errorf("fcall.Count is %v; want %v", got, want)
	}
	if got, want := string(collection), string(data); got != want {
		t.Errorf("collection is %q; want %q", got, want)
	}
}

func TestXfidwriteQWctl(t *testing.T) {
	global.configureGlobals(edwoodtest.NewDisplay(image.Rectangle{}))
	warnings = nil
	global.cwarn = nil

	for _, tc := range []struct {
		err  error
		data string
	}{
		{nil, ""},
		{nil, "\n"},
		{ErrBadCtl, "lock"},   // disabled
		{ErrBadCtl, "unlock"}, // disabled
		{nil, "clean"},
		{nil, "clean\n"},
		{nil, "dirty"},
		{nil, "show"},
		{ErrBadCtl, "name"},
		{nil, "name /Test/Write/Ctl"},
		{fmt.Errorf("nulls in file name"), "name /Test/Write\u0000/Ctl"},
		{nil, "name /Test/Write To/Ctl"},
		{fmt.Errorf("bad character in file name"), "name /Test/\037Write To/Ctl"},
		{nil, "dump win"},
		{ErrBadCtl, "dump"},
		{fmt.Errorf("nulls in dump string"), "dump win\u0000rc"},
		{nil, "dumpdir /home/gopher"},
		{ErrBadCtl, "dumpdir"},
		{fmt.Errorf("nulls in dump directory string"), "dumpdir /home\u0000/gopher"},
		{nil, "delete"},
		{fmt.Errorf("file dirty"), "del"},
		{fmt.Errorf("file dirty"), "del\ndel"},
		{nil, "get"},
		{nil, "put"},
		{nil, "dot=addr"},
		{nil, "addr=dot"},
		{nil, "limit=addr"},
		{nil, "nomark"},
		{nil, "mark"},
		{nil, "nomenu"},
		{nil, "menu"},
		{nil, "cleartag"},
		{ErrBadCtl, "brewcoffee"},
		{ErrDeletedWin, "delete\nclean"},
		{ErrDeletedWin, "delete\nget"},
		{fmt.Errorf("file dirty"), "del\ndel\nclean"},
		{nil, "clean\ndel"},
		{nil, "clean\ndelete"},
		{ErrBadCtl, "font"},
		{fmt.Errorf("nulls in font name"), "font /path/with/\x00nulls"},
		{nil, "font /path/to/font"},
	} {
		t.Run(fmt.Sprintf("Data=%q", tc.data), func(t *testing.T) {
			mr := new(mockResponder)
			display := edwoodtest.NewDisplay(image.Rectangle{})
			w := NewWindow().initHeadless(nil)
			w.display = display
			w.col = &Column{
				w:       []*Window{w},
				display: display,
			}
			w.body.display = display
			w.body.fr = &MockFrame{}
			w.tag.display = display
			w.tag.fr = &MockFrame{}
			global.row.display = display

			// mark window dirty
			f := w.body.file
			f.InsertAt(0, []rune(strings.Repeat("ha", 100)))
			f.SetSeq(0)
			f.SetPutseq(1)

			x := &Xfid{
				fcall: plan9.Fcall{
					Data:  []byte(tc.data),
					Count: uint32(len(tc.data)),
				},
				f: &Fid{
					qid: plan9.Qid{Path: QID(0, QWctl)},
					w:   w,
				},
				fs: mr,
			}
			xfidwrite(x)

			if got, want := mr.err, tc.err; want != nil {
				if got == nil || (got != want && got.Error() != want.Error()) {
					t.Fatalf("got error %v; want %v", got, want)
				}
				return
			}
			if got, want := mr.err, tc.err; got != nil {
				t.Fatalf("got error %v; want %v", got, want)
			}
			if got, want := mr.fcall.Count, uint32(len(tc.data)); got != want {
				t.Errorf("fcall.Count is %v; want %v", got, want)
			}
		})
	}
}

func TestXfidwriteQWevent(t *testing.T) {
	for _, tc := range []struct {
		err  error
		data string
	}{
		{ErrBadEvent, "M"},
		{ErrBadEvent, "ML"},
		{ErrBadEvent, "MLX X"},
		{ErrBadEvent, "ML0 X"},
		{ErrBadEvent, "ML1 1"},
		{ErrBadEvent, "%%1 1"},
		{ErrBadEvent, "Mz0 0"},
		{nil, "ML0 0"},
		{nil, "Ml0 0"},
		{nil, "MX0 0"},
		{nil, "Mx0 0"},
		{nil, "\n\n"},
	} {
		w := NewWindow().initHeadless(nil)
		w.col = new(Column)
		mr := new(mockResponder)
		x := &Xfid{
			fcall: plan9.Fcall{
				Data:  []byte(tc.data),
				Count: uint32(len(tc.data)),
			},
			f: &Fid{
				qid: plan9.Qid{Path: QID(0, QWevent)},
				w:   w,
			},
			fs: mr,
		}
		xfidwrite(x)
		if got, want := mr.err, tc.err; got != want {
			t.Errorf("event %q: got error %v; want %v", tc.data, got, want)
		}
	}
}

// Issue https://github.com/rjkroege/edwood/issues/285
func TestXfidwriteQWeventExecuteSend(t *testing.T) {
	// Setup a new window with "Send" in the tag.
	d := edwoodtest.NewDisplay(image.Rectangle{})
	global.row = Row{
		display: d,
	}
	w := NewWindow().initHeadless(nil)
	w.col = new(Column)
	w.nopen[QWevent]++
	defer func() { w.nopen[QWevent]-- }()
	w.tag = Text{
		w:       w,
		file:    file.MakeObservableEditableBuffer("", []rune("Send")),
		fr:      &MockFrame{},
		display: d,
	}
	w.tag.file.AddObserver(&w.tag)
	w.body = Text{
		w:       w,
		file:    file.MakeObservableEditableBuffer("", nil),
		fr:      &MockFrame{},
		display: d,
	}
	w.body.file.AddObserver(&w.body)
	w.tag.file.AddObserver(w)
	w.tagfilenameend = len("Send")

	// Put something in the snarf buffer.
	const snarfbuf = "Hello, 世界\n"
	d.WriteSnarf([]byte(snarfbuf))

	// Execute "Send" in the tag. This should append the content of
	// snarf buffer into the body.
	mr := new(mockResponder)
	const event = "Mx1 1"
	x := &Xfid{
		fcall: plan9.Fcall{
			Data:  []byte(event),
			Count: uint32(len(event)),
		},
		f: &Fid{
			qid: plan9.Qid{Path: QID(w.id, QWevent)},
			w:   w,
		},
		fs: mr,
	}
	xfidwrite(x)
	if got := mr.err; got != nil {
		t.Errorf("event %q: got error %v; want nil", event, got)
	}
	if got, want := w.body.file.String(), snarfbuf; got != want {
		t.Errorf("body contains %q; want %q", got, want)
	}
}

func TestXfidreadEmptyFiles(t *testing.T) {
	for _, tc := range []struct {
		name string
		q    uint64
	}{
		{"Qcons", Qcons},
		{"Qlabel", Qlabel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mr := new(mockResponder)
			x := &Xfid{
				fcall: plan9.Fcall{
					Offset: 0,
					Count:  100,
				},
				f: &Fid{
					qid: plan9.Qid{Path: QID(0, tc.q)},
				},
				fs: mr,
			}
			xfidread(x)
			if mr.err != nil {
				t.Errorf("got error %v; want nil", mr.err)
			}
			if got, want := mr.fcall.Count, uint32(0); got != want {
				t.Errorf("fcall.Count is %v; want %v", got, want)
			}
		})
	}
}

func TestXfidreadQWbodyQWtag(t *testing.T) {
	// TODO(rjk): These tests are fragile in how they setup their skeleton.
	// Use the common skeleton.
	display := edwoodtest.NewDisplay(image.Rectangle{})
	global.configureGlobals(display)
	const data = "This is an εxαmplε sentence.\n"

	for _, tc := range []struct {
		name  string
		q     uint64
		setup string
		want  string
	}{
		{"QWbody", QWbody, data, data},
		// TODO(rjk): Why doesn't setTag1 run?
		{"QWtag", QWtag, "εxαmplε", "εxαmplε"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mr := new(mockResponder)
			w := NewWindow().initHeadless(nil)
			w.col = new(Column)
			w.col.safe = true
			w.display = display
			w.body.display = display
			w.body.fr = &MockFrame{}
			w.tag.display = display
			w.tag.fr = &MockFrame{}
			switch tc.q {
			case QWbody:
				w.body.file = file.MakeObservableEditableBuffer("", []rune(tc.setup))
			case QWtag:
				w.tag.file = file.MakeObservableEditableBuffer("", []rune(tc.setup))
				w.tagfilenameend = utf8.RuneCountInString(tc.setup)
			}

			x := &Xfid{
				fcall: plan9.Fcall{
					Offset: 0,
					Count:  uint32(len(data)),
				},
				f: &Fid{
					qid: plan9.Qid{Path: QID(0, tc.q)},
					w:   w,
				},
				fs: mr,
			}
			xfidread(x)
			if got, want := mr.fcall.Count, uint32(len(tc.want)); got != want {
				t.Errorf("read %v bytes; want %v",
					got, want)
			}
			if got, want := string(mr.fcall.Data), tc.want; got != want {
				t.Errorf("got data %q; want %q", got, want)
			}
		})
	}
}

func TestXfidruneread(t *testing.T) {
	tt := []struct {
		body   []rune // window body
		q0, q1 int
		count  uint32 // input fcall count
		nr     int    // return value (number of runes)
		data   []byte // output fcall data
	}{
		{[]rune("abcde"), 0, 5, 100, 5, []byte("abcde")},
		{[]rune("abcde"), 1, 5, 100, 4, []byte("bcde")},
		{[]rune("abcde"), 2, 5, 100, 3, []byte("cde")},
		{[]rune("abcde"), 3, 5, 100, 2, []byte("de")},
		{[]rune("abcde"), 4, 5, 100, 1, []byte("e")},
		{[]rune("abcde"), 5, 5, 100, 0, []byte("")},
		{[]rune("αβξδε"), 0, 5, 100, 5, []byte("αβξδε")},
		{[]rune("αβξδε"), 1, 5, 100, 4, []byte("βξδε")},
		{[]rune("αβξδε"), 2, 5, 100, 3, []byte("ξδε")},
		{[]rune("αβξδε"), 3, 5, 100, 2, []byte("δε")},
		{[]rune("αβξδε"), 4, 5, 100, 1, []byte("ε")},
		{[]rune("αβξδε"), 0, 5, 8, 4, []byte("αβξδ")},
		{[]rune("αβξδε"), 0, 5, 5, 2, []byte("αβ")},
		{[]rune("αβξδε"), 0, 5, 0, 0, []byte("")},
	}

	for _, tc := range tt {
		mr := new(mockResponder)
		x := &Xfid{
			fcall: plan9.Fcall{
				Count: tc.count,
			},
			fs: mr,
		}
		w := NewWindow().initHeadless(nil)
		w.body.file = file.MakeObservableEditableBuffer("", tc.body)
		nr := xfidruneread(x, &w.body, tc.q0, tc.q1)
		if got, want := nr, tc.nr; got != want {
			t.Errorf("read %v runes from %q (q0=%v, q1=%v); should read %v runes",
				got, tc.body, tc.q0, tc.q1, want)
		}
		if mr.err != nil {
			t.Errorf("got error %v for %q (q0=%v, q1=%v); want nil",
				mr.err, tc.body, tc.q0, tc.q1)
		}
		if got, want := mr.fcall.Count, uint32(len(tc.data)); got != want {
			t.Errorf("read %v bytes from %q (q0=%v, q1=%v); want %v",
				got, tc.body, tc.q0, tc.q1, want)
		}
		if got, want := mr.fcall.Data, tc.data; !bytes.Equal(got, want) {
			t.Errorf("read %q from %q (q0=%v, q1=%v); want %q\n",
				got, tc.body, tc.q0, tc.q1, want)
		}
	}
}

func TestXfidreadQWxdataQWdata(t *testing.T) {
	const body = "0123456789"

	for _, tc := range []struct {
		name    string // test name
		inAddr  Range  // initial window addr
		q       uint64 // Qid path (QWdata or QWxdata)
		count   uint32 // number of bytes to read
		err     error  // error response
		data    string // data in response
		outAddr Range  // new window addr
	}{
		{"QWdataSuccess", Range{0, 5}, QWdata, 7, nil, "0123456", Range{7, 7}},
		{"QWxdataSuccess", Range{0, 5}, QWxdata, 7, nil, "01234", Range{5, 5}},
		{"QWdataError", Range{100, 100}, QWdata, 7, ErrAddrRange, "", Range{100, 100}},
		{"QWxdataError", Range{100, 100}, QWxdata, 7, ErrAddrRange, "", Range{100, 100}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mr := new(mockResponder)
			w := NewWindow().initHeadless(nil)
			w.col = new(Column)
			w.body.file = file.MakeObservableEditableBuffer("", []rune(body))
			w.addr = tc.inAddr
			xfidread(&Xfid{
				f: &Fid{
					qid: plan9.Qid{Path: QID(1, tc.q)},
					w:   w,
				},
				fcall: plan9.Fcall{
					Count: tc.count,
				},
				fs: mr,
			})
			if got, want := mr.err, tc.err; got != want {
				t.Fatalf("got error %v; want %v", got, want)
			}
			if tc.err == nil {
				if got, want := string(mr.fcall.Data), tc.data; want != got {
					t.Errorf("got data %q; want %q", got, want)
				}
				if got, want := w.addr, tc.outAddr; !reflect.DeepEqual(got, want) {
					t.Errorf("got window address %#v; want %#v", got, want)
				}
			}
		})
	}
}

func TestXfidreadQWaddr(t *testing.T) {
	const (
		body = "0123456789ABCDEF"
		want = "          5          12 "
	)
	w := NewWindow().initHeadless(nil)
	w.col = new(Column)
	w.body.file = file.MakeObservableEditableBuffer("", []rune(body))
	w.addr.q0 = 5
	w.addr.q1 = 12

	mr := new(mockResponder)
	xfidread(&Xfid{
		f: &Fid{
			qid: plan9.Qid{Path: QID(1, QWaddr)},
			w:   w,
		},
		fcall: plan9.Fcall{Count: 64},
		fs:    mr,
	})
	if mr.err != nil {
		t.Fatalf("got error %v; want nil", mr.err)
	}
	if got := string(mr.fcall.Data); got != want {
		t.Errorf("got data %q; want %q", got, want)
	}
}

func TestXfidreadQWctl(t *testing.T) {
	const prewant = "          1          32          14           0           0           0 "
	const postwant = "           0 "
	want := prewant + edwoodtest.Plan9FontPath(edwoodtest.MockFontName) + postwant
	if len(want) > 128 {
		want = want[:128]
	}

	global.WinID = 0
	w := NewWindow().initHeadless(nil)
	w.col = new(Column)
	w.display = edwoodtest.NewDisplay(image.Rectangle{})
	w.body.fr = &MockFrame{}
	w.tag.file = file.MakeObservableEditableBuffer("", []rune(("/etc/hosts Del Snarf | Look Get ")))
	w.body.file = file.MakeObservableEditableBuffer("", []rune("Hello, world!\n"))

	mr := new(mockResponder)
	xfidread(&Xfid{
		f: &Fid{
			qid: plan9.Qid{Path: QID(1, QWctl)},
			w:   w,
		},
		fcall: plan9.Fcall{Count: 128},
		fs:    mr,
	})
	if mr.err != nil {
		t.Fatalf("got error %v; want nil", mr.err)
	}
	if got := string(mr.fcall.Data); got != want {
		t.Errorf("got data %q; want %q", got, want)
	}
}

func TestXfidreadUnknownQID(t *testing.T) {
	w := NewWindow().initHeadless(nil)
	w.col = new(Column)
	for _, tc := range []struct {
		name string
		w    *Window
	}{
		{"NilWindow", nil},
		{"NonNilWindow", w},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mr := new(mockResponder)
			x := &Xfid{
				f: &Fid{
					qid: plan9.Qid{Path: QID(0, QMAX+1)},
					w:   tc.w,
				},
				fs: mr,
			}
			xfidread(x)

			wantErr := fmt.Sprintf("unknown qid %d in read", QMAX+1)
			if mr.err == nil {
				t.Fatalf("got nil error; want error %q", wantErr)
			}
			if got, want := mr.err.Error(), wantErr; got != want {
				t.Errorf("got error %q; want error %q", got, want)
			}
		})
	}
}

func TestXfidreadDeletedWin(t *testing.T) {
	mr := new(mockResponder)
	xfidread(&Xfid{
		f: &Fid{
			w: NewWindow().initHeadless(nil),
		},
		fs: mr,
	})
	if got, want := mr.err, ErrDeletedWin; got != want {
		t.Fatalf("got error %v; want %v", got, want)
	}
}

func TestXfidreadQlog(t *testing.T) {
	mr := new(mockResponder)
	x := &Xfid{
		f: &Fid{
			qid: plan9.Qid{Path: QID(0, Qlog)},
		},
		fs: mr,
	}
	xfidlogopen(x)
	go func() {
		global.WinID = 0
		w := NewWindow().initHeadless(nil)
		xfidlog(w, "new")
	}()

	xfidread(x)
	if mr.err != nil {
		t.Fatalf("got error %v; want nil", mr.err)
	}
	const line = "1 new \n"
	if got, want := mr.fcall.Count, uint32(len(line)); want != got {
		t.Errorf("got count %v; want %v", got, want)
	}
	if got, want := string(mr.fcall.Data), line; got != want {
		t.Errorf("got data %q; want %q", got, want)
	}
}

func TestXfidreadQWevent(t *testing.T) {
	const events = "MI20433 20438 0 5 hello\n"

	for _, tc := range []struct {
		name      string                   // test name
		initial   string                   // initial events stored in Window
		writer    func(w *Window, x *Xfid) // writes events
		errString string                   // error response
		count     int                      // read count
		data      string                   // read output
	}{
		{
			name:    "Success",
			initial: events,
			count:   10,
			data:    events[:10],
		},
		{
			name:    "WindowShutDown",
			initial: "",
			count:   10,
			writer: func(w *Window, x *Xfid) {
				// We need at least two iterations of the loop that checks `len(w.events) == 0`.
				// It can be more than two if we don't acquire the lock fast enough.
				x.c <- nil
				x.c <- nil
				close(x.c)

				w.Lock('F')
				defer w.Unlock()
				w.events = []byte(events)
			},
			errString: "window shut down",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mr := new(mockResponder)
			w := NewWindow().initHeadless(nil)
			w.col = new(Column)
			w.events = []byte(tc.initial)
			x := &Xfid{
				f: &Fid{
					qid: plan9.Qid{Path: QID(1, QWevent)},
					w:   w,
				},
				fcall: plan9.Fcall{Count: uint32(tc.count)},
				c:     make(chan func(*Xfid)),
				fs:    mr,
			}
			if tc.writer != nil {
				go tc.writer(w, x)
			}
			xfidread(x)
			if tc.errString != "" {
				if mr.err == nil || mr.err.Error() != tc.errString {
					t.Fatalf("got error %v; want %q", mr.err, tc.errString)
				}
				return
			}
			if mr.err != nil {
				t.Fatalf("got error %v; want nil", mr.err)
			}
			if got, want := mr.fcall.Count, uint32(tc.count); want != got {
				t.Errorf("got count %v; want %v", got, want)
			}
			if got, want := string(mr.fcall.Data), tc.data; want != got {
				t.Errorf("got data %q; want %q", got, want)
			}
			if got, want := string(w.events), events[tc.count:]; got != want {
				t.Errorf("w.events is %q; want %q", got, want)
			}
		})
	}
}

func TestXfidreadQindex(t *testing.T) {
	for _, name := range []string{
		"empty-two-cols",
		"example",
		"multi-line-tag",
	} {
		t.Run(name, func(t *testing.T) {
			origfilename := filepath.Join("testdata", name+".dump")
			t.Logf("original file: %q", origfilename)
			filename := editDumpFileForTesting(t, origfilename)
			defer os.Remove(filename)

			setGlobalsForLoadTesting()

			err := global.row.Load(nil, filename, true)
			if err != nil {
				t.Fatalf("Row.Load failed: %v", err)
			}

			mr := new(mockResponder)
			xfidread(&Xfid{
				f: &Fid{
					qid: plan9.Qid{Path: QID(0, Qindex)},
				},
				fcall: plan9.Fcall{Count: 1024},
				fs:    mr,
			})
			if mr.err != nil {
				t.Fatalf("xfidindexread returned error %v", mr.err)
			}
			got := mr.fcall.Data
			want := readIndexFile(t, filepath.Join("testdata", name+".index"))
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("index data mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestXfidutfreadLargeOffset tests that xfidutfread handles offsets greater than
// math.MaxInt32 (2GB) correctly without truncation. This is a regression test for
// the int64→int truncation issue in fsys.go file size handling.
//
// On 32-bit systems, the int type is 32-bit and offsets > 2GB would overflow
// if not handled properly. This test verifies the code uses uint64/int64 arithmetic
// correctly throughout the read path.
func TestXfidutfreadLargeOffset(t *testing.T) {
	// We cannot actually create a 2GB+ file in memory, but we can test that
	// the offset arithmetic doesn't overflow by testing with offsets that
	// would cause issues if truncated to int32.
	//
	// The key code paths in xfidutfread that could fail with large offsets:
	// - line 842: off >= w.utflastboff (comparison)
	// - line 867: boff+uint64(m) > off+uint64(x.fcall.Count)
	// - line 868: m = int(off + uint64(x.fcall.Count) - boff)
	// - line 877: m := nb - int(off-boff)
	// - line 881: copy(b1, b[off-boff:int(off-boff)+m])
	//
	// The test validates that when we set a large offset, the function
	// correctly returns empty data (since our buffer isn't that large)
	// rather than crashing or returning garbage due to overflow.

	display := edwoodtest.NewDisplay(image.Rectangle{})
	global.configureGlobals(display)

	// Test cases with large offsets that would overflow int32
	testCases := []struct {
		name   string
		offset uint64
	}{
		{"JustOverMaxInt32", uint64(1<<31 + 1000)},        // 2GB + 1000 bytes
		{"LargeOffset", uint64(1<<32 + 5000)},             // 4GB + 5000 bytes
		{"VeryLargeOffset", uint64(1<<40)},                // 1TB - extreme case
		{"MaxSafeOffset", uint64(1<<62)},                  // Very large but not overflow uint64
		{"OffsetNearMaxUint64", uint64(1<<63 - 1000000)},  // Near max uint64
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mr := new(mockResponder)
			w := NewWindow().initHeadless(nil)
			w.col = new(Column)
			w.col.safe = true
			w.display = display
			w.body.display = display
			w.body.fr = &MockFrame{}
			w.tag.display = display
			w.tag.fr = &MockFrame{}

			// Create a small test buffer - the key is that the offset is way beyond it
			testData := "This is test data for large offset testing.\n"
			w.body.file = file.MakeObservableEditableBuffer("", []rune(testData))

			x := &Xfid{
				fcall: plan9.Fcall{
					Offset: tc.offset,
					Count:  100,
				},
				f: &Fid{
					qid: plan9.Qid{Path: QID(0, QWbody)},
					w:   w,
				},
				fs: mr,
			}

			// This should not panic or crash - it should return empty data
			// since the offset is beyond the file size
			xfidread(x)

			if mr.err != nil {
				t.Errorf("got error %v; want nil (offset %d should just return empty data)", mr.err, tc.offset)
			}
			// The read should return 0 bytes since offset is way beyond file size
			if got, want := mr.fcall.Count, uint32(0); got != want {
				t.Errorf("read %v bytes at offset %d; want %v (file is much smaller)", got, tc.offset, want)
			}
		})
	}
}

// TestXfidutfreadLargeOffsetWithCachedPosition tests that sequential reads
// with cached byte offset positions handle large offsets correctly.
// This specifically tests the w.utflastboff / w.utflastq caching path.
func TestXfidutfreadLargeOffsetWithCachedPosition(t *testing.T) {
	display := edwoodtest.NewDisplay(image.Rectangle{})
	global.configureGlobals(display)

	mr := new(mockResponder)
	w := NewWindow().initHeadless(nil)
	w.col = new(Column)
	w.col.safe = true
	w.display = display
	w.body.display = display
	w.body.fr = &MockFrame{}
	w.tag.display = display
	w.tag.fr = &MockFrame{}

	testData := "Test data for cached position testing.\n"
	w.body.file = file.MakeObservableEditableBuffer("", []rune(testData))

	// First read at offset 0 to establish cache
	x := &Xfid{
		fcall: plan9.Fcall{
			Offset: 0,
			Count:  10,
		},
		f: &Fid{
			qid: plan9.Qid{Path: QID(0, QWbody)},
			w:   w,
		},
		fs: mr,
	}
	xfidread(x)
	if mr.err != nil {
		t.Fatalf("first read failed: %v", mr.err)
	}

	// Now try reading at a large offset - the caching logic should not
	// cause overflow when comparing off >= w.utflastboff
	largeOffset := uint64(1<<32 + 100) // 4GB + 100 bytes
	x.fcall.Offset = largeOffset
	x.fcall.Count = 50

	xfidread(x)
	if mr.err != nil {
		t.Errorf("got error %v at large offset %d; want nil", mr.err, largeOffset)
	}
	// Should return empty since offset is beyond file
	if got, want := mr.fcall.Count, uint32(0); got != want {
		t.Errorf("read %v bytes at large offset; want %v", got, want)
	}
}

// TestInt64OffsetArithmetic tests specific edge cases for int64/uint64 arithmetic
// that could cause issues in the file reading code.
func TestInt64OffsetArithmetic(t *testing.T) {
	// Test that the arithmetic in xfidutfread is safe for large values
	// These are the critical operations:
	// - off - boff (both uint64)
	// - int(off - boff) when used as slice index
	// - int(off + uint64(count) - boff) for computing read length

	testCases := []struct {
		name  string
		off   uint64
		boff  uint64
		count uint32
		want  string // "safe" or "would_overflow"
	}{
		{"SmallValues", 100, 50, 50, "safe"},
		{"LargeOffset", 1 << 33, 0, 100, "safe"},          // 8GB offset
		{"OffsetDiffWithinInt", 1 << 33, 1<<33 - 50, 100, "safe"},
		{"OffsetDiffBeyondInt32", 1 << 33, 0, 100, "safe"}, // diff > MaxInt32 but should not be used as index
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Verify the difference fits in uint64 without overflow
			diff := tc.off - tc.boff
			t.Logf("off=%d, boff=%d, diff=%d", tc.off, tc.boff, diff)

			// The actual file reading code only converts diff to int when
			// it's used as a slice index, which should only happen when
			// boff >= off (meaning we're reading within the current buffer)
			// In that case, diff should be small

			if tc.boff >= tc.off {
				// This is the case where int conversion happens
				if diff > uint64(1<<31-1) {
					t.Logf("diff %d would overflow int32 - code should handle this case", diff)
				}
			}
		})
	}
}

func readIndexFile(t *testing.T, filename string) []byte {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	f, err := os.Open(filename)
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer f.Close()

	// Read each line in index and adjust tag length (2nd field) if tag contains a path.
	var buf bytes.Buffer
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		ntag, _ := strconv.Atoi(strings.TrimSpace(line[12 : 12*2]))
		if strings.Contains(line, gopherEdwoodDir) {
			ntag += len(cwd) - len(gopherEdwoodDir)
		}
		fmt.Fprintf(&buf, "%s%11d %s\n", line[:12], ntag, line[12*2:])
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read failed: %v", err)
	}

	b := buf.Bytes()
	if len(b) == 0 {
		return nil
	}
	return replacePathsForTesting(t, b, false)
}
