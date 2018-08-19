package main

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"io"
	"log"
	"net"
	"os"
	"time"

	"mellium.im/communiqué/internal/ui"
	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

type logWriter struct {
	*log.Logger
}

func (lw logWriter) Write(p []byte) (int, error) {
	lw.Println(string(p))
	return len(p), nil
}

func newClient(ctx context.Context, addr, keylogFile string, pane *ui.UI, xmlIn, xmlOut, logger, debug *log.Logger, getPass func() (string, error)) *client {
	logger.Printf("User address: %q", addr)
	j, err := jid.Parse(addr)
	if err != nil {
		logger.Printf("Error parsing user address: %q", err)
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
	conn, err := dialer.Dial(ctx, "tcp", j)
	if err != nil {
		logger.Printf("Error connecting to %q: %q", j.Domain(), err)
		return nil
	}

	c := &client{
		addr:    j,
		conn:    conn,
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

	err = c.reconnect(ctx)
	if err != nil {
		logger.Printf("Error establishing stream: %q", err)
		return nil
	}

	return c
}

func (c *client) reconnect(ctx context.Context) error {
	pass, err := c.getPass()
	if err != nil {
		return err
	}

	c.Session, err = xmpp.NegotiateSession(ctx, c.addr.Domain(), c.addr, c.conn, xmpp.NewNegotiator(xmpp.StreamConfig{
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

	c.Online()
	return nil
}

// client represents an XMPP client.
type client struct {
	*xmpp.Session
	pane    *ui.UI
	logger  *log.Logger
	conn    net.Conn
	addr    jid.JID
	win     io.Writer
	wout    io.Writer
	dialer  *xmpp.Dialer
	getPass func() (string, error)
}

// Online sets the status to online.
func (c *client) Online() {
	_, err := xmlstream.Copy(c, stanza.WrapPresence(nil, stanza.AvailablePresence, nil))
	if err != nil {
		c.logger.Printf("Error sending online presence: %q", err)
		return
	}
	c.pane.Online()
}

// Away sets the status to away.
func (c *client) Away() {
	_, err := xmlstream.Copy(
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
	c.pane.Away()
}

// Busy sets the status to busy.
func (c *client) Busy() {
	_, err := xmlstream.Copy(
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
	c.pane.Busy()
}

// Offline logs the client off.
func (c *client) Offline() {
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
