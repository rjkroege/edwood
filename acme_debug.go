//go:build debug
// +build debug

package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
)

var debugAddr = flag.String("debug", "", "Serve debug information on the supplied address")

func startProfiler() {
	if *debugAddr != "" {
		go func() {
			log.Println(http.ListenAndServe(*debugAddr, nil))
		}()
	}
}
