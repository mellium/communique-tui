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
	"mellium.im/communique/internal/client/jingle"
	"mellium.im/communique/internal/gui"
	"mellium.im/communique/internal/storage"
	"mellium.im/xmpp/crypto"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

func newFyneGUIHandler(g *gui.GUI, db *storage.DB, c *client.Client, logger, debug *log.Logger) func(interface{}) {
	return func(ev interface{}) {
		switch e := ev.(type) {
		case event.StatusOffline:
			go func() {
				if err := c.Offline(); err != nil {
					logger.Printf("error going offline: %v", err)
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
				if err = db.InsertMsg(ctx, e.Account, e, c.LocalAddr()); err != nil {
					logger.Printf("error writing message to database: %v", err)
				}
			}()
		case event.NewOutgoingCall:
			go func() {
				jidCopy := jid.JID(e)
				jingleRequest, err := c.CallClient.StartOutgoingCall(jidCopy)
				if err != nil {
					g.TerminateCallSession()
					logger.Println(err)
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				err = c.UnmarshalIQ(ctx, jingle.IQ{
					IQ: stanza.IQ{
						Type: stanza.SetIQ,
						To:   jidCopy,
					},
					Jingle: jingleRequest,
				}.TokenReader(), nil)
				if err != nil {
					g.TerminateCallSession()
					c.CallClient.CancelCall()
					logger.Println(err)
					return
				}
				g.ShowOutgoingCall(jidCopy)
			}()
		case event.AcceptIncomingCall:
			go func() {
				jingleResponse, err := c.CallClient.AcceptIncomingCall(e)
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				if err != nil {
					c.UnmarshalIQ(ctx, jingle.IQ{
						IQ: stanza.IQ{
							Type: stanza.SetIQ,
							To:   jid.MustParse(e.Initiator),
						},
						Jingle: jingle.JingleFailed(e.SID),
					}.TokenReader(), nil)
					return
				}
				err = c.UnmarshalIQ(ctx, jingle.IQ{
					IQ: stanza.IQ{
						Type: stanza.SetIQ,
						To:   jid.MustParse(e.Initiator),
					},
					Jingle: jingleResponse,
				}.TokenReader(), nil)
				if err != nil {
					g.TerminateCallSession()
					c.CallClient.TerminateCall()
					logger.Println(err)
					return
				}
				g.ShowCallSession()
			}()
		case event.CancelCall:
			go func() {
				jingleTerminate, err := c.CallClient.CancelCall()
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				if err != nil {
					_, _, sid := c.CallClient.GetCurrentState()
					c.UnmarshalIQ(ctx, jingle.IQ{
						IQ: stanza.IQ{
							Type: stanza.SetIQ,
							To:   c.CallClient.GetPartnerJid(),
						},
						Jingle: jingle.JingleFailed(sid),
					}.TokenReader(), nil)
					logger.Println(err)
					return
				}
				err = c.UnmarshalIQ(ctx, jingle.IQ{
					IQ: stanza.IQ{
						Type: stanza.SetIQ,
						To:   c.CallClient.GetPartnerJid(),
					},
					Jingle: jingleTerminate,
				}.TokenReader(), nil)
				if err != nil {
					logger.Println(err)
				}
				g.TerminateCallSession()
			}()
		case event.TerminateCall:
			go func() {
				jingleTerminate, err := c.CallClient.TerminateCall()
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				if err != nil {
					c.UnmarshalIQ(ctx, jingle.IQ{
						IQ: stanza.IQ{
							Type: stanza.SetIQ,
							To:   c.CallClient.GetPartnerJid(),
						},
						Jingle: jingle.JingleFailed(jingleTerminate.SID),
					}.TokenReader(), nil)
					logger.Println(err)
					return
				}
				err = c.UnmarshalIQ(ctx, jingle.IQ{
					IQ: stanza.IQ{
						Type: stanza.SetIQ,
						To:   c.CallClient.GetPartnerJid(),
					},
					Jingle: jingleTerminate,
				}.TokenReader(), nil)
				if err != nil {
					logger.Println(err)
				}
				g.TerminateCallSession()
			}()
		default:
			debug.Printf("unrecognized gui event: %T(%[1]q)", e)
		}
	}
}

func newXMPPClientHandler(g *gui.GUI, db *storage.DB, c *client.Client, logger, debug *log.Logger) func(interface{}) {
	return func(ev interface{}) {
		switch e := ev.(type) {
		case event.StatusAway:
			g.Away(jid.JID(e))
		case event.StatusBusy:
			g.Busy(jid.JID(e))
		case event.StatusOnline:
			g.Online(jid.JID(e))
		case event.StatusOffline:
			g.Offline(jid.JID(e))
		case event.FetchBookmarks:
			for _ = range e.Items {
				// Noop but still iterate to consume the channel
				// Bookmarks is currently not implemented yet in the GUI
			}
		case event.FetchRoster:
			// Only call replaceroster to consume the channel
			// roster is not yet implemented in the GUI
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := db.ReplaceRoster(ctx, e)
			if err != nil {
				logger.Printf("error updating to roster ver %q: %v", e.Ver, err)
			}
		case event.UpdateRoster:
			// Based on previous implementation
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
			g.WriteMessage(e)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := db.InsertMsg(ctx, e.Account, e, c.LocalAddr()); err != nil {
				logger.Printf("error writing message to database: %v", err)
			}
		case event.HistoryMessage:
			// Only write to DB, not yet implemented in GUI
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := db.InsertMsg(ctx, true, e.Result.Forward.Msg, c.LocalAddr()); err != nil {
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
				discoInfo, err = disco.GetInfo(ctx, "", e.To, c.Session)
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
		case event.OutgoingCallAccepted:
			go func() {
				err := c.CallClient.AcceptOutgoingCall(e)
				if err != nil {
					c.CallClient.CancelCall()
					logger.Println(err)
					return
				}
				g.ShowCallSession()
			}()
		case event.NewIncomingCall:
			go func() {
				g.ShowIncomingCall(e)
			}()
		case event.CancelCall:
			go func() {
				c.CallClient.CancelCall()
				g.TerminateCallSession()
			}()
		case event.TerminateCall:
			go func() {
				c.CallClient.TerminateCall()
				g.TerminateCallSession()
			}()
		default:
			debug.Printf("unrecognized client event: %T(%[1]q)", e)
		}
	}
}
