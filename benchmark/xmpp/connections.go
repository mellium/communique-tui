package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"mellium.im/communique/internal/client"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
)

const (
	account = "kenshin@slickerius.com"
	pwd     = "himura"
)

var (
	clientList  []*client.Client = []*client.Client{}
	clientCount int64
	clientMu    sync.Mutex
)

func closeClient() {
	clientMu.Lock()
	defer clientMu.Unlock()
	for _, client := range clientList {
		client.Close()
	}
}

func addClient(c *client.Client) {
	clientMu.Lock()
	defer clientMu.Unlock()
	// fmt.Printf("%s Registered as Client %d\n", c.LocalAddr(), clientCount)
	clientList = append(clientList, c)
	clientCount++
}

func getPass(ctx context.Context) (string, error) {
	return pwd, nil
}

func newConnection(ctx context.Context) error {
	logger := log.New(io.Discard, "ACCOUNT ", log.LstdFlags)
	debug := log.New(io.Discard, "DEBUG ", log.LstdFlags)

	j, err := jid.Parse(account)
	if err != nil {
		return err
	}

	dialer := &dial.Dialer{
		TLSConfig: &tls.Config{
			ServerName: j.Domain().String(),
			MinVersion: tls.VersionTLS12,
		},
		NoLookup: false,
		NoTLS:    true,
	}

	c := client.New(
		j, logger, debug,
		client.Timeout(30*time.Second),
		client.Dialer(dialer),
		client.NoTLS(true),
		client.Password(getPass),
		client.Quic(useQuic),
	)

	c.Handler(newTestHandler(c))

	if err = c.Online(ctx); err != nil {
		return err
	}

	addClient(c)

	return nil
}

func startMultiConn(num int) {
	var wg sync.WaitGroup

	for i := 0; i < num; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			err := newConnection(ctx)
			if err != nil {
				fmt.Printf("Error starting connection: %v\n", err)
			}
		}()
	}

	wg.Wait()
}
