// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"golang.org/x/text/transform"

	"mellium.im/communique/internal/client/event"
	"mellium.im/communique/internal/escape"
	"mellium.im/communique/internal/storage"
	"mellium.im/communique/internal/ui"
	"mellium.im/xmpp/roster"
)

func writeMessage(sent bool, pane *ui.UI, configPath string, msg event.ChatMessage) error {
	historyAddr := msg.From
	arrow := "←"
	if sent {
		historyAddr = msg.To
		arrow = "→"
	}

	historyLine := fmt.Sprintf("%s %s %s\n", time.Now().UTC().Format(time.RFC3339), arrow, msg.Body)

	history := pane.History()

	j := historyAddr.Bare()
	if pane.ChatsOpen() {
		if item, ok := pane.Roster().GetSelected(); ok && item.Item.JID.Equal(j) {
			// If the message JID is selected and the window is open, write it to the
			// history window.
			_, err := io.WriteString(history, historyLine)
			return err
		}
	}

	// If it's not selected (or the message window is not open), mark the item as
	// unread in the roster
	ok := pane.Roster().MarkUnread(j.String(), msg.ID)
	if !ok {
		// If the item did not exist, create it then try to mark it as unread
		// again.
		pane.UpdateRoster(ui.RosterItem{
			Item: roster.Item{
				JID: j,
				// TODO: get the preferred nickname.
				Name:         j.Localpart(),
				Subscription: "none",
			},
		})
		pane.Roster().MarkUnread(j.String(), msg.ID)
	}
	pane.Redraw()
	return nil
}

func loadBuffer(pane *ui.UI, db *storage.DB, configPath string, ev event.OpenChat, msgID string) error {
	history := pane.History()
	history.SetText("")

	iter := db.QueryHistory(context.TODO(), ev.JID.String(), "")
	for iter.Next() {
		sent, cur := iter.Message()
		if cur.ID != "" && cur.ID == msgID {
			_, err := io.WriteString(history, "─\n")
			if err != nil {
				return err
			}
		}
		err := writeMessage(sent, pane, configPath, cur)
		if err != nil {
			history.SetText(fmt.Sprintf("Error writing history: %v", err))
			return nil
		}
	}
	if err := iter.Err(); err != nil {
		history.SetText(fmt.Sprintf("Error querying for history: %v", err))
	}
	return nil
}

// unreadMarkReader wraps an io.Reader in a new reader that will insert an
// unread marker at the given offset.
func unreadMarkReader(r io.Reader, color tcell.Color, offset int64) io.Reader {
	t := escape.Transformer()

	return io.MultiReader(
		transform.NewReader(io.LimitReader(r, offset), t),
		// This marker is used by the text view UI to draw the unread marker
		strings.NewReader("─\n"),
		transform.NewReader(r, t),
	)
}
