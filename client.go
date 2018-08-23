// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"io"
	"log"
	"os"
	"time"

	"mellium.im/communiqué/internal/ui"
	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/roster"
	"mellium.im/xmpp/stanza"
)

type logWriter struct {
	*log.Logger
}

func (lw logWriter) Write(p []byte) (int, error) {
	lw.Println(string(p))
	return len(p), nil
}

// newClient creates a new XMPP client but does not attempt to negotiate a
// session or send an initial presence, etc.
func newClient(configPath, addr, keylogFile string, pane *ui.UI, xmlIn, xmlOut, logger, debug *log.Logger, getPass func(context.Context) (string, error)) *client {
	var j jid.JID
	var err error
	if addr == "" {
		logger.Printf(`No user address specified, edit %q and add:

	jid="me@example.com"

`, configPath)
	} else {
		logger.Printf("User address: %q", addr)
		j, err = jid.Parse(addr)
		if err != nil {
			logger.Printf("Error parsing user address: %q", err)
		}
	}

	var keylog io.Writer
	if keylogFile != "" {
		keylog, err = os.OpenFile(keylogFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0400)
		if err != nil {
			logger.Printf("Error creating keylog file: %q", err)
		}
	}
	dialer := &xmpp.Dialer{
		TLSConfig: &tls.Config{
			ServerName:   j.Domain().String(),
			KeyLogWriter: keylog,
		},
	}

	c := &client{
		addr:    j,
		dialer:  dialer,
		logger:  logger,
		pane:    pane,
		getPass: getPass,
	}
	if xmlIn != nil {
		c.win = logWriter{xmlIn}
	}
	if xmlOut != nil {
		c.wout = logWriter{xmlOut}
	}

	pane.Offline()
	return c
}

func (c *client) reconnect(ctx context.Context) error {
	if c.online {
		return nil
	}

	pass, err := c.getPass(ctx)
	if err != nil {
		return err
	}
	conn, err := c.dialer.Dial(ctx, "tcp", c.addr)
	if err != nil {
		return err
	}

	c.Session, err = xmpp.NegotiateSession(ctx, c.addr.Domain(), c.addr, conn, xmpp.NewNegotiator(xmpp.StreamConfig{
		Features: []xmpp.StreamFeature{
			xmpp.StartTLS(true, c.dialer.TLSConfig),
			xmpp.SASL("", pass, sasl.ScramSha256Plus, sasl.ScramSha1Plus, sasl.ScramSha256, sasl.ScramSha1),
			xmpp.BindResource(),
		},
		TeeIn:  c.win,
		TeeOut: c.wout,
	}))
	if err != nil {
		return err
	}

	c.online = true
	go func() {
		err := c.Serve(nil)
		if err != nil {
			c.logger.Printf("Error while handling XMPP streams: %q", err)
		}
		c.online = false
		c.pane.Offline()
		if err = conn.Close(); err != nil {
			c.logger.Printf("Error closing the connection: %q", err)
		}
	}()
	return nil
}

// client represents an XMPP client.
type client struct {
	*xmpp.Session
	pane    *ui.UI
	logger  *log.Logger
	addr    jid.JID
	win     io.Writer
	wout    io.Writer
	dialer  *xmpp.Dialer
	getPass func(context.Context) (string, error)
	online  bool
}

// Online sets the status to online.
// The provided context is only used if the client was previously offline and we
// have to re-establish the session.
func (c *client) Online(ctx context.Context) {
	err := c.reconnect(ctx)
	if err != nil {
		c.logger.Println(err)
		return
	}

	_, err = xmlstream.Copy(c, stanza.WrapPresence(nil, stanza.AvailablePresence, nil))
	if err != nil {
		c.logger.Printf("Error sending online presence: %q", err)
		return
	}
	if err = c.Flush(); err != nil {
		c.logger.Printf("Error sending online presence: %q", err)
		return
	}
	c.pane.Online()
}

// Roster requests the users contact list.
func (c *client) Roster(ctx context.Context) error {
	rosterIQ := roster.IQ{}
	r, err := c.Send(ctx, rosterIQ.TokenReader())
	if err != nil {
		return err
	}

	// TODO: don't parse this all at once, do it incrementally.
	d := xml.NewTokenDecoder(r)
	err = d.Decode(&rosterIQ)
	if err != nil && err != io.EOF {
		return err
	}

	for _, item := range rosterIQ.Query.Item {
		if item.Name == "" {
			item.Name = item.JID.Localpart()
		}
		if item.Name == "" {
			item.Name = item.JID.Domainpart()
		}
		c.pane.AddRoster(ui.RosterItem{Item: item})
	}

	return nil
}

// Away sets the status to away.
func (c *client) Away(ctx context.Context) {
	err := c.reconnect(ctx)
	if err != nil {
		c.logger.Println(err)
		return
	}

	_, err = xmlstream.Copy(
		c,
		stanza.WrapPresence(
			nil,
			stanza.AvailablePresence,
			xmlstream.Wrap(
				xmlstream.ReaderFunc(func() (xml.Token, error) {
					return xml.CharData("away"), io.EOF
				}),
				xml.StartElement{Name: xml.Name{Local: "show"}},
			)))
	if err != nil {
		c.logger.Printf("Error sending away presence: %q", err)
		return
	}
	if err = c.Flush(); err != nil {
		c.logger.Printf("Error sending away presence: %q", err)
		return
	}
	c.pane.Away()
}

// Busy sets the status to busy.
func (c *client) Busy(ctx context.Context) {
	err := c.reconnect(ctx)
	if err != nil {
		c.logger.Println(err)
		return
	}

	_, err = xmlstream.Copy(
		c,
		stanza.WrapPresence(
			nil,
			stanza.AvailablePresence,
			xmlstream.Wrap(
				xmlstream.ReaderFunc(func() (xml.Token, error) {
					return xml.CharData("dnd"), io.EOF
				}),
				xml.StartElement{Name: xml.Name{Local: "show"}},
			)))
	if err != nil {
		c.logger.Printf("Error sending busy presence: %q", err)
		return
	}
	if err = c.Flush(); err != nil {
		c.logger.Printf("Error sending busy presence: %q", err)
		return
	}
	c.pane.Busy()
}

// Offline logs the client off.
func (c *client) Offline() {
	if !c.online {
		c.pane.Offline()
		return
	}

	err := c.SetCloseDeadline(time.Now().Add(30 * time.Second))
	if err != nil {
		c.logger.Printf("Error setting close deadline: %q", err)
		// Don't return; we still want to attempt to close the connection.
	}
	err = c.Close()
	if err != nil {
		c.logger.Printf("Error logging off: %q", err)
		return
	}
}
