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
	clientevent "mellium.im/communique/internal/client/event"
	"mellium.im/communique/internal/storage"
	"mellium.im/communique/internal/ui"
	"mellium.im/communique/internal/ui/event"
	legacybookmarks "mellium.im/legacy/bookmarks"
	"mellium.im/xmpp/bookmarks"
	"mellium.im/xmpp/commands"
	"mellium.im/xmpp/disco/info"
	"mellium.im/xmpp/history"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"
	"mellium.im/xmpp/upload"
)

// newUIHandler returns a handler for events that are emitted by the UI that
// need to modify the client state.
func newUIHandler(acct account, pane *ui.UI, db *storage.DB, c *client.Client, logger, debug *log.Logger) func(interface{}) {
	p := pane.Printer()
	return func(ev interface{}) {
		switch e := ev.(type) {
		case event.ExecCommand:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
				defer cancel()
				debug.Print(p.Sprintf("executing command: %+v", e))
				resp, trc, err := commands.Command(e).Execute(ctx, nil, c.Session)
				if err != nil {
					logger.Print(p.Sprintf("error executing command %q on %q: %v", e.Node, e.JID, err))
				}
				err = showCmd(pane, c, resp, trc, debug)
				if err != nil {
					logger.Print(p.Sprintf("error showing next command for %q: %v", e.JID, err))
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
					debug.Print(p.Sprintf("error fetching commands for %q: %v", j, err))
				}
				err = iter.Close()
				if err != nil {
					debug.Print(p.Sprintf("error closing commands iter for %q: %v", j, err))
				}
				pane.SetCommands(j, cmd)
			}()
		case event.StatusAway:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
				defer cancel()
				if err := c.Away(ctx); err != nil {
					logger.Print(p.Sprintf("error setting away status: %v", err))
				}
			}()
		case event.StatusOnline:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
				defer cancel()
				if err := c.Online(ctx); err != nil {
					logger.Print(p.Sprintf("error setting online status: %v", err))
				}
			}()
		case event.StatusBusy:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
				defer cancel()
				if err := c.Busy(ctx); err != nil {
					logger.Print(p.Sprintf("error setting busy status: %v", err))
				}
			}()
		case event.StatusOffline:
			go func() {
				if err := c.Offline(); err != nil {
					logger.Print(p.Sprintf("error going offline: %v", err))
				}
			}()
		case event.UpdateRoster:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := roster.Set(ctx, c.Session, e.Item)
				if err != nil {
					logger.Print(p.Sprintf("error adding roster item %s: %v", e.JID, err))
				}
			}()
		case event.DeleteRosterItem:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := roster.Delete(ctx, c.Session, e.JID)
				if err != nil {
					logger.Print(p.Sprintf("error removing roster item %s: %v", e.JID, err))
				}
			}()
		case event.UpdateBookmark:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				info, err := c.Disco(c.LocalAddr().Bare())
				if err != nil {
					debug.Print(p.Sprintf("error discovering bookmarks support: %v", err))
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
						logger.Print(p.Sprintf("error publishing legacy bookmark %s: %v", e.JID, err))
					}
				}
				// Always publish the bookmark to PEP bookmarks in case we're using a
				// client that only supports those in addition to this one.
				err = bookmarks.Publish(ctx, c.Session, bookmarks.Channel(e))
				if err != nil {
					logger.Print(p.Sprintf("error publishing bookmark %s: %v", e.JID, err))
				}
			}()
		case event.DeleteBookmark:
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				info, err := c.Disco(c.LocalAddr().Bare())
				if err != nil {
					debug.Print(p.Sprintf("error discovering bookmarks support: %v", err))
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
						logger.Print(p.Sprintf("error removing legacy bookmark %s: %v", e.JID, err))
					}
				}
				// Always try to delete the bookmark from PEP bookmarks in case we're
				// using a client that only supports those in addition to this one.
				err = bookmarks.Delete(ctx, c.Session, e.JID)
				// Only report the error if we're actually using PEP native bookmarks
				// though (otherwise we'll most likely report "item-not-found" every
				// single time).
				if err != nil && bookmarkSync {
					logger.Print(p.Sprintf("error removing bookmark %s: %v", e.JID, err))
				}
			}()
		case event.ChatMessage:
			go sendMessage(c, logger, db, pane, e)
		case event.OpenChannel:
			go openChannel(e, c, acct, debug, logger)
		case event.OpenChat:
			go openChat(e, pane, db, logger)
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
				logger.Print(p.Sprintf("error sending presence pre-approval to %s: %v", jid.JID(e), err))
			}
			err = c.Send(ctx, stanza.Presence{
				To:   jid.JID(e),
				Type: stanza.SubscribePresence,
			}.Wrap(nil))
			if err != nil {
				logger.Print(p.Sprintf("error sending presence request to %s: %v", jid.JID(e), err))
			}
		case event.PullToRefreshChat:
			go pullToRefresh(e, c, pane, db, debug, logger)
		case event.UploadFile:
			go uploadFile(c, logger, debug, db, pane, e)
		default:
			debug.Print(p.Sprintf("unrecognized ui event: %T(%[1]q)", e))
		}
	}
}

