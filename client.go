package main

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"io"
	"log"
	"os"
	"strings"

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

func newClient(ctx context.Context, addr, pass, keylogFile string, pane ui.UI, xmlIn, xmlOut, logger, debug *log.Logger) *client {
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

	s, err := xmpp.NewClientSession(
		ctx, j, "en", conn,
		xmpp.StartTLS(true, dialer.TLSConfig),
		xmpp.SASL("", pass, sasl.ScramSha256Plus, sasl.ScramSha1Plus, sasl.ScramSha256, sasl.ScramSha1),
		xmpp.BindResource(),
	)
	if err != nil {
		logger.Printf("Error establishing stream: %q", err)
		return nil
	}

	// TODO: figure out a way to get the xml log from the beginning of the session
	// including establishing the stream.

	c := &client{Session: s, pane: pane}
	if xmlOut != nil {
		c.w = xmlstream.MultiWriter(s, xml.NewEncoder(logWriter{xmlOut}))
	}
	if xmlIn != nil {
		c.r = xmlstream.TeeReader(s, xml.NewEncoder(logWriter{xmlIn}))
	}

	c.Online()

	go s.Serve(xmpp.HandlerFunc(func(t xmlstream.TokenReadWriter, start *xml.StartElement) error {
		sbuild := &strings.Builder{}
		e := xml.NewEncoder(sbuild)
		err := e.EncodeToken(start.Copy())
		if err != nil {
			debug.Printf("Error encoding start token: %q", err)
			return nil
		}
		var r xml.TokenReader = t
		var xmlInEnc *xml.Encoder
		if xmlIn != nil {
			xmlInEnc = xml.NewEncoder(logWriter{xmlIn})
			r = xmlstream.TeeReader(r, xmlInEnc)
		}
		_, err = xmlstream.Copy(e, r)
		if err != nil {
			debug.Println("Error reencoding token: %q", err)
			return nil
		}
		if xmlInEnc != nil {
			/* #nosec */
			xmlInEnc.Flush()
		}
		if x := sbuild.String(); x != "" {
			debug.Printf("In:\n\t%s", x)
		}
		return nil
	}))

	return c
}

// client represents an XMPP client.
type client struct {
	*xmpp.Session
	pane ui.UI
	w    xmlstream.TokenWriter
	r    xml.TokenReader
}

func (c *client) EncodeToken(t xml.Token) error {
	if c.w != nil {
		return c.w.EncodeToken(t)
	}
	return c.Session.EncodeToken(t)
}

func (c *client) Flush() error {
	if c.w != nil {
		return c.w.Flush()
	}
	return c.Session.Flush()
}

func (c *client) Token() (xml.Token, error) {
	if c.r != nil {
		return c.r.Token()
	}
	return c.Session.Token()
}

// Online sets the status to online.
func (c *client) Online() {
	_, err := xmlstream.Copy(c, stanza.WrapPresence(nil, stanza.AvailablePresence, nil))
	if err != nil {
		log.Printf("Error sending initial presence: %q", err)
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
		log.Printf("Error sending initial presence: %q", err)
		return
	}
	c.pane.Away()
}
