package main

import (
	"bytes"
	"io/ioutil"
	"os"
//	"os/user"
)


const (
	MaxBlock  = 8 * 1024	
	Blockincr = 256
)


type Block struct {
	addr uint   // disk address in bytes

	// NB: in the C version, these are in a union together. Only one is
	// used at a time.
	n    uint   // number of used runes in block
	next *Block // pointer to next in free list
}

type Disk struct {
	fd   *os.File
	addr uint
	free [MaxBlock/Blockincr + 1]*Block
}

func NewDisk() *Disk {
	// tmp, err := ioutil.TempFile("/tmp", fmt.Sprintf("X%d.%.4sacme", os.Getpid(), u.Username))
	tmp, err := ioutil.TempFile("", "acme")
	if err != nil {
		panic(err)
	}
	return &Disk {
		fd: tmp,
	}
}

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
		if blist == nil {
			bl := new(Block)
			blist = bl
			for j := 0; j < 100-1; j++ {
				bl.next = new(Block)
				bl = bl.next
			}
		}
		b = blist
		blist = b.next
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

func (d *Disk) Read(b *Block, r []rune, n uint) {
	if n > b.n {
		panic("internal error: disk.Read")
	}
	// this is a simplified way of checking that b.n < MaxBlock
	_, _ = ntosize(b.n)
	buf := make([]byte, n)
	if m, err := d.fd.ReadAt(buf, int64(b.addr)); err != nil {
		panic(err)
	} else if m != len(r) {
		panic("read error from temp file")
	}
	copy(r, bytes.Runes(buf))
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
	if m, err := d.fd.WriteAt([]byte(string(r)), int64(bl.addr)); err != nil {
		panic(err)
	} else if m != len(r) {
		panic("write error to temp file")
	}
	bl.n = n
}