// sendMessage sends a message and writes it to the database and UI.
func sendMessage(c *client.Client, logger *log.Logger, db *storage.DB, ui *ui.UI, message event.ChatMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	p := c.Printer()

	msg, err := c.SendMessage(ctx, clientevent.ChatMessage{
		Message: message.Message,
		Body:    message.Body,
	})
	if err != nil {
		logger.Print(p.Sprintf("error sending message: %v", err))
	}
	if err = writeMessage(ui, msg, false); err != nil {
		logger.Print(p.Sprintf("error saving sent message to history: %v", err))
	}
	if err = db.InsertMsg(ctx, msg.Account, msg, c.LocalAddr()); err != nil {
		logger.Print(p.Sprintf("error writing message to database: %v", err))
	}
	// If we sent the message that wasn't automated (it has a body), assume
	// we've read everything before it.
	if message.Body != "" {
		ui.Roster().MarkRead(message.To.Bare().String())
	}
}

// uploadFile HTTP-uploads a file and sends the GET URL to recipients.
func uploadFile(c *client.Client, logger *log.Logger, debug *log.Logger, db *storage.DB, ui *ui.UI, ev event.UploadFile) {
	p := c.Printer()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	services, err := db.GetServices(ctx, info.Feature{Var: upload.NS})
	if err != nil {
		logger.Print(p.Sprintf("could not get the upload services: %v", err))
		return
	}
	if len(services) == 0 {
		logger.Print(p.Sprintf("no upload service available"))
		return
	}
	url, err := c.Upload(ctx, ev.Path, services[0])
	if err != nil {
		logger.Print(p.Sprintf("could not upload %q: %v", ev.Path, err))
		return
	}
	debug.Print(p.Sprintf("uploaded %q as %s", ev.Path, url))
	ev.Message.Body = url
	sendMessage(c, logger, db, ui, ev.Message)
}

func openChat(e event.OpenChat, pane *ui.UI, db *storage.DB, logger *log.Logger) {
	var firstUnread string
	bare := e.JID.Bare().String()
	item, ok := pane.Roster().GetItem(bare)
	if ok {
		firstUnread = item.FirstUnread()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := loadBuffer(ctx, pane, db, roster.Item(e), firstUnread, logger); err != nil {
		p := pane.Printer()
		logger.Print(p.Sprintf("error loading chat: %v", err))
		return
	}
	pane.History().ScrollToEnd()
	pane.Roster().MarkRead(bare)
	pane.Conversations().MarkRead(bare)
	pane.Redraw()
}

func openChannel(e event.OpenChannel, c *client.Client, acct account, debug, logger *log.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if acct.Name == "" {
		acct.Name = c.LocalAddr().Localpart()
	}
	p := c.Printer()
	j, err := e.JID.WithResource(acct.Name)
	if err != nil {
		logger.Print(p.Sprintf("invalid nick %s in config: %v", acct.Name, err))
		return
	}
	debug.Print(p.Sprintf("joining room %v…", j))
	err = c.JoinMUC(ctx, j)
	if err != nil {
		logger.Print(p.Sprintf("error joining room %s: %v", e.JID, err))
	}
}

func pullToRefresh(e event.PullToRefreshChat, c *client.Client, pane *ui.UI, db *storage.DB, debug, logger *log.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	p := c.Printer()
	// TODO: if mam:2#extended is supported, use archive ID
	_, t, err := db.BeforeID(ctx, e.JID)
	if err != nil {
		logger.Print(p.Sprintf("error fetching earliest message info for %v from database: %v", e, err))
		return
	}
	if t.IsZero() {
		debug.Print(p.Sprintf("no scrollback for %v", e.JID))
		return
	}
	debug.Print(p.Sprintf("fetching scrollback before %v for %v…", t, e.JID))
	_, _, _, screenHeight := pane.GetRect()
	_, err = history.Fetch(ctx, history.Query{
		With:    e.JID,
		End:     t,
		Limit:   uint64(2 * screenHeight), // #nosec G115
		Reverse: true,
		Last:    true,
	}, c.Session.LocalAddr().Bare(), c.Session)
	if err != nil {
		debug.Print(p.Sprintf("error fetching scrollback for %v: %v", e.JID, err))
	}
	if err := loadBuffer(ctx, pane, db, roster.Item(e), "", logger); err != nil {
		logger.Print(p.Sprintf("error loading scrollback into pane for %v: %v", e.JID, err))
		return
	}
	// TODO: scroll to an offset that keeps context so that we don't lose
	// our position.
}
