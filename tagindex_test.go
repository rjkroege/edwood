package main

import (
	"testing"
)

func TestParseTagHelper(t *testing.T) {
	tests := []struct {
		offered, expected string
	}{
		{"''", "''"},
		{"This is a busted tag Del Snarf | Look","This"},
		{"'This is a newfangled tag' Del Snarf | Look","'This is a newfangled tag'"},
	}

	for _, v := range tests {
		returned := parsetaghelper(v.offered)
		if returned != v.expected {
			t.Errorf("Tag [%s]: expected [%s], got [%s]", v.offered, v.expected, returned)
		}
	}
}