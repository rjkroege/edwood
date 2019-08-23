package ninep

import (
	"fmt"
	"testing"

	"9fans.net/go/plan9"
	"github.com/google/go-cmp/cmp"
)

func TestReadString(t *testing.T) {
	tt := []struct {
		ofcall, ifcall plan9.Fcall
		src            string
	}{
		{
			plan9.Fcall{Data: nil, Count: 0},
			plan9.Fcall{Offset: 0, Count: 10},
			"",
		},
		{
			plan9.Fcall{Data: nil, Count: 0},
			plan9.Fcall{Offset: 100, Count: 10},
			"abcd",
		},
		{
			plan9.Fcall{Data: []byte("abcd"), Count: 4},
			plan9.Fcall{Offset: 0, Count: 10},
			"abcd",
		},
		{
			plan9.Fcall{Data: []byte("abcd"), Count: 4},
			plan9.Fcall{Offset: 3, Count: 4},
			"xxxabcdzzz",
		},
	}
	for _, tc := range tt {
		var got plan9.Fcall
		ReadString(&got, &tc.ifcall, tc.src)
		want := &tc.ofcall
		if diff := cmp.Diff(want, &got); diff != "" {
			t.Errorf("mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestDirRead(t *testing.T) {
	want := []plan9.Dir{
		{Name: "one"},
		{Name: "two"},
	}

	for _, tc := range []struct {
		name  string
		count uint32
		ndir  int
	}{
		{"TwoEntries", 512, 2},
		{"OneEntry", 80, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ifcall := plan9.Fcall{
				Count: tc.count,
			}
			var ofcall plan9.Fcall

			DirRead(&ofcall, &ifcall, func(i int) *plan9.Dir {
				if i < len(want) {
					return &want[i]
				}
				return nil
			})
			got, err := UnmarshalDirs(ofcall.Data)
			if err != nil {
				t.Fatalf("failed to unmarshal directory entries: %v", err)
			}
			if diff := cmp.Diff(want[:tc.ndir], got); diff != "" {
				t.Errorf("directory entries mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUnmarshalDirs(t *testing.T) {
	dir := plan9.Dir{Name: "hello.txt"}
	b, _ := dir.Bytes()
	for _, tc := range []struct {
		name string
		b    []byte
		dirs []plan9.Dir
		err  error
	}{
		{"Success", b, []plan9.Dir{dir}, nil},
		{"PartialDir", []byte{0, 0}, []plan9.Dir{dir}, fmt.Errorf("partial directory entry")},
		{"MalformedDir", []byte{2, 0, 0, 0}, []plan9.Dir{dir}, fmt.Errorf("malformed Dir")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := UnmarshalDirs(tc.b)
			if tc.err != nil {
				if err != tc.err && err.Error() != tc.err.Error() {
					t.Fatalf("got error %v; want %v", err, tc.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("got error %v", err)
			}
			if diff := cmp.Diff(tc.dirs, got); diff != "" {
				t.Errorf("directory entries mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
