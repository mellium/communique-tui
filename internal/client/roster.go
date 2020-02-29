// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package client

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"

	"mellium.im/communiqué/internal/client/event"
)

func rosterPushHandler(c *Client) mux.IQHandlerFunc {
	return func(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
		item := roster.Item{}
		err := xml.NewTokenDecoder(t).Decode(&item)
		if err != nil {
			return err
		}

		c.handler(event.UpdateRoster(item))
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
}
