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
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"
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
	rosterHandler := newRosterHandler(c)

	return mux.New(
		mux.IQ(xmpp.HandlerFunc(func(t xmlstream.TokenReadWriter, start *xml.StartElement) error {
			tok, err := t.Token()
			if err != nil {
				return err
			}
			iq := stanza.IQ{}
			// TODO: Move this logic out somewhere else. Maybe into xmpp/stanza
			for _, attr := range start.Attr {
				switch attr.Name.Local {
				case "id":
					iq.ID = attr.Value
				case "to":
					j, err := jid.Parse(attr.Value)
					if err != nil {
						return err
					}
					iq.To = j
				case "from":
					j, err := jid.Parse(attr.Value)
					if err != nil {
						return err
					}
					iq.From = j
				case "lang":
					if attr.Name.Space == `http://www.w3.org/XML/1998/namespace` {
						iq.Lang = attr.Value
					}
				case "Type":
					// TODO: should we validate this?
					iq.Type = stanza.IQType(attr.Value)
				}
			}
			// If this isn't, the XML is broken and we've already detected it at a
			// higher level, right?
			// TODO: verify this assumption and maybe document this as an okay thing
			// to do in handlers.
			payload := tok.(xml.StartElement)
			switch start.Name.Space {
			case roster.NS:
				return rosterHandler(t, iq, &payload)
			}
			return nil
		})),
	)
	return nil
}
