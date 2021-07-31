// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"io"
	"time"

	"mellium.im/xmpp/dial"
)

// Option is used to configure a client.
type Option func(*Client)

// RosterVer sets the last seen and stored roster version.
func RosterVer(ver string) Option {
	return func(c *Client) {
		c.rosterVer = ver
	}
}

// Timeout sets a timeout for reads and writes from the client.
// If no timeout is provided, the default is 30 seconds.
func Timeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// Dialer sets the dialer used to make the underlying XMPP connections.
//
// If this option is not provided, a default TLS dialer is used with the
// servername set to the domainpart of the JID.
func Dialer(d *dial.Dialer) Option {
	return func(c *Client) {
		if d != nil {
			c.dialer = d
		}
	}
}

// Tee mirrors XML from the XMPP stream to the underlying writers similar to the
// tee(1) command.
//
// If a nil writer is provided for either argument, that stream will not be
// mirrored.
func Tee(in io.Writer, out io.Writer) Option {
	return func(c *Client) {
		if in != nil {
			c.win = in
		}
		if out != nil {
			c.wout = out
		}
	}
}

func emptyPass(context.Context) (string, error) {
	return "", nil
}

// Password is a function that will be used to fetch the password for the
// account when the client connects.
//
// The getPass function will be called every time the password is required,
// including reconnects so you may wish to use a memoized function.
// If the option is not provided or a nil function is used, the password will
// always be an empty string.
func Password(getPass func(context.Context) (string, error)) Option {
	return func(c *Client) {
		if getPass == nil {
			getPass = emptyPass
		}
		c.getPass = getPass
	}
}

func emptyHandler(interface{}) {}

// Handler configures a handler function to be used for events emitted by the
// client.
//
// For a list of events that any handler function may handle, see the event
// package.
func Handler(h func(interface{})) Option {
	return func(c *Client) {
		if h == nil {
			h = emptyHandler
		}
		c.handler = h
	}
}
