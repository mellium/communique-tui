// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"log"
	"time"

	/* #nosec */
	_ "crypto/sha1"
	_ "crypto/sha256"

	"mellium.im/communique/internal/client"
	"mellium.im/communique/internal/client/event"
	"mellium.im/communique/internal/storage"
	"mellium.im/communique/internal/ui"
	legacybookmarks "mellium.im/legacy/bookmarks"
	"mellium.im/xmpp/bookmarks"
	"mellium.im/xmpp/commands"
	"mellium.im/xmpp/crypto"
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
		case event.UpdateRoster:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := roster.Set(ctx, c.Session, e.Item)
				if err != nil {
					logger.Printf("error adding roster item %s: %v", e.JID, err)
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
		case event.UpdateBookmark:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				info, err := c.Disco(c.LocalAddr().Bare())
				if err != nil {
					debug.Printf("error discovering bookmarks support: %v", err)
				}
				var bookmarkSync = false
				for _, feature := range info.Features {
					if feature.Var == bookmarks.NSCompat {
						bookmarkSync = true
						break
					}
				}
				if !bookmarkSync {
					err = legacybookmarks.Set(ctx, c.Session, bookmarks.Channel(e))
					if err != nil {
						logger.Printf("error publishing legacy bookmark %s: %v", e.JID, err)
					}
				}
				// Always publish the bookmark to PEP bookmarks in case we're using a
				// client that only supports those in addition to this one.
				err = bookmarks.Publish(ctx, c.Session, bookmarks.Channel(e))
				if err != nil {
					logger.Printf("error publishing bookmark %s: %v", e.JID, err)
				}
			}()
		case event.DeleteBookmark:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				info, err := c.Disco(c.LocalAddr().Bare())
				if err != nil {
					debug.Printf("error discovering bookmarks support: %v", err)
				}
				var bookmarkSync = false
				for _, feature := range info.Features {
					if feature.Var == bookmarks.NSCompat {
						bookmarkSync = true
						break
					}
				}
				if !bookmarkSync {
					err = legacybookmarks.Delete(ctx, c.Session, e.JID)
					if err != nil {
						logger.Printf("error removing legacy bookmark %s: %v", e.JID, err)
					}
				}
				// Always try to delete the bookmark from PEP bookmarks in case we're
				// using a client that only supports those in addition to this one.
				err = bookmarks.Delete(ctx, c.Session, e.JID)
				// Only report the error if we're actually using PEP native bookmarks
				// though (otherwise we'll most likely report "item-not-found" every
				// single time).
				if err != nil && bookmarkSync {
					logger.Printf("error removing bookmark %s: %v", e.JID, err)
				}
			}()
		case event.ChatMessage:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()

				e, err := c.SendMessage(ctx, e)
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
		case event.OpenChannel:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if acct.Name == "" {
					acct.Name = c.LocalAddr().Localpart()
				}
				j, err := e.JID.WithResource(acct.Name)
				if err != nil {
					logger.Printf("invalid nick %s in config: %v", acct.Name, err)
					return
				}
				debug.Printf("joining room %v…", j)
				err = c.JoinMUC(ctx, j)
				if err != nil {
					logger.Printf("error joining room %s: %v", e.JID, err)
				}
			}()
		case event.OpenChat:
			go func() {
				var firstUnread string
				bare := e.JID.Bare().String()
				item, ok := pane.Roster().GetItem(bare)
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
				pane.Roster().MarkRead(bare)
				pane.Conversations().MarkRead(bare)
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
			debug.Printf("unrecognized ui event: %T(%[1]q)", e)
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
		case event.FetchBookmarks:
			for bookmark := range e.Items {
				pane.UpdateBookmarks(bookmarks.Channel(bookmark))
			}
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
			pane.UpdateRoster(ui.RosterItem{Item: e.Item})
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			err := db.UpdateRoster(ctx, e.Ver, e)
			if err != nil {
				debug.Printf("error updating roster version: %v", err)
			}
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
				err := db.InsertCaps(ctx, e.From, e.Caps)
				if err != nil {
					logger.Printf("error inserting entity capbailities hash: %v", err)
				}
			}()
		case event.NewFeatures:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				result := struct {
					Info disco.Info
					Err  error
				}{}
				discoInfo, caps, err := db.GetInfo(ctx, e.To)
				if err != nil {
					logger.Printf("error fetching info from cache: %v", err)
					logger.Print("falling back to network query…")
				}
				// If we have previously fetched disco info (and have a valid caps to
				// compare against), check that it's up to date.
				if (len(discoInfo.Features) != 0 || len(discoInfo.Identity) != 0 || len(discoInfo.Form) != 0) && caps.Hash.Available() {
					dbHash := discoInfo.Hash(caps.Hash.New())
					if caps.Ver != "" && dbHash == caps.Ver {
						// Cache !
						debug.Printf("caps cache hit for %s: %s:%s", e.To, caps.Hash, dbHash)
						result.Info = discoInfo
						e.Info <- result
						return
					}
					debug.Printf("caps cache miss for %s: %s:%s, %[2]s:%[4]s", e.To, caps.Hash, dbHash, caps.Ver)
				}

				// If we do not have any previously fetched info, or we had a cache miss
				// and need to update it, go ahead and fetch it the long way…
				discoInfo, err = disco.GetInfo(ctx, "", e.To, client.Session)
				if err != nil {
					result.Err = err
					e.Info <- result
					return
				}
				// If no caps were found in the database already, add a sha1 hash string
				// to save us a lookup later in the most common case where a client is
				// already using SHA1.
				if caps.Ver == "" {
					caps = disco.Caps{
						Hash: crypto.SHA1,
						Ver:  discoInfo.Hash(crypto.SHA1.New()),
					}
				}
				err = db.UpsertDisco(ctx, e.To, caps, discoInfo)
				if err != nil {
					logger.Printf("error saving entity caps to the database: %v", err)
				}
				result.Info = discoInfo
				e.Info <- result
			}()
		default:
			debug.Printf("unrecognized client event: %T(%[1]q)", e)
		}
	}
}
