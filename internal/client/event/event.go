// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package event contains events that may be emited by the client.
package event // import "mellium.im/communique/internal/client/event"

import (
	"mellium.im/communique/internal/client/jingle"
	"mellium.im/xmpp/bookmarks"
	"mellium.im/xmpp/commands"
	"mellium.im/xmpp/delay"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/forward"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"
)

type (
	// Connected is sent immediately after the connection is esablished.
	Connected struct{}

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

	// FetchRoster is sent when a roster is fetched.
	FetchRoster struct {
		Ver   string
		Items <-chan UpdateRoster
	}

	// FetchBookmarks is sent when the full list of bookmarks is fetched.
	FetchBookmarks struct {
		Items <-chan UpdateBookmark
	}

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

	// HistoryMessage is sent on incoming messages resulting from a history query.
	HistoryMessage struct {
		stanza.Message
		Result struct {
			QueryID string `xml:"queryid,attr"`
			ID      string `xml:"id,attr"`
			Forward struct {
				forward.Forwarded
				Msg ChatMessage `xml:"jabber:client message"`
			} `xml:"urn:xmpp:forward:0 forwarded"`
		} `xml:"urn:xmpp:mam:2 result"`
	}

	// Receipt is sent when a message receipt is received and represents the ID of
	// the message that should be marked as received.
	// It may be sent by itself, or in addition to a ChatMessage event (or others)
	// if the payload containing the receipt also contains other events.
	Receipt string

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

	// NewCaps is sent when new capabilities have been discovered.
	NewCaps struct {
		From jid.JID
		Caps disco.Caps
	}

	// NewFeatures is sent when entity features should be refreshed.
	NewFeatures struct {
		To   jid.JID
		Info chan<- struct {
			Info disco.Info
			Err  error
		}
	}

	// NewOutgoingCall is sent when there's a new request to start a call from current client
	NewOutgoingCall jid.JID

	// OutgoingCallAccepted is sent to gui when responder accept this client call initation request
	OutgoingCallAccepted *jingle.Jingle

	// NewIncomingCall is sent to gui to show calling window when a new call request arrive from client
	NewIncomingCall *jingle.Jingle

	// AcceptIncomingCall is sent from gui to indicate acceptance oh call initiation request
	AcceptIncomingCall *jingle.Jingle

	// CancelCall is sent when current client or another client cancel a call request
	CancelCall string

	// TerminateCall is sent when current client or another client want to terminate call session
	TerminateCall string
)
