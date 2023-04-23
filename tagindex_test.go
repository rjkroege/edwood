package main

import (
	"testing"
)

func TestParseTagHelper(t *testing.T) {
	tests := []struct {
		offered, expected string
	}{
		{"''", "''"},
		{"This is a busted tag Del Snarf | Look", "This"},
		{"'This is a newfangled tag' Del Snarf | Look", "'This is a newfangled tag'"},
	}

	for _, v := range tests {
		returned := parsetaghelper(v.offered)
		if returned != v.expected {
			t.Errorf("Tag [%s]: expected [%s], got [%s]", v.offered, v.expected, returned)
		}
	}
}

func TestQuoteFilename(t *testing.T) {
	for _, tc := range []struct {
		in, out string
	}{
		{"quote me", "'quote me'"},
		{"dontquoteme\\", "dontquoteme\\"},
		{" quote me", "' quote me'"},
		{"quote me ", "'quote me '"},
		{"'dontrequoteme'", "'dontrequoteme'"},
	} {
		if qt := QuoteFilename(tc.in); qt != tc.out {
			t.Errorf("QuoteFilename failed: Expected [%v] got [%v]", tc.out, qt)
		}
	}
}

func TestUnquoteFilename(t *testing.T) {
	for _, tc := range []struct {
		out, in string
	}{
		{"unquote me", "'unquote me'"},
		{"dontunquoteme\\", "dontunquoteme\\"},
	} {
		if qt := UnquoteFilename(tc.in); qt != tc.out {
			t.Errorf("UnquoteFilename failed: Expected [%v] got [%v]", tc.out, qt)
		}
	}
}
