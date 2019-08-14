// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/xml"
	"log"

	"mellium.im/communiqué/internal/ui"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/roster"
)

func newUIHandler(c *client, logger, debug *log.Logger) func(ui.Event) {
	return func(e ui.Event) {
		switch e {
		case ui.GoAway:
			go c.Away(context.TODO())
		case ui.GoOnline:
			go c.Online(context.TODO())
		case ui.GoBusy:
			go c.Busy(context.TODO())
		case ui.GoOffline:
			go c.Offline()
		default:
			debug.Printf("Unrecognized event: %q", e)
		}
	}
}

func newXMPPHandler(c *client) xmpp.Handler {
	iqHandler := newIQHandler(c)

	return mux.New(
		mux.IQ(iqHandler),
	)
}

func newIQHandler(c *client) xmpp.HandlerFunc {
	return func(t xmlstream.TokenReadEncoder, iq *xml.StartElement) error {
		tok, err := t.Token()
		if err != nil {
			return err
		}
		payload := tok.(xml.StartElement)
		switch payload.Name.Space {
		case roster.NS:
			return rosterPushHandler(t, c, iq, &payload)
		}
		return nil
	}
}
