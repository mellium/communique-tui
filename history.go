// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"golang.org/x/text/transform"

	"mellium.im/communique/internal/client/event"
	"mellium.im/communique/internal/escape"
	"mellium.im/communique/internal/ui"
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

	finfo, err := f.Stat()
	if err != nil {
		return err
	}
	unreadSize := finfo.Size()

	historyLine := fmt.Sprintf("%s %s %s\n", time.Now().UTC().Format(time.RFC3339), arrow, msg.Body)

	_, err = io.WriteString(f, historyLine)
	if err != nil {
		return err
	}

	history := pane.History()

	j := historyAddr.Bare()
	if pane.ChatsOpen() {
		if item, ok := pane.Roster().GetSelected(); ok && item.Item.JID.Equal(j) {
			// If the message JID is selected and the window is open, write it to the
			// history window.
			_, err = io.WriteString(history, historyLine)
			return err
		}
	}

	// If it's not selected (or the message window is not open), mark the item as
	// unread in the roster
	ok := pane.Roster().MarkUnread(j.String(), unreadSize)
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
		pane.Roster().MarkUnread(j.String(), unreadSize)
	}
	pane.Redraw()
	return nil
}

func loadBuffer(pane *ui.UI, configPath string, ev event.OpenChat, unreadSize int64) error {
	history := pane.History()
	history.SetText("")

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
	// TODO: figure out how to make the unread line full width without wrapping if
	// the terminal is resized.
	_, err = io.Copy(history, unreadMarkReader(file, tview.Styles.ContrastSecondaryTextColor, unreadSize))
	if err != nil {
		history.SetText(fmt.Sprintf("Error copying history: %v", err))
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
