// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package client

import (
	"encoding/xml"
	"io"

	"mellium.im/communique/internal/client/event"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/carbons"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/history"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/receipts"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"
)

func newXMPPHandler(c *Client) xmpp.Handler {
	msgHandler := newMessageHandler(c)
	return mux.New(
		disco.Handle(),
		roster.Handle(roster.Handler{
			Push: func(ver string, item roster.Item) error {
				c.rosterVer = ver
				c.handler(event.UpdateRoster{Ver: ver, Item: item})
				return nil
			},
		}),
		carbons.Handle(carbons.Handler{
			F: func(_ stanza.Message, sent bool, inner xml.TokenReader) error {
				d := xml.NewTokenDecoder(inner)
				e := event.ChatMessage{Sent: sent}
				err := d.Decode(&e)
				if err != nil {
					return err
				}
				c.handler(e)
				return nil
			},
		}),
		mux.Presence("", xml.Name{}, newPresenceHandler(c)),
		mux.Message(stanza.NormalMessage, xml.Name{Local: "body"}, msgHandler),
		mux.Message(stanza.ChatMessage, xml.Name{Local: "body"}, msgHandler),
		receipts.Handle(c.receiptsHandler),
		history.Handle(history.NewHandler(newHistoryHandler(c))),
	)
}

func newPresenceHandler(c *Client) mux.PresenceHandlerFunc {
	return func(p stanza.Presence, t xmlstream.TokenReadEncoder) error {
		if !p.From.Equal(c.LocalAddr()) {
			return nil
		}

		// Throw away the start presence token.
		_, err := t.Token()
		if err != nil {
			return err
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

func newMessageHandler(c *Client) mux.MessageHandlerFunc {
	return func(_ stanza.Message, r xmlstream.TokenReadEncoder) error {
		msg := event.ChatMessage{}

		d := xml.NewTokenDecoder(r)
		err := d.Decode(&msg)
		if err != nil {
			return err
		}
		fromBare := msg.From.Bare()
		if fromBare.Equal(jid.JID{}) || fromBare.Equal(c.addr.Bare()) {
			msg.Account = true
		}
		c.handler(msg)
		return nil
	}
}

func newHistoryHandler(c *Client) mux.MessageHandlerFunc {
	return func(m stanza.Message, r xmlstream.TokenReadEncoder) error {
		msg := event.HistoryMessage{Message: m}

		d := xml.NewTokenDecoder(r)
		err := d.Decode(&msg.Result)
		if err != nil {
			return err
		}
		if !msg.From.Equal(jid.JID{}) && !msg.From.Equal(c.addr.Bare()) {
			c.debug.Printf("possibly spoofed history message from %s", msg.From)
			return nil
		}
		fromBare := msg.Result.Forward.Msg.From.Bare()
		if fromBare.Equal(jid.JID{}) || fromBare.Equal(c.addr.Bare()) {
			msg.Result.Forward.Msg.Account = true
		}
		msg.Result.Forward.Msg.Sent = fromBare.Equal(c.addr.Bare())
		msg.Result.Forward.Msg.Delay = msg.Result.Forward.Delay
		c.handler(msg)
		return nil
	}
}
