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

func rosterPushHandler(t xmlstream.TokenReadWriter, c *client, iq, payload *xml.StartElement) error {
	if payload.Name.Local == "query" {
		item := roster.Item{}
		err := xml.NewTokenDecoder(t).Decode(&item)
		if err != nil {
			return err
		}

		c.pane.UpdateRoster(ui.RosterItem{Item: item})
		return nil

		//iqVal, err := stanza.NewIQ(iq)
		//if err != nil {
		//	return err
		//}
		//if iqVal.From.String() != "" {
		//	return stanza.Error{
		//		Type:      stanza.Cancel,
		//		Condition: stanza.Forbidden,
		//	}
		//}

		//iqVal = iqVal.Result()
		//_, err = xmlstream.Copy(t, roster.IQ{IQ: iqVal}.TokenReader())
		//return err
	}

	return stanza.Error{
		Type:      stanza.Cancel,
		Condition: stanza.FeatureNotImplemented,
	}
}
