package main

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"9fans.net/go/plan9"
	"github.com/rjkroege/edwood/internal/draw"
)

func TestXfidAlloc(t *testing.T) {

	cxfidalloc = make(chan *Xfid)
	cxfidfree = make(chan *Xfid)

	d := (draw.Display)(nil)
	go xfidallocthread(d)

	cxfidalloc <- (*Xfid)(nil) // Request an xfid
	x := <-cxfidalloc
	if x == nil {
		t.Errorf("Failed to get an Xfid")
	}
	cxfidfree <- x
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
		w.body.file.b = Buffer(tc.body)
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

func TestXfidreadQWxdata(t *testing.T) {
	const body = "αβξδεabcde"

	for _, tc := range []struct {
		q0  int
		err error
	}{
		{0, nil},
		{len([]rune(body)) + 1, ErrAddrRange},
	} {
		w := NewWindow().initHeadless(nil)
		w.col = new(Column)
		w.body.file.b = Buffer(body)
		w.addr.q0 = tc.q0
		w.addr.q1 = len([]rune(body))
		mr := new(mockResponder)
		xfidread(&Xfid{
			f: &Fid{
				qid: plan9.Qid{
					Path: QID(1, QWxdata),
					Vers: 0,
					Type: 0,
				},
				w: w,
			},
			fcall: plan9.Fcall{
				Count: 64,
			},
			fs: mr,
		})
		if got, want := mr.err, tc.err; got != want {
			t.Errorf("got error %v; want %v", got, want)
		}
		if tc.err == nil {
			if got, want := string(mr.fcall.Data), body; want != got {
				t.Errorf("got data %q; want %q", got, want)
			}
			if q0, q1 := w.addr.q0, w.addr.q1; q0 != q1 {
				t.Errorf("w.addr.q0=%v and w.addr.q1=%v", q0, q1)
			}
		}
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
