package main

import (
	"flag"
	"io"
	"log"
	"path/filepath"

	"9fans.net/go/acme"
	"github.com/rjkroege/gozen"
)

var debug = flag.Bool("d", false, "set for verbose debugging")

func main() {
	flag.Parse()
	if !*debug {
		log.SetOutput(io.Discard)
	}
	log.Println("hi", flag.Args())

	// The argument to Editinacme needs to be an absolute path.
	ap, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		log.Fatalf("can't abs %q: %v", flag.Arg(0), err)
	}

	win, err := gozen.Editinwin(ap, gozen.Addtotag("hello, added with E"))
	if err != nil {
		log.Fatalf("Editinwin failed for %s: %v", ap, err)
	}
	// Close it here so that Edwood will deliver delete messages. Otherwise,
	// the open connection to Edwood retained inside gozen (p9/acme package
	// actually) will keep the window alive.
	win.CloseFiles()

	// Wait for when the file is closed.
	edwoodlog, err := acme.Log()
	if err != nil {
		log.Fatalf("acme.Log creation failed: %v", err)
	}

	for {
		log.Println("top of loop!")
		ev, err := edwoodlog.Read()
		if err != nil {
			log.Fatalf("log reading failed %v", err)
		}
		log.Println(ev)

		if ev.Name == ap && ev.Op == "del" {
			edwoodlog.Close()
			return
		}
	}

}
