// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"mellium.im/communique/internal/client/event"
	"mellium.im/communique/internal/client/jingle"
	"mellium.im/communique/internal/client/quic"
	legacybookmarks "mellium.im/legacy/bookmarks"
	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/bookmarks"
	"mellium.im/xmpp/carbons"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	"mellium.im/xmpp/receipts"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"
	"mellium.im/xmpp/version"
)

func noopHandler(interface{}) {}

// New creates a new XMPP client but does not attempt to negotiate a session or
// send an initial presence, etc.
func New(j jid.JID, logger, debug *log.Logger, opts ...Option) *Client {
	var c *Client
	c = &Client{
		timeout: 30 * time.Second,
		addr:    j,
		dialer: &dial.Dialer{
			TLSConfig: &tls.Config{
				ServerName: j.Domain().String(),
				MinVersion: tls.VersionTLS12,
			},
		},
		logger:  logger,
		debug:   debug,
		getPass: emptyPass,
		handler: noopHandler,
		receiptsHandler: &receipts.Handler{
			Unhandled: func(id string) { c.handler(event.Receipt(id)) },
		},
		// TODO: mediated muc invitations
		mucClient: &muc.Client{},
		channels:  make(map[string]*muc.Channel),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Disco fetches the service discovery information associated with the provided
// JID by sending an event capable of returning the results over a channel.
func (c *Client) Disco(j jid.JID) (disco.Info, error) {
	infoChan := make(chan struct {
		Info disco.Info
		Err  error
	})
	c.handler(event.NewFeatures{
		To:   j,
		Info: infoChan,
	})
	result := <-infoChan
	return result.Info, result.Err
}

// Handler configures a handler function to be used for events emitted by the
// client.
//
// For a list of events that any handler function may handle, see the event
// package.
func (c *Client) Handler(h func(interface{})) {
	if h == nil {
		c.handler = noopHandler
		return
	}
	c.handler = h
}

func (c *Client) reconnect(ctx context.Context) error {
	if c.online {
		return nil
	}

	pass, err := c.getPass(ctx)
	if err != nil {
		return err
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var conn io.ReadWriter
	var quicConn *quic.QuicConn
	if c.useQuic {
		quicConn, err = quic.Connect(ctx, c.addr, c.logger)
	} else {
		var dialConn net.Conn
		dialConn, err = c.dialer.Dial(ctx, "tcp", c.addr)
		if err != nil {
			return fmt.Errorf("error dialing connection: %v", err)
		}
		tcpConn := dialConn.(*net.TCPConn)
		err = tcpConn.SetReadBuffer(1048576)
		if err != nil {
			c.logger.Println(err)
		}
		err = tcpConn.SetWriteBuffer(1048576)
		if err != nil {
			c.logger.Println(err)
		}
		conn = tcpConn
	}
	if err != nil {
		return fmt.Errorf("error dialing connection: %v", err)
	}
	if c.useQuic {
		c.quicConn = quicConn
		conn = quicConn.Stream
	}

	saslFeature := xmpp.SASL("", pass,
		// sasl.ScramSha256Plus,
		// sasl.ScramSha1Plus,
		// sasl.ScramSha256,
		sasl.ScramSha1,
		sasl.Plain,
	)
	if c.noTLS {
		saslFeature.Necessary &^= xmpp.Secure
	}

	negotiator := xmpp.NewNegotiator(func(*xmpp.Session, *xmpp.StreamConfig) xmpp.StreamConfig {
		return xmpp.StreamConfig{
			Features: []xmpp.StreamFeature{
				disco.StreamFeature(),
				xmpp.StartTLS(c.dialer.TLSConfig),
				saslFeature,
				roster.Versioning(),
				xmpp.BindResource(),
			},
			TeeIn:  c.win,
			TeeOut: c.wout,
		}
	})
	state := xmpp.SessionState(0)
	if c.useQuic {
		state = state | xmpp.Secure
	}
	c.Session, err = xmpp.NewSession(ctx, c.addr.Domain(), c.addr, conn, state, negotiator)
	if err != nil {
		return fmt.Errorf("error negotiating session: %v", err)
	}

	c.online = true

	c.debug.Println("Finished session negotiation")

	go func() {
		err := c.Serve(newXMPPHandler(c))
		if err != nil {
			c.logger.Printf("Error while handling XMPP streams: %q", err)
		}

		c.handler(event.StatusOffline{})
		err = c.Offline()
		if err != nil {
			c.logger.Printf("Error going offline: %q", err)
		}
		if c.useQuic {
			err = c.quicConn.Conn.CloseWithError(0x42, "Closing the connection")
		} else {
			err = c.Session.Conn().Close()
		}
		if err != nil {
			c.logger.Printf("Error closing the connection: %q", err)
		}
	}()

	// If the stream contained entity capabilities, go ahead and send an alert so
	// that we can fetch the disco
	c.debug.Println("Fetching server caps")
	if caps, ok := disco.ServerCaps(c.Session); ok {
		c.handler(event.NewCaps{
			From: c.Session.In().From,
			Caps: caps,
		})
	}

	// Enable message carbons.
	c.debug.Println("Enabling message carbons")
	carbonsCtx, carbonsCancel := context.WithTimeout(context.Background(), c.timeout)
	defer carbonsCancel()
	err = carbons.Enable(carbonsCtx, c.Session)
	if err != nil {
		c.debug.Printf("error enabling carbons: %q", err)
		return err
	}

	// Fetch the roster
	c.debug.Println("Fetching roster")
	rosterCtx, rosterCancel := context.WithTimeout(context.Background(), c.timeout)
	defer rosterCancel()
	err = c.Roster(rosterCtx)
	if err != nil {
		c.logger.Printf("error fetching roster: %q", err)
	}

	// Fetch the bookmarks
	c.debug.Println("Fetching bookmarks")
	bookmarksCtx, bookmarksCancel := context.WithTimeout(context.Background(), c.timeout)
	defer bookmarksCancel()
	err = c.Bookmarks(bookmarksCtx)
	if err != nil {
		c.logger.Printf("error fetching bookmarks: %q", err)
	}

	// Init CallClient
	c.CallClient = jingle.New(c.LocalAddr(), newOnIceCandidateHandler(c), c.debug)

	return nil
}

// Client represents an XMPP client.
type Client struct {
	*xmpp.Session
	timeout         time.Duration
	logger          *log.Logger
	debug           *log.Logger
	addr            jid.JID
	win             io.Writer
	wout            io.Writer
	dialer          *dial.Dialer
	getPass         func(context.Context) (string, error)
	online          bool
	handler         func(interface{})
	receiptsHandler *receipts.Handler
	rosterVer       string
	noTLS           bool
	mucClient       *muc.Client
	chanM           sync.Mutex
	channels        map[string]*muc.Channel
	useQuic         bool
	quicConn        *quic.QuicConn
	CallClient      *jingle.CallClient
}

// Online sets the status to online.
// The provided context is used if the client was previously offline and we
// have to re-establish the session, so if it includes a timeout make sure to
// account for the fact that we might reconnect.
func (c *Client) Online(ctx context.Context) error {
	err := c.reconnect(ctx)
	if err != nil {
		return err
	}

	err = c.Send(ctx, stanza.Presence{Type: stanza.AvailablePresence}.Wrap(nil))
	if err != nil {
		return err
	}
	return nil
}

// Bookmarks fetches the users list of bookmarked chat rooms.
func (c *Client) Bookmarks(ctx context.Context) error {
	var iter interface {
		Next() bool
		Err() error
		Bookmark() bookmarks.Channel
		io.Closer
	}

	info, err := c.Disco(c.LocalAddr().Bare())
	if err != nil {
		c.debug.Printf("error discovering bookmarks support: %v", err)
	}

	useLegacyBookmarks := true
	for _, feature := range info.Features {
		if feature.Var == bookmarks.NSCompat {
			useLegacyBookmarks = false
			break
		}
	}

	if useLegacyBookmarks {
		query, err := version.Get(ctx, c.Session, c.Session.LocalAddr().Domain())
		if err != nil {
			c.debug.Printf(`error fetching version information: %v`, err)
		}
		var bookmarksErr string
		if query.Name == "Prosody" {
			switch {
			case strings.HasPrefix(query.Version, "trunk"):
				bookmarksErr = `
To fix this, contact your server administrator and ask them to enable "mod_bookmarks".`
			case strings.HasPrefix(query.Version, "0.11"):
				bookmarksErr = `
To fix this, contact your server administrator and ask them to enable "mod_bookmarks2".`
			}
		}
		c.logger.Printf(`

	--
Your server does not support bookmark unification, an important feature that stops newer clients from seeing a different list of chat rooms than older clients that do not yet support the latest features.%s
	--

`, bookmarksErr)
		iter = legacybookmarks.Fetch(ctx, c.Session)
	} else {
		iter = bookmarks.Fetch(ctx, c.Session)
	}

	defer func() {
		e := iter.Close()
		if e != nil {
			c.debug.Printf("error closing bookmarks stream: %v", e)
		}
	}()
	items := make(chan event.UpdateBookmark)
	go func() {
		defer close(items)
		for iter.Next() {
			items <- event.UpdateBookmark(iter.Bookmark())
		}
	}()
	c.handler(event.FetchBookmarks{
		Items: items,
	})
	err = iter.Err()
	if err == io.EOF {
		err = nil
	}
	return err
}

// Roster requests the users contact list.
func (c *Client) Roster(ctx context.Context) error {
	rosterIQ := roster.IQ{}
	rosterIQ.Query.Ver = c.rosterVer
	iter := roster.FetchIQ(ctx, rosterIQ, c.Session)
	defer func() {
		e := iter.Close()
		if e != nil {
			c.debug.Printf("Error closing roster stream: %q", e)
		}
	}()
	items := make(chan event.UpdateRoster)
	go func() {
		defer close(items)
		for iter.Next() {
			item := iter.Item()
			if item.Name == "" {
				item.Name = item.JID.Localpart()
			}
			if item.Name == "" {
				item.Name = item.JID.Domainpart()
			}
			items <- event.UpdateRoster{
				Item: item,
				Ver:  iter.Version(),
			}
		}
	}()
	c.handler(event.FetchRoster{
		Ver:   iter.Version(),
		Items: items,
	})
	err := iter.Err()
	if err == io.EOF {
		err = nil
	}

	return err
}

// Away sets the status to away.
func (c *Client) Away(ctx context.Context) error {
	err := c.reconnect(ctx)
	if err != nil {
		return err
	}

	err = c.Send(
		ctx,
		stanza.Presence{Type: stanza.AvailablePresence}.Wrap(
			xmlstream.Wrap(
				xmlstream.ReaderFunc(func() (xml.Token, error) {
					return xml.CharData("away"), io.EOF
				}),
				xml.StartElement{Name: xml.Name{Local: "show"}},
			)))
	if err != nil {
		return err
	}
	return nil
}

// Busy sets the status to busy.
func (c *Client) Busy(ctx context.Context) error {
	err := c.reconnect(ctx)
	if err != nil {
		return err
	}

	err = c.Send(
		ctx,
		stanza.Presence{Type: stanza.AvailablePresence}.Wrap(
			xmlstream.Wrap(
				xmlstream.ReaderFunc(func() (xml.Token, error) {
					return xml.CharData("dnd"), io.EOF
				}),
				xml.StartElement{Name: xml.Name{Local: "show"}},
			)))
	if err != nil {
		return err
	}
	return nil
}

// Offline logs the client off.
func (c *Client) Offline() error {
	if !c.online {
		return nil
	}
	defer func() {
		c.online = false
	}()

	/* #nosec */
	_ = c.SetCloseDeadline(time.Now().Add(2 * c.timeout))
	// Don't handle the error when setting the close deadline; we still want to
	// attempt to close the connection.

	err := c.Close()
	if err != nil {
		return err
	}
	return nil
}

// Timeout is the read/write timeout used by the client.
func (c *Client) Timeout() time.Duration {
	return c.timeout
}

// SendMessage encodes the provided message to the output stream and adds a
// request for a receipt. It then blocks until the message receipt is received,
// or the context is canceled.
func (c *Client) SendMessage(ctx context.Context, msg event.ChatMessage) (event.ChatMessage, error) {
	if msg.ID == "" {
		id := randomID()
		msg.ID = id
		msg.OriginID.ID = id
	}
	msg.From = c.Session.LocalAddr()

	return msg, c.Session.Send(ctx, receipts.Request(encodeMessage(msg)))
}

func omitEmpty(s string, name xml.Name) xml.TokenReader {
	if s == "" {
		// Returns nil, EOF
		return xmlstream.Token(nil)
	}
	return xmlstream.Wrap(xmlstream.Token(xml.CharData(s)), xml.StartElement{Name: name})
}

func encodeMessage(e event.ChatMessage) xml.TokenReader {
	// Make sure we've already set the namespace, otherwise receipt wrapping and
	// the like doesn't work.
	e.Message.XMLName = xml.Name{Space: "jabber:client", Local: "message"}
	return e.Message.Wrap(xmlstream.MultiReader(
		omitEmpty(e.Body, xml.Name{Local: "body"}),
		e.OriginID.TokenReader(),
	))
}

// JoinMUC joins a multi-user chat, or rejoins it if it was already joined.
func (c *Client) JoinMUC(ctx context.Context, room jid.JID) error {
	s := room.Bare().String()
	c.chanM.Lock()
	defer c.chanM.Unlock()
	mucChan, ok := c.channels[s]
	if ok {
		return mucChan.Join(ctx)
	}
	mucChan, err := c.mucClient.Join(ctx, room, c.Session, muc.MaxHistory(100))
	if err != nil {
		return err
	}
	c.channels[s] = mucChan
	return nil
}

// LeaveMUC exits the given multi-user chat..
func (c *Client) LeaveMUC(ctx context.Context, room jid.JID, reason string) error {
	s := room.Bare().String()
	c.chanM.Lock()
	defer c.chanM.Unlock()
	mucChan, ok := c.channels[s]
	if !ok {
		return nil
	}
	err := mucChan.Leave(ctx, reason)
	if err != nil {
		return err
	}
	delete(c.channels, s)
	return nil
}
