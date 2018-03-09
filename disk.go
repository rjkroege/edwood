package main

import (
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"unsafe"
)

// Acme stores blocks as (raw) runes. This does not align well with how Go
// works where the language (rightly) tries to make sure that we don't do
// non-portable C things.
//
// I have switched this to Acme-style raw rune handling to avoid multiple
// copies and rune to byte conversion. This seems better in keeping with how
// Acme actually should work. However, this is not perhaps the right way
// to do it. It might be better to always store files in UTF8 (ideally as strings).

const (
	MaxBlock  = 8 * 1024
	Blockincr = 256
)

type Block struct {
	addr uint // disk address in bytes

	// NB: in the C version, these are in a union together. Only one is
	// used at a time.
	n    uint   // number of used runes in block
	next *Block // pointer to next in free list
}

// Disk is a singleton managing the file that Acme blocks. Blocks
// are sized from 256B to 8K in 256B increments.
type Disk struct {
	fd    *os.File
	addr  uint
	free  [MaxBlock/Blockincr + 1]*Block // Disk-backed blocks bucketed by size
	blist *Block                         // Empty block objects
}

// NewDisk creates a new backing on-disk file for Acme's paging.
func NewDisk() *Disk {
	// tmp, err := ioutil.TempFile("/tmp", fmt.Sprintf("X%d.%.4sacme", os.Getpid(), u.Username))
	tmp, err := ioutil.TempFile("", "acme")
	if err != nil {
		panic(err)
	}
	return &Disk{
		fd: tmp,
	}
}

// ntosize computes the size of block to hold n bytes where n must be
// MaxBlock and the index into the bucket array. The 0-th bucket holds
// only 0-length blocks.
func ntosize(n uint) (uint, uint) {
	if n > MaxBlock {
		panic("internal error: ntosize")
	}
	size := n
	if size&(Blockincr-1) != 0 {
		size += Blockincr - (size & (Blockincr - 1))
	}

	// last bucket holds blocks of exactly Maxblock
	ip := size / Blockincr
	return size, ip
}

func (d *Disk) NewBlock(n uint) *Block {
	size, i := ntosize(n)
	b := d.free[i]
	if b != nil {
		d.free[i] = b.next
	} else {
		if d.blist == nil {
			// TODO(rjk): Is allocating 1/time really a perf bottleneck in go?
			bl := new(Block)
			d.blist = bl
			for j := 0; j < 100-1; j++ {
				bl.next = new(Block)
				bl = bl.next
			}
		}
		b = d.blist
		d.blist = b.next
		b.addr = d.addr
		d.addr += size
	}
	b.n = n
	return b
}

func (d *Disk) Release(b *Block) {
	_, i := ntosize(b.n)
	b.next = d.free[i]
	d.free[i] = b
}

// TODO(rjk): Remove the n? Return an error?
func (d *Disk) Read(b *Block, r []rune, n uint) {
	if n > b.n {
		panic("internal error: disk.Read")
	}
	// this is a simplified way of checking that b.n < MaxBlock
	_, _ = ntosize(b.n)

	// Test prior to defeating type safety.
	if n > uint(len(r)) {
		panic("internal error: disk Read, n greater than r")
	}

	uby, sz := makeAliasByteArray(r, n)
	if m, err := d.fd.ReadAt(uby, int64(b.addr)); err != nil {
		panic(err)
	} else if m != sz*int(n) {
		panic("read error from temp file, m !=  sz * n ")
	}
}

func makeAliasByteArray(r []rune, un uint) ([]byte, int) {
	n := int(un)
	sz := int(unsafe.Sizeof(' '))
	rhdr := (*reflect.SliceHeader)(unsafe.Pointer(&r))

	uby := make([]byte, 0)
	bhdr := (*reflect.SliceHeader)(unsafe.Pointer(&uby))
	bhdr.Data = rhdr.Data
	bhdr.Len = sz * n
	bhdr.Cap = sz * n

	return uby, sz
}

func (d *Disk) Write(bp **Block, r []rune, n uint) {
	bl := *bp
	size, _ := ntosize(bl.n)
	nsize, _ := ntosize(n)
	if size != nsize {
		d.Release(bl)
		bl = d.NewBlock(n)
		*bp = bl
	}

	// Test prior to defeating type safety.
	if n > uint(len(r)) {
		panic("internal error: disk Write, n greater than r")
	}

	uby, sz := makeAliasByteArray(r, n)
	if m, err := d.fd.WriteAt(uby, int64(bl.addr)); err != nil {
		panic(err)
	} else if m != int(n)*sz {
		log.Println("write mismatch", m, int(n)*sz)
		panic("write error to temp file")
	}
	bl.n = n
}
