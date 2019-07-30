// Package ninep contains helper routines for implementing a 9P2000 protocol server.
package ninep

import "9fans.net/go/plan9"

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
