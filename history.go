// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"mellium.im/communiqué/internal/client/event"
	"mellium.im/communiqué/internal/ui"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/roster"
)

func getHistoryPath(configPath string, j jid.JID) (string, error) {
	historyDir := path.Join(configPath, "history")
	/* #nosec */
	err := os.MkdirAll(historyDir, 0755)
	if err != nil {
		return "", err
	}
	return path.Join(historyDir, j.Bare().String()), nil
}

func writeMessage(sent bool, pane *ui.UI, configPath string, msg event.ChatMessage) error {
	historyAddr := msg.From
	arrow := "←"
	if sent {
		historyAddr = msg.To
		arrow = "→"
	}
	historyPath, err := getHistoryPath(configPath, historyAddr)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(historyPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	/* #nosec */
	defer f.Close()

	historyLine := fmt.Sprintf("%s %s %s\n", time.Now().UTC().Format(time.RFC3339), arrow, msg.Body)

	_, err = io.WriteString(f, historyLine)
	if err != nil {
		return err
	}

	j := historyAddr.Bare()
	if item, ok := pane.Roster().GetSelected(); ok && item.Item.JID.Equal(j) {
		// If the message JID is selected, write it to the history window.
		_, err = io.WriteString(pane.History(), historyLine)
		return err
	} else {
		// If it's not selected, mark the item as unread in the roster
		ok := pane.Roster().MarkUnread(j.String())
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
			pane.Roster().MarkUnread(j.String())
		}
		pane.Redraw()
	}
	return nil
}

func loadBuffer(pane *ui.UI, configPath string, ev event.OpenChat) error {
	pane.History().SetText("")
	configPath, err := getHistoryPath(configPath, ev.JID)
	if err != nil {
		return err
	}

	/* #nosec */
	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	/* #nosec */
	defer file.Close()
	_, err = io.Copy(pane.History(), file)
	if err != nil {
		pane.History().SetText(fmt.Sprintf("Error copying history: %v", err))
	}
	return nil
}
