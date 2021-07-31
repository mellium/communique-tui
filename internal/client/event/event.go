// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package event contains events that may be emited by the client.
package event // import "mellium.im/communique/internal/client/event"

import (
	"mellium.im/xmpp/delay"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"
)

type (
	// StatusOnline is sent when the user should come online.
	StatusOnline struct{}

	// StatusOffline is sent when the user should go offline.
	StatusOffline struct{}

	// StatusAway is sent when the user should change their status to away.
	StatusAway struct{}

	// StatusBusy is sent when the user should change their status to busy.
	StatusBusy struct{}

	// FetchRoster is sent when a roster is fetched.
	FetchRoster struct {
		Ver   string
		Items <-chan UpdateRoster
	}

	// UpdateRoster is sent when a roster item should be updated (eg. after a
	// roster push).
	UpdateRoster struct {
		roster.Item
		Ver string
	}

	// ChatMessage is sent when messages of type "chat" or "normal" are received
	// or sent.
	ChatMessage struct {
		stanza.Message

		Body     string          `xml:"body,omitempty"`
		OriginID stanza.OriginID `xml:"urn:xmpp:sid:0 origin-id"`
		SID      []stanza.ID     `xml:"urn:xmpp:sid:0 stanza-id"`
		Delay    delay.Delay     `xml:"urn:xmpp:delay delay"`

		// Sent is true if this message is one that we sent from another device (for
		// example, a message forwarded to us by message carbons).
		Sent bool `xml:"-"`
		// Account is true if this message was sent by the server (empty from, or
		// from matching the bare JID of the authenticated account).
		Account bool `xml:"-"`
	}

	// Receipt is sent when a message receipt is received and represents the ID of
	// the message that should be marked as received.
	// It may be sent by itself, or in addition to a ChatMessage event (or others)
	// if the payload containing the receipt also contains other events.
	Receipt string

	// OpenChat is sent when a roster item is selected.
	OpenChat roster.Item

	// CloseChat is sent when the chat view is closed.
	CloseChat roster.Item
)
