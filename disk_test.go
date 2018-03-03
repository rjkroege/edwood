package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestNewBlock(t *testing.T) {
	disk := NewDisk()
	defer os.Remove(disk.fd.Name())

	if len(disk.free) != 33 {
		t.Errorf("disk.free isn't big enough or I don't understand the code anymore.")
	}

	b := disk.NewBlock(255)
	if got, want := b.addr, uint(0); got != want {
		t.Errorf("got b.addr %d, want %d", got, want)
	}
	if got, want := b.n, uint(255); got != want {
		t.Errorf("got b.n %d, want %d", got, want)
	}
	if got, want := disk.addr, uint(256); got != want {
		t.Errorf("got disk.addr %d, want %d", got, want)
	}

	if b == disk.blist {
		t.Errorf("b should not be at the head of the pre-allocated list.")
	}
	if b.next != disk.blist {
		// This property exists becuase we don't have union types.
		// TODO(rjk): Fragile under a more idiomatic implementation.
		t.Errorf("b.next should point at the pre-allocated list.")
	}

	disk.Release(b)

	if b != disk.free[1] {
		t.Errorf("b didn't get put in the first element in appropriate free list")
	}

	b2 := disk.NewBlock(251)
	if b2 != b {
		t.Errorf("we failed to recycle b to b2")
	}
	if got, want := b2.n, uint(251); got != want {
		t.Errorf("got b2.n %d, want %d", got, want)
	}
}

// writereadtestcore provides the core write a rune array, read it back and compare for
// equality.
func writereadtestcore(t *testing.T, testname, inputstring string, oblock *Block, disk *Disk) *Block {
	inputrunes := bytes.Runes([]byte(inputstring))
	inputlen := len(inputrunes)

	nblock := oblock
	disk.Write(&nblock, inputrunes, uint(inputlen))

	// In this case, we are not changing the length.
	outputrunes := make([]rune, inputlen)
	disk.Read(nblock, outputrunes, uint(inputlen))

	var b strings.Builder
	for _, r := range outputrunes {
		b.WriteRune(r)
	}

	if got, want := b.String(), inputstring; got != want {
		t.Errorf("%s got %s, want %s", testname, got, want)
	}
	return nblock
}

func TestReadWriteSmall(t *testing.T) {
	disk := NewDisk()

	oblock := disk.NewBlock(uint(4))
	nblock := writereadtestcore(t, "small write-read test", "a日本b", oblock, disk)

	if oblock != nblock {
		t.Errorf("without resizing, nblock should equal oblock")
	}
}

func TestReadWriteBig(t *testing.T) {
	disk := NewDisk()

	// Roundtrip a bigger unicode string
	// Make a larger string.
	var b strings.Builder
	for i := 0; i < 100; i++ {
		b.WriteString("a日本b")
	}
	bigstring := b.String()

	original_large_blk := disk.NewBlock(uint(4 * 100))
	new_large_blk := writereadtestcore(t, "big write-read test", bigstring, original_large_blk, disk)

	if original_large_blk != new_large_blk {
		t.Errorf("without resizing, new_large_blk should equal original_large_blk")
	}

	// Resize it with a little string.
	new_small_blk := writereadtestcore(t, "small size-changing write-read test", "c日本d", new_large_blk, disk)
	if new_small_blk == original_large_blk {
		t.Errorf("with resizing, new_small_blk should not equal original_large_blk")
	}

	if original_large_blk != disk.free[2] {
		t.Errorf("with resizing, original_large_blk should be in the free-bucket for re-use")
	}
	if original_large_blk.next != nil {
		t.Errorf("Release failed to make original_large_blk.next point to nothing")
	}

	// Resize with a big string, make sure that we reuse the block
	b.Reset()
	for i := 0; i < 98; i++ {
		b.WriteString("eö本f")
	}
	bigstring = b.String()

	different_large_blk := writereadtestcore(t, "small to large size-changing write-read test", bigstring, new_small_blk, disk)
	if new_small_blk == different_large_blk {
		t.Errorf("with resizing, from small to large, different_large_blk should not equal new_small_blk")
	}
	if different_large_blk != original_large_blk {
		t.Errorf("with resizing to a previously-used large block, should have reused original_large_blk")
	}
	if disk.free[2] != nil {
		t.Errorf("reusing block from bucket should have removed the block")
	}
}

func TestNtosize(t *testing.T) {
	testvector := []struct {
		n  uint
		sz uint
		ip uint
	}{
		{MaxBlock, MaxBlock, 32},
		{255, 256, 1},
		{1, 256, 1},
		{256, 256, 1},
		{257, 512, 2},
		{0, 0, 0},
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
