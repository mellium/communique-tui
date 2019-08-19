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
	"mellium.im/xmpp/stanza"
)

func newXMPPHandler(c *Client) xmpp.Handler {
	iqHandler := newIQHandler(c)
	presenceHandler := newPresenceHandler(c)
	messageHandler := newMessageHandler(c)

	return mux.New(
		mux.IQ(iqHandler),
		mux.Presence(presenceHandler),
		mux.Message(messageHandler),
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

type deferEOF struct {
	r xml.TokenReader
}

func (r *deferEOF) Token() (xml.Token, error) {
	if r == nil {
		return nil, io.EOF
	}

	tok, err := r.r.Token()
	if err != nil {
		if err == io.EOF && tok != nil {
			r.r = nil
			return tok, nil
		}
	}
	return tok, err
}

func newMessageHandler(c *Client) xmpp.HandlerFunc {
	return func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
		msg := event.ChatMessage{}

		// TODO: Remove this workaround when https://golang.org/cl/130556 is
		// released.
		d := xml.NewTokenDecoder(&deferEOF{r: xmlstream.MultiReader(
			xmlstream.Token(start.Copy()),
			xmlstream.Inner(t),
			xmlstream.Token(start.End()),
		)})
		err := d.Decode(&msg)
		if err != nil {
			return err
		}
		switch msg.Type {
		case stanza.NormalMessage, stanza.ChatMessage:
			c.handler(msg)
		}
		return nil
	}
}
