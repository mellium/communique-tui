// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/xml"
	"log"
	"time"

	"mellium.im/communique/internal/client"
	"mellium.im/communique/internal/client/event"
	"mellium.im/communique/internal/ui"
	"mellium.im/xmlstream"
	"mellium.im/xmpp/roster"
)

// newUIHandler returns a handler for events that are emitted by the UI that
// need to modify the client state.
func newUIHandler(configPath string, pane *ui.UI, c *client.Client, logger, debug *log.Logger) func(interface{}) {
	return func(ev interface{}) {
		switch e := ev.(type) {
		case event.StatusAway:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
				defer cancel()
				if err := c.Away(ctx); err != nil {
					logger.Printf("Error setting away status: %v", err)
				}
			}()
		case event.StatusOnline:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
				defer cancel()
				if err := c.Online(ctx); err != nil {
					logger.Printf("Error setting online status: %v", err)
				}
			}()
		case event.StatusBusy:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
				defer cancel()
				if err := c.Busy(ctx); err != nil {
					logger.Printf("Error setting busy status: %v", err)
				}
			}()
		case event.StatusOffline:
			go func() {
				if err := c.Offline(); err != nil {
					logger.Printf("Error going offline: %v", err)
				}
			}()
		case event.UpdateRoster:
			panic("event.UpdateRoster: not yet implemented")
		case event.ChatMessage:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err := c.SendMessage(ctx, e.Message, xmlstream.Wrap(
					xmlstream.Token(xml.CharData(e.Body)),
					xml.StartElement{Name: xml.Name{Local: "body"}},
				))
				if err != nil {
					logger.Printf("Error sending message: %v", err)
				}
				if err := writeMessage(true, pane, configPath, e); err != nil {
					logger.Printf("Error saving sent message to history: %v", err)
				}
			}()
		case event.OpenChat:
			go func() {
				var unreadSize int64
				item, ok := pane.Roster().GetItem(e.JID.Bare().String())
				if ok {
					unreadSize = item.UnreadSize()
				}
				if err := loadBuffer(pane, configPath, e, unreadSize); err != nil {
					logger.Printf("Error loading chat: %v", err)
					return
				}
				history := pane.History()
				history.ScrollToEnd()
				pane.Roster().MarkRead(e.JID.Bare().String())
			}()
		case event.CloseChat:
			history := pane.History()
			history.SetText("")
		default:
			debug.Printf("Unrecognized ui event: %q", e)
		}
	}
}

// newClientHandler returns a handler for events that are emitted by the client
// that need to modify the UI.
func newClientHandler(configPath string, pane *ui.UI, logger, debug *log.Logger) func(interface{}) {
	return func(ev interface{}) {
		switch e := ev.(type) {
		case event.StatusAway:
			pane.Away()
		case event.StatusBusy:
			pane.Busy()
		case event.StatusOnline:
			pane.Online()
		case event.StatusOffline:
			pane.Offline()
		case event.UpdateRoster:
			pane.UpdateRoster(ui.RosterItem{Item: roster.Item(e)})
		case event.ChatMessage:
			if err := writeMessage(false, pane, configPath, e); err != nil {
				logger.Printf("Error writing received message to history: %v", err)
			}
		default:
			debug.Printf("Unrecognized client event: %q", e)
		}
	}
}
