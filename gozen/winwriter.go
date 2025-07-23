package gozen

import (
	"io"

	"9fans.net/go/acme"
)

type WindowWriter struct {
	dest string
	win  *acme.Win
}

func NewWindowWriter(dest string, win *acme.Win) *WindowWriter {
	return &WindowWriter{
		dest: dest,
		win:  win,
	}
}

func (w *WindowWriter) Write(p []byte) (n int, err error) {
	return w.win.Write(w.dest, p)
}

// Prove that I've written the correct thing.
var _ io.Writer = (*WindowWriter)(nil)
