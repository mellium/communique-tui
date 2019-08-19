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
)

func getHistoryPath(configPath string, j jid.JID) (string, error) {
	historyDir := path.Join(configPath, "history")
	err := os.MkdirAll(historyDir, 0755)
	if err != nil {
		return "", err
	}
	return path.Join(historyDir, j.Bare().String()), nil
}

func writeMessage(pane *ui.UI, configPath string, msg event.ChatMessage) error {
	historyPath, err := getHistoryPath(configPath, msg.From)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(historyPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	/* #nosec */
	defer f.Close()

	historyLine := fmt.Sprintf("%s %s\n", time.Now().UTC().Format(time.RFC3339), msg.Body)

	_, err = io.WriteString(f, historyLine)
	if err != nil {
		return err
	}
	if item, ok := pane.Roster().GetSelected(); ok && item.Item.JID.Equal(msg.From.Bare()) {
		_, err = io.WriteString(pane.History(), historyLine)
		return err
	}
	return nil
}

func loadBuffer(pane *ui.UI, configPath string, ev event.OpenChat) error {
	pane.History().SetText("")
	configPath, err := getHistoryPath(configPath, ev.JID)
	if err != nil {
		return err
	}

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
