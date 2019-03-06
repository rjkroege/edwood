package file

import (
	"bytes"
	"crypto/sha1"
	"io"
	"os"
)

type Hash [sha1.Size]byte

var EmptyHash Hash

func (h *Hash) Set(b []byte) {
	if len(b) != len(h) {
		panic("internal error: wrong hash size")
	}
	copy(h[:], b)
}

func (h Hash) Eq(h1 Hash) bool {
	return bytes.Equal(h[:], h1[:])
}

func CalcHash(b []byte) Hash {
	return sha1.Sum(b)
}

func HashFor(filename string) (h Hash, err error) {
	fd, err := os.Open(filename)
	if err != nil {
		return h, err
	}
	defer fd.Close()

	hh := sha1.New()
	if _, err := io.Copy(hh, fd); err != nil {
		return h, err
	}
	h.Set(hh.Sum(nil))
	return
}
