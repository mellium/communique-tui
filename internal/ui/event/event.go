// Copyright 2024 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package event contains events that may be emitted by the UI.
package event // import "mellium.im/communique/internal/ui/event"

import (
	"mellium.im/xmpp/bookmarks"
	"mellium.im/xmpp/commands"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"
)

type (
	// StatusOnline is sent when the user should come online.
	StatusOnline jid.JID

	// StatusOffline is sent when the user should go offline.
	StatusOffline jid.JID

	// StatusAway is sent when the user should change their status to away.
	StatusAway jid.JID

	// StatusBusy is sent when the user should change their status to busy.
	StatusBusy jid.JID

	// LoadingCommands is sent by the UI when the ad-hoc command window opens.
	LoadingCommands jid.JID

	// ExecCommand is sent by the UI when an ad-hoc command should be executed.
	ExecCommand commands.Command

	// DeleteRosterItem is sent when a roster item has been removed (eg. after
	// UpdateRoster triggers a removal or it is removed in the UI).
	DeleteRosterItem roster.Item

	// UpdateRoster is sent when a roster item should be updated (eg. after a
	// roster push).
	UpdateRoster struct {
		roster.Item
		Ver string
	}

	// UpdateBookmark is sent when a bookmark should be updated (eg. if you have
	// subscribed to bookmark updates and received a push).
	UpdateBookmark bookmarks.Channel

	// DeleteBookmark is sent when a bookmark has been removed.
	DeleteBookmark bookmarks.Channel

	// ChatMessage is sent when messages of type "chat" or "normal" are received
	// or sent.
	ChatMessage struct {
		stanza.Message

		Body string `xml:"body,omitempty"`
	}

	// OpenChat is sent when a roster item is selected.
	OpenChat roster.Item

	// OpenChannel is sent when a bookmark is selected.
	OpenChannel bookmarks.Channel

	// CloseChat is sent when the chat view is closed.
	CloseChat roster.Item

	// Subscribe is sent when we subscribe to a users presence.
	Subscribe jid.JID

	// PullToRefreshChat is sent when we scroll up while already at the top of
	// the history or when we simply scroll to the top of the history.
	PullToRefreshChat roster.Item

	// UploadFile is sent to instruct the client to perform HTTP upload.
	UploadFile struct {
		Path    string
		Message ChatMessage
	}
)
