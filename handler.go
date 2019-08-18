// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"log"

	"mellium.im/communiqué/internal/client"
	"mellium.im/communiqué/internal/client/event"
	"mellium.im/communiqué/internal/ui"
	"mellium.im/xmpp/roster"
)

func errLogger(logger *log.Logger) func(string, error) {
	return func(msg string, err error) {
		if err != nil {
			logger.Printf("%s: %q", msg, err)
		}
	}
}

// newUIHandler returns a handler for events that are emitted by the UI that
// need to modify the client state.
func newUIHandler(pane *ui.UI, c *client.Client, logger, debug *log.Logger) func(interface{}) {
	logErr := errLogger(logger)

	return func(ev interface{}) {
		switch e := ev.(type) {
		case event.StatusAway:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
				defer cancel()
				logErr("Error setting away status", c.Away(ctx))
			}()
		case event.StatusOnline:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
				defer cancel()
				logErr("Error setting online status", c.Online(ctx))
			}()
		case event.StatusBusy:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
				defer cancel()
				logErr("Error setting busy status", c.Busy(ctx))
			}()
		case event.StatusOffline:
			go logErr("Error going offline", c.Offline())
		case event.UpdateRoster:
			panic("event.UpdateRoster: not yet implemented")
		default:
			debug.Printf("Unrecognized ui event: %q", e)
		}
	}
}

// newClientHandler returns a handler for events that are emitted by the client
// that need to modify the UI.
func newClientHandler(pane *ui.UI, logger, debug *log.Logger) func(interface{}) {
	return func(ev interface{}) {
		switch e := ev.(type) {
		case event.StatusAway:
			pane.Away()
		case event.StatusBusy:
			pane.Busy()
		case event.StatusOnline:
			pane.Online()
		case event.StatusOffline:
			pane.Offline()
		case event.UpdateRoster:
			pane.UpdateRoster(ui.RosterItem{Item: roster.Item(e)})
		default:
			debug.Printf("Unrecognized client event: %q", e)
		}
	}
}
