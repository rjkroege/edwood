package main

import (
	"testing"
)

func TestPost9P(t *testing.T) {
	if getuser() == "" {
		t.Errorf("Didn't get a username")
	}
	ns := getns()
	if ns == "" {
		t.Errorf("Namespace failed: %v", ns)
	}
}
