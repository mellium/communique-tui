// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"mellium.im/communique/internal/client"
	"mellium.im/communique/internal/client/event"
	"mellium.im/communique/internal/storage"
	"mellium.im/communique/internal/ui"
	"mellium.im/xmpp/commands"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/history"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"
)

// newUIHandler returns a handler for events that are emitted by the UI that
// need to modify the client state.
func newUIHandler(configPath string, acct account, pane *ui.UI, db *storage.DB, c *client.Client, logger, debug *log.Logger) func(interface{}) {
	return func(ev interface{}) {
		switch e := ev.(type) {
		case event.ExecCommand:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
				defer cancel()
				debug.Printf("executing command: %+v", e)
				resp, trc, err := commands.Command(e).Execute(ctx, nil, c.Session)
				if err != nil {
					logger.Printf("error executing command %q on %q: %v", e.Node, e.JID, err)
				}
				err = showCmd(pane, c, resp, trc, debug)
				if err != nil {
					logger.Printf("error showing next command for %q: %v", e.JID, err)
				}
			}()
		case event.LoadingCommands:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
				defer cancel()
				j := jid.JID(e)
				iter := commands.Fetch(ctx, j, c.Session)
				var cmd []commands.Command
				for iter.Next() {
					cmd = append(cmd, iter.Command())
				}
				err := iter.Err()
				if err != nil {
					debug.Printf("error fetching commands for %q: %v", j, err)
				}
				err = iter.Close()
				if err != nil {
					debug.Printf("error closing commands iter for %q: %v", j, err)
				}
				pane.SetCommands(j, cmd)
			}()
		case event.StatusAway:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
				defer cancel()
				if err := c.Away(ctx); err != nil {
					logger.Printf("error setting away status: %v", err)
				}
			}()
		case event.StatusOnline:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
				defer cancel()
				if err := c.Online(ctx); err != nil {
					logger.Printf("error setting online status: %v", err)
				}
			}()
		case event.StatusBusy:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
				defer cancel()
				if err := c.Busy(ctx); err != nil {
					logger.Printf("error setting busy status: %v", err)
				}
			}()
		case event.StatusOffline:
			go func() {
				if err := c.Offline(); err != nil {
					logger.Printf("error going offline: %v", err)
				}
			}()
		case event.DeleteRosterItem:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := roster.Delete(ctx, c.Session, e.JID)
				if err != nil {
					logger.Printf("error removing roster item %s: %v", e.JID, err)
				}
			}()
		case event.UpdateRoster:
			if !e.Room {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				j, err := e.JID.WithResource(acct.Name)
				if err != nil {
					logger.Printf("invalid nick %s in config: %v", acct.Name, err)
					return
				}
				err = c.JoinMUC(ctx, j)
				if err != nil {
					logger.Printf("error joining room %s: %v", e.JID, err)
				}
			}()
		case event.ChatMessage:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()

				var err error
				e, err = c.SendMessage(ctx, e)
				if err != nil {
					logger.Printf("error sending message: %v", err)
				}
				if err = writeMessage(pane, configPath, e, false); err != nil {
					logger.Printf("error saving sent message to history: %v", err)
				}
				if err = db.InsertMsg(ctx, e.Account, e, c.LocalAddr()); err != nil {
					logger.Printf("error writing message to database: %v", err)
				}
				// If we sent the message that wasn't automated (it has a body), assume
				// we've read everything before it.
				if e.Sent && e.Body != "" {
					pane.Roster().MarkRead(e.To.Bare().String())
				}
			}()
		case event.OpenChat:
			go func() {
				var firstUnread string
				item, ok := pane.Roster().GetItem(e.JID.Bare().String())
				if ok {
					firstUnread = item.FirstUnread()
				}
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				if err := loadBuffer(ctx, pane, db, configPath, roster.Item(e), firstUnread, logger); err != nil {
					logger.Printf("error loading chat: %v", err)
					return
				}
				pane.History().ScrollToEnd()
				pane.Roster().MarkRead(e.JID.Bare().String())
				pane.Redraw()
			}()
		case event.CloseChat:
			history := pane.History()
			history.SetText("")
		case event.Subscribe:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := c.Send(ctx, stanza.Presence{
				To:   jid.JID(e),
				Type: stanza.SubscribedPresence,
			}.Wrap(nil))
			if err != nil {
				logger.Printf("error sending presence pre-approval to %s: %v", jid.JID(e), err)
			}
			err = c.Send(ctx, stanza.Presence{
				To:   jid.JID(e),
				Type: stanza.SubscribePresence,
			}.Wrap(nil))
			if err != nil {
				logger.Printf("error sending presence request to %s: %v", jid.JID(e), err)
			}
		case event.PullToRefreshChat:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				// TODO: if mam:2#extended is supported, use archive ID
				_, t, err := db.BeforeID(ctx, e.JID)
				if err != nil {
					logger.Printf("error fetching earliest message info for %v from database: %v", e, err)
					return
				}
				if t.IsZero() {
					debug.Printf("no scrollback for %v", e.JID)
					return
				}
				debug.Printf("fetching scrollback before %v for %v…", t, e.JID)
				_, _, _, screenHeight := pane.GetRect()
				_, err = history.Fetch(ctx, history.Query{
					With:    e.JID,
					End:     t,
					Limit:   uint64(2 * screenHeight),
					Reverse: true,
					Last:    true,
				}, c.Session.LocalAddr().Bare(), c.Session)
				if err != nil {
					debug.Printf("error fetching scrollback for %v: %v", e.JID, err)
				}
				if err := loadBuffer(ctx, pane, db, configPath, roster.Item(e), "", logger); err != nil {
					logger.Printf("error scrollback for %v: %v", e.JID, err)
					return
				}
				// TODO: scroll to an offset that keeps context so that we don't lose
				// our position.
			}()
		default:
			debug.Printf("unrecognized ui event: %q", e)
		}
	}
}

