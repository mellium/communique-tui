// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package client

import (
	"encoding/xml"
	"io"

	"mellium.im/communiqué/internal/client/event"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/roster"
)

func newXMPPHandler(c *Client) xmpp.Handler {
	iqHandler := newIQHandler(c)
	presenceHandler := newPresenceHandler(c)

	return mux.New(
		mux.IQ(iqHandler),
		mux.Presence(presenceHandler),
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

func getAttr(attr []xml.Attr, local string) string {
	for _, a := range attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

func newPresenceHandler(c *Client) xmpp.HandlerFunc {
	return func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
		from, err := jid.Parse(getAttr(start.Attr, "from"))
		if err != nil {
			return err
		}
		if !from.Equal(c.LocalAddr()) {
			return nil
		}

		var status string
		for {
			tok, err := t.Token()
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			start, ok := tok.(xml.StartElement)
			switch {
			case !ok:
				continue
			case start.Name.Local != "show":
				err = xmlstream.Skip(t)
				if err != nil {
					return err
				}
				continue
			}

			tok, err = t.Token()
			if err != nil {
				return err
			}
			chars, ok := tok.(xml.CharData)
			if !ok {
				// Treat an invalid encoding of the status as an unrecognized status.
				return nil
			}
			status = string(chars)
			break
		}

		// See https://tools.ietf.org/html/rfc6121#section-4.7.2.1
		switch status {
		case "away", "xa":
			c.handler(event.StatusAway{})
		case "chat", "":
			c.handler(event.StatusOnline{})
		case "dnd":
			c.handler(event.StatusBusy{})
		}
		return nil
	}
}
