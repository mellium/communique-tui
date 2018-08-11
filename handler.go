package main

import (
	"log"

	"mellium.im/communiqué/internal/ui"
)

func newUIHandler(c *client, logger, debug *log.Logger) func(ui.Event) {
	return func(e ui.Event) {
		switch e {
		case ui.GoAway:
			go c.Away()
		case ui.GoOnline:
			go c.Online()
		case ui.GoBusy:
			go c.Busy()
		case ui.GoOffline:
			go c.Offline()
		default:
			debug.Printf("Unrecognized event: %q", e)
		}
	}
}
