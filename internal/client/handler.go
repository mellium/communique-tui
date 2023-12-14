// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package client

import (
	//"context"
	"context"
	"encoding/xml"
	"io"
	"strconv"
	"time"

	//"time"

	"mellium.im/communique/internal/client/event"

	//"mellium.im/communique/internal/client/omemo"
	omemoreceiver "mellium.im/communique/internal/client/omemo/receiver"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/carbons"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/history"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/receipts"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"
)

func newXMPPHandler(c *Client) xmpp.Handler {
	msgHandler := newMessageHandler(c)
	omemoHandler := newOMEMOMessageHandler(c)
	return mux.New(
		c.In().XMLNS,
		disco.Handle(),
		disco.HandleCaps(func(p stanza.Presence, caps disco.Caps) {
			c.handler(event.NewCaps{
				From: p.From,
				Caps: caps,
			})
		}),
		muc.HandleClient(c.mucClient),
		// TODO: direct muc invitations.
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
		mux.Message(stanza.ChatMessage, xml.Name{Local: "encrypted"}, omemoHandler),
		mux.Message(stanza.GroupChatMessage, xml.Name{Local: "body"}, msgHandler),
		receipts.Handle(c.receiptsHandler),
		history.Handle(history.NewHandler(newHistoryHandler(c))),
	)
}

func newPresenceHandler(c *Client) mux.PresenceHandlerFunc {
	return func(p stanza.Presence, t xmlstream.TokenReadEncoder) error {
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
			c.handler(event.StatusAway(p.From))
		case "chat", "":
			c.handler(event.StatusOnline(p.From))
		case "dnd":
			c.handler(event.StatusBusy(p.From))
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

func newOMEMOMessageHandler(c *Client) mux.MessageHandlerFunc {
	return func(sm stanza.Message, r xmlstream.TokenReadEncoder) error {
		var currentEl, currentJid, currentRid string
		var fromJid jid.JID
		var keyElement, payload string
		keyExchange := false

		for t, _ := r.Token(); t != nil; t, _ = r.Token() {
			switch se := t.(type) {
			case xml.StartElement:
				currentEl = se.Name.Local

				switch se.Name.Local {
				case "message":
					for _, attr := range se.Attr {
						switch attr.Name.Local {
						case "from":
							fromJid = jid.MustParse(attr.Value)
						}
					}
				case "keys":
					currentJid = se.Attr[0].Value
				case "key":
					for _, attr := range se.Attr {
						switch attr.Name.Local {
						case "kex":
							keyExchange, _ = strconv.ParseBool(attr.Value)
						case "rid":
							currentRid = attr.Value
						}
					}
				}
			case xml.CharData:
				content := string(se[:])
				switch currentEl {
				case "key":
					if currentJid == c.addr.Bare().String() && currentRid == c.DeviceId {
						keyElement = content
					}
				case "payload":
					payload = content
				}
			}
		}

		var msgBody, opkId string
		var err error

		if keyExchange {
			msgBody, opkId = omemoreceiver.ReceiveKeyAgreement(keyElement, payload, fromJid.Bare().String(), c.DeviceId, c.IdPrivKey, c.SpkPriv, c.TmpDhPrivKey, c.TmpDhPubKey, c.OpkList, c.MessageSession, c.logger)
			c.RenewKeys(opkId)
		} else {
			msgBody, err = omemoreceiver.ReceiveEncryptedMessage(payload, fromJid.Bare().String(), c.DeviceId, c.MessageSession, c.logger)
			if err != nil {
				return nil
			}
		}

		c.logger.Print(opkId)

		msg := event.ChatMessage{}
		msg.Message = sm
		msg.Body = msgBody

		c.handler(msg)

		if keyExchange {
			tokenReader := omemoreceiver.PublishKeyBundle(c.DeviceId, c.LocalAddr().Bare().String(), c.IdPubKey, c.SpkPub, c.SpkSig, c.TmpDhPubKey, c.OpkList, c.logger)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			err := c.UnmarshalIQ(ctx, tokenReader, nil)

			if err != nil {
				c.logger.Printf("Error sending key bundle: %q", err)
			}
		}

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
