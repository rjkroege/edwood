package main

import (
	"testing"
)


/*
func TestNewBlock(t *testing.T) {

	disk := NewDisk()
	defer disk.fd.Name().Remove()


	


}
*/


func TestNtosize(t *testing.T) {

	testvector := []struct {
		n uint
		sz uint
		ip uint
	}{
		{ MaxBlock, MaxBlock, 32 },
		{ 255, 256, 1 },
		{ 1, 256, 1 },
		{ 256, 256, 1 },
		{ 257, 512, 2 },
		{ 0, 0, 0 },
	}

	for _, tv := range testvector {
		sz, ip := ntosize(tv.n)

		if got, want := sz, tv.sz; got != want {
			t.Errorf("for %d, got sz %d, want sz %d", tv.n, sz, tv.sz)
		}
		if got, want := ip, tv.ip; got != want {
			t.Errorf("for %d, got ip %d, want ip %d", tv.n, ip, tv.ip)
		}
	}
}
