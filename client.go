package main

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"os"

	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
)

func client(ctx context.Context, addr, pass, keylogFile string, logger, debug *log.Logger) {
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
		return
	}

	_, err = xmpp.NewClientSession(
		ctx, j, "en", conn,
		xmpp.StartTLS(true, dialer.TLSConfig),
		xmpp.SASL("", pass, sasl.ScramSha256Plus, sasl.ScramSha1Plus, sasl.ScramSha256, sasl.ScramSha1),
		xmpp.BindResource(),
	)
	if err != nil {
		logger.Printf("Error establishing stream: %q", err)
		return
	}
}
