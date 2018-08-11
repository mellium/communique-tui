package main

import (
	"log"

	"mellium.im/communiqué/internal/ui"
)

func newUIHandler(c *client, logger, debug *log.Logger) func(ui.Event) {
	return func(e ui.Event) {
		switch e {
		case ui.GoAway:
			c.Away()
		case ui.GoOnline:
			c.Online()
		case ui.GoBusy:
			c.Busy()
		// TODO:
		// case ui.GoOffline:
		//c.Offline()
		default:
			debug.Printf("Unrecognized event: %q", e)
		}
	}
}
