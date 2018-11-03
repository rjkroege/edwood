// Package frame supports frames of editable text.
//
// This is a port of Plan9's libframe to Go. It supports displaying
// a frame of editable text in a single font on
// raster displays, such as would be found in sam(1) and 9term(1). Frames may hold any
// character except NUL (0). Long lines are folded and tabs are at fixed
// intervals.
package frame
