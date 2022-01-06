package main

import (
	"log"

	"9fans.net/go/plan9/client"
	"9fans.net/go/plumb"
)

func plumbthread() {
	handlePlumb(&plumbFsys{})
	log.Printf("plumbthread died")
}

type plumbFsys struct{}

func (fsys *plumbFsys) Open(name string, mode uint8) (*client.Fid, error) {
	return plumb.Open(name, int(mode))
}
