package ninep

import (
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
