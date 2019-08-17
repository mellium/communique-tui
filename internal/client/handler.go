// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package client

import (
	"encoding/xml"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/roster"
)

func newXMPPHandler(c *Client) xmpp.Handler {
	iqHandler := newIQHandler(c)

	return mux.New(
		mux.IQ(iqHandler),
	)
}

func newIQHandler(c *Client) xmpp.HandlerFunc {
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
