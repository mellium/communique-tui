// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package event contains events that may be emited by the client.
package event // import "mellium.im/communique/internal/client/event"

import (
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

	// UpdateRoster is sent when a roster item should be updated (eg. after a
	// roster fetch or a roster push).
	UpdateRoster roster.Item

	// ChatMessage is sent when messages of type "chat" or "normal" are received
	// or sent.
	ChatMessage struct {
		stanza.Message

		Body string `xml:"body,omitempty"`

		// True if this message is one that we sent from another device (for
		// example, a message forwarded to us by message carbons).
		Sent bool `xml:"-"`
	}

	// OpenChat is sent when a roster item is selected.
	OpenChat roster.Item

	// CloseChat is sent when the chat view is closed.
	CloseChat roster.Item
)
