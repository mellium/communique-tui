// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"

	"mellium.im/communiqué/internal/ui"
)

func newRosterHandler(c *client) func(xmlstream.TokenReadWriter, stanza.IQ, *xml.StartElement) error {
	return func(t xmlstream.TokenReadWriter, iq stanza.IQ, payload *xml.StartElement) error {
		// TODO: will the server always have an empty from, or will it ever be the
		// domain? If it is sometimes a domain JID, can we normalize it to always be
		// empty?
		if payload.Name.Local == "query" && payload.Name.Space == roster.NS && iq.From.String() == "" {
			item := roster.Item{}
			err := xml.NewTokenDecoder(t).Decode(&item)
			if err != nil {
				return err
			}
			c.pane.AddRoster(ui.RosterItem{Item: item})
			iq.Type = stanza.ResultIQ
			iq.From, iq.To = iq.To, iq.From
			_, err = xmlstream.Copy(t, roster.IQ{IQ: iq}.TokenReader())
			return err
		}
		return nil
	}
}
