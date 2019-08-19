// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"mellium.im/communiqué/internal/client/event"
	"mellium.im/communiqué/internal/ui"
	"mellium.im/xmpp/jid"

	"github.com/fsnotify/fsnotify"
)

func getHistoryPath(configPath string, j jid.JID) (string, error) {
	historyDir := path.Join(configPath, "history")
	err := os.MkdirAll(historyDir, 0755)
	if err != nil {
		return "", err
	}
	return path.Join(historyDir, j.Bare().String()), nil
}

func writeMessage(configPath string, msg event.ChatMessage) error {
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

	_, err = fmt.Fprintf(f, "%s %s\n", time.Now().UTC().Format(time.RFC3339), msg.Body)
	return err
}

func loadBuffer(ctx context.Context, pane *ui.UI, configPath string, ev event.OpenChat) error {
	pane.History().SetText("")
	configPath, err := getHistoryPath(configPath, ev.JID)
	if err != nil {
		return err
	}

	// TODO: for some reason I did this with inotify, but I don't really care if
	// something external updates the history file. Instead, remove this
	// dependency and use the new message signal to update the buffer at the same
	// time it writes to the history file.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	/* #nosec */
	defer watcher.Close()
	if err = watcher.Add(configPath); err != nil {
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
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-watcher.Errors:
			return err
		case fevent := <-watcher.Events:
			switch fevent.Op {
			case fsnotify.Create:
				file, err = os.Open(configPath)
				if err != nil {
					return err
				}
				/* #nosec */
				defer file.Close()
				fallthrough
			case fsnotify.Write:
				_, err = io.Copy(pane.History(), file)
				if err != nil {
					return err
				}
			case fsnotify.Remove, fsnotify.Rename:
				pane.History().SetText("")
				file.Close()
			}
		}
	}
}
