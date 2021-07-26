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
	"time"

	"mellium.im/communique/internal/client/event"
	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/receipts"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"
)

// New creates a new XMPP client but does not attempt to negotiate a session or
// send an initial presence, etc.
func New(j jid.JID, logger, debug *log.Logger, opts ...Option) *Client {
	c := &Client{
		timeout: 30 * time.Second,
		addr:    j,
		dialer: &dial.Dialer{
			TLSConfig: &tls.Config{
				ServerName: j.Domain().String(),
			},
		},
		logger:          logger,
		debug:           debug,
		getPass:         emptyPass,
		handler:         emptyHandler,
		receiptsHandler: &receipts.Handler{},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
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

	conn, err := c.dialer.Dial(ctx, "tcp", c.addr)
	if err != nil {
		return fmt.Errorf("error dialing connection: %v", err)
	}

	negotiator := xmpp.NewNegotiator(xmpp.StreamConfig{
		Features: func(*xmpp.Session, ...xmpp.StreamFeature) []xmpp.StreamFeature {
			return []xmpp.StreamFeature{
				xmpp.StartTLS(c.dialer.TLSConfig),
				xmpp.SASL("", pass,
					sasl.ScramSha256Plus,
					sasl.ScramSha1Plus,
					sasl.ScramSha256,
					sasl.ScramSha1,
				),
				xmpp.BindResource(),
			}
		},
		TeeIn:  c.win,
		TeeOut: c.wout,
	})
	c.Session, err = xmpp.NewSession(ctx, c.addr.Domain(), c.addr, conn, 0, negotiator)
	if err != nil {
		return fmt.Errorf("error negotiating session: %v", err)
	}

	c.online = true
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
		if err = conn.Close(); err != nil {
			c.logger.Printf("Error closing the connection: %q", err)
		}
	}()

	// Put a special case in the roster so we can send notes to ourselves easily.
	c.handler(event.UpdateRoster(roster.Item{
		JID:  c.addr.Bare(),
		Name: "Me",
	}))

	// TODO: should this be synchronous so that when we call reconnect we fail if
	// the roster isn't fetched?
	go func() {
		rosterCtx, rosterCancel := context.WithTimeout(context.Background(), c.timeout)
		defer rosterCancel()
		err = c.Roster(rosterCtx)
		if err != nil {
			c.logger.Printf("Error fetching roster: %q", err)
		}
	}()
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

// Roster requests the users contact list.
func (c *Client) Roster(ctx context.Context) error {
	iter := roster.Fetch(ctx, c.Session)
	defer func() {
		e := iter.Close()
		if e != nil {
			c.debug.Printf("Error closing roster stream: %q", e)
		}
	}()
	for iter.Next() {
		item := iter.Item()
		if item.Name == "" {
			item.Name = item.JID.Localpart()
		}
		if item.Name == "" {
			item.Name = item.JID.Domainpart()
		}
		c.handler(event.UpdateRoster(item))
	}
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
func (c *Client) SendMessage(ctx context.Context, msg stanza.Message, payload xml.TokenReader) error {
	return c.receiptsHandler.SendMessageElement(ctx, c.Session, payload, msg)
}
