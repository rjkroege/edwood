package util

import (
	"reflect"
	"testing"
)

func TestCvttorunes(t *testing.T) {
	testCases := []struct {
		p     []byte
		n     int
		r     []rune
		nb    int
		nulls bool
	}{
		{[]byte("Hello world"), 11, []rune("Hello world"), 11, false},
		{[]byte("Hello \x00\x00world"), 13, []rune("Hello world"), 13, true},
		{[]byte("Hello 世界"), 6 + 3 + 3, []rune("Hello 世界"), 6 + 3 + 3, false},
		{[]byte("Hello 世界"), 6 + 3 + 1, []rune("Hello 世界"), 6 + 3 + 3, false},
		{[]byte("Hello 世界"), 6 + 3 + 2, []rune("Hello 世界"), 6 + 3 + 3, false},
		{[]byte("Hello 世\xe7\x95"), 6 + 3 + 1, []rune("Hello 世�"), 6 + 3 + 1, false},
		{[]byte("Hello 世\xe7\x95"), 6 + 3 + 2, []rune("Hello 世��"), 6 + 3 + 2, false},
		{[]byte("\xe4\xb8\x96\xe7\x95\x8c hello"), 3 + 3 + 6, []rune("世界 hello"), 3 + 3 + 6, false},
		{[]byte("\xb8\x96\xe7\x95\x8c hello"), 2 + 3 + 6, []rune("��界 hello"), 2 + 3 + 6, false},
		{[]byte("\x96\xe7\x95\x8c hello"), 1 + 3 + 6, []rune("�界 hello"), 1 + 3 + 6, false},
	}
	for _, tc := range testCases {
		r, nb, nulls := Cvttorunes(tc.p, tc.n)
		if !reflect.DeepEqual(r, tc.r) || nb != tc.nb || nulls != tc.nulls {
			t.Errorf("Cvttorunes of (%q, %v) returned %q, %v, %v; expected %q, %v, %v\n",
				tc.p, tc.n, r, nb, nulls, tc.r, tc.nb, tc.nulls)
		}
	}
}
