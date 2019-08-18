// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package event contains events that may be emited by the client.
package event

import (
	"mellium.im/xmpp/roster"
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
)
