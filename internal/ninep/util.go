// Package ninep contains helper routines for implementing a 9P2000 protocol server.
package ninep

import (
	"fmt"

	"9fans.net/go/plan9"
)

// ReadBuffer sets Count and Data in response ofcall based the
// request ifcall and full data src. The Data is set to a sub-slice of src
// and the Count is set to the length of the sub-slice.
// This function is similar to readbuf(3) in lib9p.
func ReadBuffer(ofcall, ifcall *plan9.Fcall, src []byte) {
	n := len(src)
	off := ifcall.Offset
	cnt := ifcall.Count

	if len(src) == 0 || off >= uint64(n) {
		ofcall.Count = 0
		ofcall.Data = nil
		return
	}
	if off+uint64(cnt) > uint64(n) {
		cnt = uint32(uint64(n) - off)
	}
	ofcall.Count = cnt
	ofcall.Data = src[off : off+uint64(cnt)]
}

// ReadString is the same as ReadBuffer but for a string src.
// It trims src info blocks without respecting utf8 boundaries,
// as it's generally the reader's responsibility to do more reads to get full runes.
// This function is similar to readstr(3) in lib9p.
func ReadString(ofcall, ifcall *plan9.Fcall, src string) {
	ReadBuffer(ofcall, ifcall, []byte(src))
}

// DirRead sets ofcall.Data to at most ifcall.Count bytes of directory
// entries read from offset ifcall.Offset. The function gen is called
// to obtain the n-th directory entry. Gen should return nil on end
// of directory. DirRead returns the number of directory entries read.
// This function is similar to dirread9p(3) in lib9p.
func DirRead(ofcall, ifcall *plan9.Fcall, gen func(i int) *plan9.Dir) int {
	o := ifcall.Offset
	e := ifcall.Offset + uint64(ifcall.Count)
	data := make([]byte, ifcall.Count)
	n := 0
	i := uint64(0)
	dirindex := 0
	for i < e {
		d := gen(dirindex)
		if d == nil {
			break
		}
		b, _ := d.Bytes()
		length := len(b)
		if length > len(data[n:]) {
			break
		}
		if i >= o {
			copy(data[n:], b)
			n += length
		}
		dirindex++
		i += uint64(length)
	}
	ofcall.Data = data[:n]
	return dirindex
}

func gbit16(b []byte) (uint16, []byte) {
	return uint16(b[0]) | uint16(b[1])<<8, b[2:]
}

// UnmarshalDirs decodes and returns one or more directory entries in b.
// This function exists because plan9.UnmarshalDir cannot deal with
// multiple entries.
func UnmarshalDirs(b []byte) ([]plan9.Dir, error) {
	var result []plan9.Dir
	for {
		if len(b) <= 2 {
			break
		}
		n, _ := gbit16(b)
		d, err := plan9.UnmarshalDir(b[:2+n])
		if err != nil {
			return nil, err
		}
		b = b[2+n:]

		result = append(result, *d)
	}
	if len(b) != 0 {
		return nil, fmt.Errorf("partial directory entry")
	}
	return result, nil
}
