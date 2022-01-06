//go:build !plan9
// +build !plan9

package main

import (
	"time"

	"9fans.net/go/plan9/client"
)

func plumbthread() {
	// Loop so that if plumber is restarted, acme need not be.
	for {
		var fsys *client.Fsys

		// Connect to plumber.
		for {
			// We can't use plumb.Open here because it caches the client.Fsys.
			var err error
			fsys, err = client.MountService("plumb")
			if err != nil {
				time.Sleep(2 * time.Second) // Try every 2 seconds
			} else {
				break
			}
		}

		handlePlumb(fsys)
	}
}
