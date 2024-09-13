// Copyright 2024 The Mellium Contributors.
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
	"mellium.im/xmpp/bookmarks"
	"mellium.im/xmpp/crypto"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/history"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/roster"
)

// newClientHandler returns a handler for events that are emitted by the client
// that need to modify the UI.
func newClientHandler(client *client.Client, pane *ui.UI, db *storage.DB, logger, debug *log.Logger) func(interface{}) {
	p := client.Printer()
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
				logger.Print(p.Sprintf("error updating to roster ver %q: %v", e.Ver, err))
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
				logger.Print(p.Sprintf("error querying database for last seen messages: %v", err))
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
							logger.Print(p.Sprintf("error fetching history after %s for %s: %v", id.ID, item.JID, err))
						}
						return
					}

					// We don't have any history yet, so bootstrap a limited amount of
					// history from the server.
					_, _, _, screenHeight := pane.GetRect()
					_, err := history.Fetch(ctx, history.Query{
						With:    item.JID.Bare(),
						End:     time.Now(),
						Limit:   uint64(2 * screenHeight), // #nosec G115
						Reverse: true,
						Last:    true,
					}, accountBare, client.Session)
					if err != nil {
						debug.Print(p.Sprintf("error bootstraping history for %s: %v", item.JID, err))
					}
				}()
			})
			if err != nil {
				logger.Print(p.Sprintf("error iterating over roster items: %v", err))
			}
		case event.UpdateRoster:
			pane.UpdateRoster(ui.RosterItem{Item: e.Item})
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			err := db.UpdateRoster(ctx, e.Ver, e)
			if err != nil {
				debug.Print(p.Sprintf("error updating roster version: %v", err))
			}
		case event.Receipt:
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			err := db.MarkReceived(ctx, e)
			if err != nil {
				logger.Print(p.Sprintf("error marking message %q as received: %v", e, err))
			}
		case event.ChatMessage:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := writeMessage(pane, e, false); err != nil {
				logger.Print(p.Sprintf("error writing received message to chat: %v", err))
			}
			if err := db.InsertMsg(ctx, e.Account, e, client.LocalAddr()); err != nil {
				logger.Print(p.Sprintf("error writing message to database: %v", err))
			}
			// If we sent the message that wasn't automated (it has a body), assume
			// we've read everything before it.
			if e.Sent && e.Body != "" {
				pane.Roster().MarkRead(e.To.Bare().String())
			}
		case event.HistoryMessage:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := writeMessage(pane, e.Result.Forward.Msg, false); err != nil {
				logger.Print(p.Sprintf("error writing history message to chat: %v", err))
			}
			if err := db.InsertMsg(ctx, true, e.Result.Forward.Msg, client.LocalAddr()); err != nil {
				logger.Print(p.Sprintf("error writing history to database: %v", err))
			}
		case event.NewCaps:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				err := db.InsertCaps(ctx, e.From, e.Caps)
				if err != nil {
					logger.Print(p.Sprintf("error inserting entity capbailities hash: %v", err))
				}
			}()
		case event.NewFeatures:
			go newFeatures(e, client, db, debug, logger)
		default:
			debug.Print(p.Sprintf("unrecognized client event: %T(%[1]q)", e))
		}
	}
}

func newFeatures(e event.NewFeatures, client *client.Client, db *storage.DB, debug, logger *log.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result := struct {
		Info disco.Info
		Err  error
	}{}
	p := client.Printer()
	discoInfo, caps, err := db.GetInfo(ctx, e.To)
	if err != nil {
		logger.Print(p.Sprintf("error fetching info from cache: %v", err))
		logger.Print(p.Sprintf("falling back to network query…"))
	}
	// If we have previously fetched disco info (and have a valid caps to
	// compare against), check that it's up to date.
	if (len(discoInfo.Features) != 0 || len(discoInfo.Identity) != 0 || len(discoInfo.Form) != 0) && caps.Hash.Available() {
		dbHash := discoInfo.Hash(caps.Hash.New())
		if caps.Ver != "" && dbHash == caps.Ver {
			// Cache !
			debug.Print(p.Sprintf("caps cache hit for %s: %s:%s", e.To, caps.Hash, dbHash))
			result.Info = discoInfo
			e.Info <- result
			return
		}
		debug.Print(p.Sprintf("caps cache miss for %s: %s:%s, %[2]s:%[4]s", e.To, caps.Hash, dbHash, caps.Ver))
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
		logger.Print(p.Sprintf("error saving entity caps to the database: %v", err))
	}
	result.Info = discoInfo
	e.Info <- result
}
