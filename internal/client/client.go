// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/message"

	"mellium.im/communique/internal/client/event"
	legacybookmarks "mellium.im/legacy/bookmarks"
	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/bookmarks"
	"mellium.im/xmpp/carbons"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/disco/items"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	"mellium.im/xmpp/receipts"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"
	"mellium.im/xmpp/upload"
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

	p := c.Printer()

	pass, err := c.getPass(ctx)
	if err != nil {
		return err
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, c.timeout)
	defer cancel()

	conn, err := c.dialer.Dial(ctx, "tcp", c.addr)
	if err != nil {
		return fmt.Errorf("error dialing connection: %w", err)
	}

	var mechanisms []sasl.Mechanism
	if c.addr.Localpart() == "" {
		mechanisms = []sasl.Mechanism{sasl.Anonymous}
	} else {
		mechanisms = []sasl.Mechanism{
			sasl.ScramSha256Plus,
			sasl.ScramSha1Plus,
			sasl.ScramSha256,
			sasl.ScramSha1,
			sasl.Plain,
		}
	}
	saslFeature := xmpp.SASL("", pass, mechanisms...)
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
	c.Session, err = xmpp.NewSession(ctx, c.addr.Domain(), c.addr, conn, 0, negotiator)
	if err != nil {
		return fmt.Errorf("error negotiating session: %w", err)
	}

	c.online = true

	go func() {
		err := c.Serve(newXMPPHandler(c))
		if err != nil {
			c.logger.Print(p.Sprintf("Error while handling XMPP streams: %q", err))
		}

		c.handler(event.StatusOffline{})
		err = c.Offline()
		if err != nil {
			c.logger.Print(p.Sprintf("Error going offline: %q", err))
		}
		if err = conn.Close(); err != nil {
			c.logger.Print(p.Sprintf("Error closing the connection: %q", err))
		}
	}()

	// If the stream contained entity capabilities, go ahead and send an alert so
	// that we can fetch the disco
	if caps, ok := disco.ServerCaps(c.Session); ok {
		c.handler(event.NewCaps{
			From: c.Session.In().From,
			Caps: caps,
		})
	}

	// Discover advertised services and their features.
	serviceCtx, serviceCancel := context.WithTimeout(context.Background(), c.timeout)
	defer serviceCancel()
	itms := disco.FetchItems(serviceCtx, items.Item{JID: c.LocalAddr().Domain()}, c.Session)
	var services []jid.JID
	for itms.Next() {
		services = append(services, itms.Item().JID)
	}
	if err = itms.Err(); err != nil {
		c.logger.Print(p.Sprintf("error occured during service discovery: %v", err))
	}
	if err = itms.Close(); err != nil {
		c.logger.Print(p.Sprintf("error when closing the items iterator: %v", err))
	}
	for _, j := range services {
		_, err := c.Disco(j)
		if err != nil {
			c.logger.Print(p.Sprintf("feature discovery failed for %q: %v", j, err))
			continue
		}
	}

	// Enable message carbons.
	carbonsCtx, carbonsCancel := context.WithTimeout(context.Background(), c.timeout)
	defer carbonsCancel()
	err = carbons.Enable(carbonsCtx, c.Session)
	if err != nil {
		c.debug.Print(p.Sprintf("error enabling carbons: %q", err))
		return err
	}

	// Fetch the roster
	rosterCtx, rosterCancel := context.WithTimeout(context.Background(), c.timeout)
	defer rosterCancel()
	err = c.Roster(rosterCtx)
	if err != nil {
		c.logger.Print(p.Sprintf("error fetching roster: %q", err))
	}

	// Fetch the bookmarks
	bookmarksCtx, bookmarksCancel := context.WithTimeout(context.Background(), c.timeout)
	defer bookmarksCancel()
	err = c.Bookmarks(bookmarksCtx)
	if err != nil {
		c.logger.Print(p.Sprintf("error fetching bookmarks: %q", err))
	}

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
	p               *message.Printer
	httpClient      *http.Client
}

// Printer returns the message printer that the client is using for
// translations.
func (c *Client) Printer() *message.Printer {
	return c.p
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
	p := c.Printer()
	var iter interface {
		Next() bool
		Err() error
		Bookmark() bookmarks.Channel
		io.Closer
	}

	info, err := c.Disco(c.LocalAddr().Bare())
	if err != nil {
		c.debug.Print(p.Sprintf("error discovering bookmarks support: %v", err))
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
			c.debug.Print(p.Sprintf("error fetching version information: %v", err))
		}
		var (
			bookmarksErr string
			modName      string
		)
		if query.Name == "Prosody" {
			switch {
			case strings.HasPrefix(query.Version, "trunk"):
				modName = "mod_bookmarks"
			case strings.HasPrefix(query.Version, "0.11"):
				modName = "mod_bookmarks2"
			}
		}
		if modName != "" {
			bookmarksErr = p.Sprintf("To fix this, contact your server administrator and ask them to enable %q", modName)
		}
		c.logger.Printf(`
--
%s
%s
--
`,
			p.Sprintf("Your server does not support bookmark unification, an important feature that stops newer clients from seeing a different list of chat rooms than older clients that do not yet support the latest features."),
			bookmarksErr,
		)

		iter = legacybookmarks.Fetch(ctx, c.Session)
	} else {
		iter = bookmarks.Fetch(ctx, c.Session)
	}

	defer func() {
		e := iter.Close()
		if e != nil {
			c.debug.Print(p.Sprintf("error closing bookmarks stream: %v", e))
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
	p := c.Printer()
	rosterIQ := roster.IQ{}
	rosterIQ.Query.Ver = c.rosterVer
	iter := roster.FetchIQ(ctx, rosterIQ, c.Session)
	defer func() {
		e := iter.Close()
		if e != nil {
			c.debug.Print(p.Sprintf("Error closing roster stream: %q", e))
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

// Upload HTTP-uploads a file specified by path to the service specified by jid
// and returns the GET URL.
func (c *Client) Upload(ctx context.Context, path string, jid jid.JID) (string, error) {
	path = filepath.Clean(path)
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() {
		err := file.Close()
		if err != nil && !errors.Is(err, os.ErrClosed) {
			c.debug.Printf("error when closing file: %v", err)
		}
	}()

	info, err := file.Stat()
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return "", errors.New("cannot upload directory")
	}

	name := filepath.Base(path)

	slot, err := upload.GetSlot(ctx, upload.File{
		Name: name,
		Size: int(info.Size()),
	}, jid, c.Session)
	if err != nil {
		return "", err
	}

	req, err := slot.Put(ctx, file)
	if err != nil {
		return "", err
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 120 * time.Second}
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			c.debug.Printf("error when closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status code: %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return slot.GetURL.String(), nil
}
