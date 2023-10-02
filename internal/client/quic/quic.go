package quic

import (
	"context"
	"crypto/tls"
	"log"
	"net"

	"github.com/quic-go/quic-go"
	"mellium.im/xmpp/jid"
)

type QuicConn struct {
	Conn    quic.Connection
	Stream  quic.Stream
}

func Connect(ctx context.Context, addr jid.JID, logger *log.Logger) (*QuicConn, error) {
	domain := addr.Domainpart()
	tlsCfg := &tls.Config{
		ServerName: domain,
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"xmpp-client"},
	}

	logger.Println("Resolving IP...")

	ips, err := net.LookupIP(domain)
	if err != nil {
		return nil, err
	}

	var ipAddr string
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			ipAddr = ipv4.String()
			break
		}
	}

	logger.Println("Connecting to server...")

	conn, err := quic.DialAddr(ctx, ipAddr+":443", tlsCfg, nil)
	if err != nil {
		return nil, err
	}

	logger.Println("Opening stream...")

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}

	return &QuicConn{Conn: conn, Stream: stream}, nil
}
