//go:build !debug
// +build !debug

// We get a smaller binary by not importing net/http/pprof.

package main

func startProfiler() {}
