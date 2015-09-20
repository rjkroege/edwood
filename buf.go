package main

import (
	"bytes"
	"os"
	"unicode/utf8"
)

const (
	Slop = 100
)

type Buffer struct {
	nc     uint
	c      []rune
	cnc    uint
	cmax   uint
	cq     uint
	cdirty bool
	cbi    uint
	bl     []*Block
	nbl    uint
}

func (b *Buffer) sizecache(n uint) {
	if n <= b.cmax {
		return
	}
	b.cmax = n + Slop
	tmp := b.c
	b.c = make([]rune, b.cmax)
	copy(b.c, tmp)
}

func (b *Buffer) addblock(i, n uint) {
	if i > b.nbl {
		panic("internal error: addblock")
	}
	b.bl = append(b.bl[:i], append([]*Block{disk.NewBlock(n)}, b.bl[i:]...)...)
	b.nbl++
}

func (b *Buffer) deleteblock(i uint) {
	if i >= b.nbl {
		panic("internal error: deleteblock")
	}
	disk.Release(b.bl[i])
	b.nbl -= 1
	b.bl = append(b.bl[:i], b.bl[i+1:]...)
}

func (b *Buffer) flush() {
	if b.cdirty || b.cnc == 0 {
		if b.cnc == 0 {
			b.deleteblock(b.cbi)
		} else {
			disk.Write(&b.bl[b.cbi], b.c, b.cnc)
		}
		b.cdirty = false
	}
}

func (b *Buffer) setcache(q0 uint) {
	var i, q uint

	if q0 > b.nc {
		panic("internal error: setcache")
	}

	// flush and reload if q0 is not in cache
	if b.nc == 0 || (b.cq <= q0 && q0 < b.cq+b.cnc) {
		return
	}

	// if q0 is at end of file and end of cache, continue to grow this block
	if q0 == b.nc && q0 == b.cq+b.cnc && b.cnc < MaxBlock {
		return
	}
	b.flush()
	// find block
	if q0 < b.cq {
		q = 0
		i = 0
	} else {
		q = b.cq
		i = b.cbi
	}

	blp := &b.bl[i]
	for q+(*blp).n <= q0 && q+(*blp).n < b.nc {
		q += (*blp).n
		i++
		blp = &b.bl[i]
		if i >= b.nbl {
			panic("internal error: setcache block not found")
		}
	}

	bl := *blp
	b.cbi = i
	b.cq = q
	b.sizecache(bl.n)
	b.cnc = bl.n
	disk.Read(bl, b.c, b.cnc)
}

func (b *Buffer) Insert(q0, n uint, r []rune) {
	var i, m uint

	s := r[:]

	if q0 > b.nc {
		panic("internal error: bufinsert")
	}

	for n > 0 {
		b.setcache(q0)
		off := q0 - b.cq

		if b.cnc+n <= MaxBlock {
			t := b.cnc + n
			m := n

			if b.bl == nil {
				if b.cnc != 0 {
					panic("internal error: bufinsert1 cnc != 0")
				}
				b.addblock(0, t)
				b.cbi = 0
			}
			b.sizecache(t)

			// shift the beginning of the cache up and then
			// copy in new data
			copy(b.c[off+m:], b.c[off:b.cnc])
			copy(b.c[off:], s[:m])
			b.cnc = t
			goto Tail
		}

		/*
		 * We must make a new block.  If q0 is at
		 * the very beginning or end of this block,
		 * just make a new block and fill it.
		 */
		if q0 == b.cq || q0 == b.cq+b.cnc {
			if b.cdirty {
				b.flush()
			}
			m = uint(min(int(n), MaxBlock))
			if b.bl == nil {
				if b.cnc != 0 {
					panic("internal error: bufinsert2 cnc != 0")
				}
				i = 0
			} else {
				i = b.cbi
				if q0 > b.cq {
					i++
				}
			}

			b.addblock(i, uint(m))
			b.sizecache(uint(m))
			copy(b.c, s[:m])
			goto Tail
		}

		/*
		 * Split the block; cut off the right side and
		 * let go of it.
		 */
		m = b.cnc - off
		if m > 0 {
			i = b.cbi + 1
			b.addblock(i, m)
			disk.Write(&b.bl[i], b.c[off:], m)
			b.cnc -= m
		}

		/*
		 * Now at end of block.  Take as much input
		 * as possible and tack it on end of block.
		 */
		m = uint(min(int(n), int(MaxBlock-b.cnc)))
		b.sizecache(b.cnc + m)
		copy(b.c[b.cnc:], s[:m])
		b.cnc += m
	Tail:
		b.nc += m
		q0 += m
		s = s[m:]
		n -= m
		b.cdirty = true
	}
}

func (b *Buffer) Delete(q0, q1 uint) {
	if !(q0 < q1 && q0 <= b.nc && q1 <= b.nc) {
		panic("internal error: bufdelete")
	}
	var m, n uint
	for q1 > q0 {
		b.setcache(q0)
		off := q0 - b.cq
		if q1 > b.cq+b.cnc {
			n = b.cnc - off
		} else {
			n = q1 - q0
		}
		m = b.cnc - (off - n)
		if m > 0 {
			copy(b.c[off:], b.c[off+n:off+n+m])
		}
		b.cnc -= n
		b.cdirty = true
		q1 -= n
		b.nc -= n
	}
}

func (b *Buffer) Load(q0 uint, fd *os.File, nulls *int) int {
	if q0 > b.nc {
		panic("internal error: buffer.Load")
	}

	p := make([]byte, MaxBlock+utf8.UTFMax+1)

	m := 0
	n := 1
	q1 := q0
	var err error

	for n > 0 {
		n, err = fd.Read(p[m:])
		if err != nil {
			panic(err)
		}
		m += n
		l := m
		if n > 0 {
			l -= utf8.UTFMax
		}
		r := bytes.Runes(p[:l])
		nr := len(r)
		//nb := len([]byte(string(r)))
		copy(p, p[:m])
		b.Insert(q1, uint(nr), r)
		q1 += uint(nr)
	}
	return int(q1 - q0)
}

func (b *Buffer) Read(q0, n uint, r []rune) {
	if !(q0 <= b.nc && q0+n <= b.nc) {
		panic("internal error: Buffer.Read")
	}

	s := r[:]
	for n > 0 {
		b.setcache(q0)
		m := min(int(n), int(b.cnc-(q0-b.cq)))
		// copy m runes from cache at (q0 - b.cq) to s
		copy(s, b.c[int(q0-b.cq):int(q0-b.cq+uint(m))])
		q0 += uint(m)
		s = s[m:]
		n -= uint(m)
	}
}

func (b *Buffer) Close() {
	b.Reset()
	b.c = nil
	b.cnc = 0
	b.bl = nil
	b.nbl = 0
}

func (b *Buffer) Reset() {
	b.nc = 0
	b.cnc = 0
	b.cq = 0
	b.cdirty = false
	b.cbi = 0
	// delete backwards to avoid n^2 behaviour
	for i := b.nbl - 1; i >= 0; i-- {
		b.deleteblock(i)
	}
}