// newClientHandler returns a handler for events that are emitted by the client
// that need to modify the UI.
func newClientHandler(configPath string, client *client.Client, pane *ui.UI, db *storage.DB, logger, debug *log.Logger) func(interface{}) {
	return func(ev interface{}) {
		switch e := ev.(type) {
		case event.StatusAway:
			pane.Away(jid.JID(e), jid.JID(e).Equal(client.LocalAddr()))
		case event.StatusBusy:
			pane.Busy(jid.JID(e), jid.JID(e).Equal(client.LocalAddr()))
		case event.StatusOnline:
			pane.Online(jid.JID(e), jid.JID(e).Equal(client.LocalAddr()))
		case event.StatusOffline:
			pane.Offline(jid.JID(e), jid.JID(e).Equal(client.LocalAddr()))
		case event.FetchRoster:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := db.ReplaceRoster(ctx, e)
			if err != nil {
				logger.Printf("error updating to roster ver %q: %v", e.Ver, err)
			}
			ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			accountBare := client.Session.LocalAddr().Bare()
			afterIDs := db.AfterID(ctx)
			ids := make(map[string]storage.AfterIDResult)
			for afterIDs.Next() {
				id := afterIDs.Result()
				ids[id.Addr.Bare().String()] = id
			}
			if err := afterIDs.Err(); err != nil {
				logger.Printf("error querying database for last seen messages: %v", err)
				return
			}
			err = db.ForRoster(ctx, func(item event.UpdateRoster) {
				pane.UpdateRoster(ui.RosterItem{Item: roster.Item(item.Item)})
				id, ok := ids[item.JID.Bare().String()]
				go func() {
					// We don't really care how long it takes to get history, and it will
					// continue to be processed even if we time out, so just set this to a
					// long time.
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
					defer cancel()
					if ok {
						// We have some history already, catch up from the last known
						// message if extended queries are supported, or from the last known
						// datetime if not.

						// if extended {
						// 	_, err := history.Fetch(ctx, history.Query{
						// 		With:    item.JID.Bare(),
						// 		AfterID: id.ID,
						// 	}, accountBare, client.Session)
						// 	if err != nil {
						// 		logger.Printf("error fetching history after %s for %s: %v", id.ID, item.JID, err)
						// 	}
						// 	return
						// }

						_, err := history.Fetch(ctx, history.Query{
							With:  item.JID.Bare(),
							Start: id.Delay,
						}, accountBare, client.Session)
						if err != nil {
							logger.Printf("error fetching history after %s for %s: %v", id.ID, item.JID, err)
						}
						return
					}

					// We don't have any history yet, so bootstrap a limited amount of
					// history from the server.
					_, _, _, screenHeight := pane.GetRect()
					_, err := history.Fetch(ctx, history.Query{
						With:    item.JID.Bare(),
						End:     time.Now(),
						Limit:   uint64(2 * screenHeight),
						Reverse: true,
						Last:    true,
					}, accountBare, client.Session)
					if err != nil {
						debug.Printf("error bootstraping history for %s: %v", item.JID, err)
					}
				}()
			})
			if err != nil {
				logger.Printf("error iterating over roster items: %v", err)
			}
		case event.UpdateRoster:
			pane.UpdateRoster(ui.RosterItem{Item: e.Item, Room: e.Room})
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			db.UpdateRoster(ctx, e.Ver, e)
		case event.Receipt:
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			err := db.MarkReceived(ctx, e)
			if err != nil {
				logger.Printf("error marking message %q as received: %v", e, err)
			}
		case event.ChatMessage:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := writeMessage(pane, configPath, e, false); err != nil {
				logger.Printf("error writing received message to chat: %v", err)
			}
			if err := db.InsertMsg(ctx, e.Account, e, client.LocalAddr()); err != nil {
				logger.Printf("error writing message to database: %v", err)
			}
			// If we sent the message that wasn't automated (it has a body), assume
			// we've read everything before it.
			if e.Sent && e.Body != "" {
				pane.Roster().MarkRead(e.To.Bare().String())
			}
		case event.HistoryMessage:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := writeMessage(pane, configPath, e.Result.Forward.Msg, false); err != nil {
				logger.Printf("error writing history message to chat: %v", err)
			}
			if err := db.InsertMsg(ctx, true, e.Result.Forward.Msg, client.LocalAddr()); err != nil {
				logger.Printf("error writing history to database: %v", err)
			}
		case event.NewCaps:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				err := db.UpdateDisco(ctx, e.From, e.Caps, func(ctx context.Context) (disco.Info, error) {
					info, err := disco.GetInfo(ctx, e.Caps.Node+"#"+e.Caps.Ver, e.From, client.Session)
					if err != nil {
						return info, err
					}
					h := info.Hash(e.Caps.Hash.New())
					if h != e.Caps.Ver {
						return info, fmt.Errorf("hash mismatch: got=%q, want=%q", h, e.Caps.Ver)
					}
					return info, nil
				})
				if err != nil {
					logger.Printf("error updating service disco cache: %v", err)
				}
			}()
		default:
			debug.Printf("unrecognized client event: %q", e)
		}
	}
}
