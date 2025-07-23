package gozen

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"9fans.net/go/acme"
)

// Editinwin creates or opens plumbstring as needed and returns the window
// object.
func Editinwin(plumbstring string, opts ...option) (*acme.Win, error) {
	chunks := strings.Split(plumbstring, ":")
	if len(chunks) > 2 {
		return nil, fmt.Errorf("plumbhelper bad plumb address string")
	}
	fn := chunks[0]
	addr := ""
	if len(chunks) > 1 {
		addr = chunks[1]
	}
	log.Println("plumbhelper", fn, addr)

	// Two choices: we already have the Window open.
	wins, err := acme.Windows()
	if err != nil {
		return nil, fmt.Errorf("plumbhelper acme.Windows list was not available")
	}

	win := (*acme.Win)(nil)
	for _, wi := range wins {
		log.Println("wi", wi.Name)
		if wi.Name == fn {
			win, err = acme.Open(wi.ID, nil)
			if err != nil {
				return nil, fmt.Errorf("plumbhelper acme.Open")
			}
			break
		}
	}

	wasnew := false
	if win == nil {
		log.Println("plumbhelper making a new window")
		wasnew = true
		var err error
		win, err = acme.New()
		if err != nil {
			return nil, fmt.Errorf("plumbhelper acme.New: %v", err)
		}

		if err := win.Ctl("nomark"); err != nil {
			return nil, fmt.Errorf("plumbhelper win.Ctl nomark: %v", err)
		}

		if err := win.Name(fn); err != nil {
			return nil, fmt.Errorf("plumbhelper win.Name: %v", err)
		}

		if err := win.Ctl("get"); err != nil {
			return nil, fmt.Errorf("plumbhelper win.Ctl get: %v", err)
		}

		if err = win.Ctl("mark"); err != nil {
			return nil, fmt.Errorf("plumbhelper %q: %v", "mark", err)
		}

		if err = win.Ctl("clean"); err != nil {
			return nil, fmt.Errorf("plumbhelper %q: %v", "clean", err)
		}
	}

	if err := win.Addr(string(addr)); err != nil {
		return nil, fmt.Errorf("plumbhelper win.Addr: %v", err)
	}
	if err := win.Ctl("dot=addr\nshow\n"); err != nil {
		return nil, fmt.Errorf("plumbhelper win.Addr: %v", err)
	}

	// This general structure permits (I think) adding an arbitrary number of
	// additional customization settings (e.g. setting the dump command or
	// the like.)
	allerrs := make([]error, 0)
	for _, opt := range opts {
		allerrs = append(allerrs, opt(win, wasnew))
	}

	return win, errors.Join(allerrs...)
}

// Editinacme directly opens plumbstring in Acme/Edwood because regular
// plumb can't handle the paths found in the Go package database.
// Note that paths in plumbstring need to be absolute.
func Editinacme(plumbstring string, opts ...option) error {
	w, err := Editinwin(plumbstring, opts...)
	w.CloseFiles()
	return err
}
