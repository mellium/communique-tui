// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package privatexml implements the legacy private XML storage system.
//
// It can be used to store and retrieve snippets of XML on the server. New uses
// of this package should likely use PEP instead.
package privatexml

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/stanza"
)

// NS is the namespace used by this package.
const NS = `jabber:iq:private`

// Set stores the XML copied from r on the server for later retrieval.
func Set(ctx context.Context, s *xmpp.Session, r xml.TokenReader) error {
	return SetIQ(ctx, stanza.IQ{}, s, r)
}

// SetIQ is like Set except that the IQ stanza can be customized.
// Changing the type of the stanza has no effect.
func SetIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session, r xml.TokenReader) error {
	iq.Type = stanza.SetIQ
	return s.UnmarshalIQElement(ctx, xmlstream.Wrap(
		r,
		xml.StartElement{Name: xml.Name{Space: NS, Local: "query"}},
	), iq, nil)
}

// Get requests XML that was previously stored on the server.
func Get(ctx context.Context, s *xmpp.Session, name xml.Name) (xmlstream.TokenReadCloser, error) {
	return GetIQ(ctx, stanza.IQ{}, s, name)
}

type readCloser struct {
	TokenReader xml.TokenReader
	Closer      io.Closer
	closed      bool
}

func (r *readCloser) Token() (xml.Token, error) {
	tok, err := r.TokenReader.Token()
	// Close early if we finish reading the stream.
	if err == io.EOF {
		e := r.Closer.Close()
		if e != nil {
			return tok, e
		}
	}
	return tok, err
}

func (r *readCloser) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	return r.Closer.Close()
}

// GetIQ is like Get except that the IQ stanza can be customized.
// Changing the type of the stanza has no effect.
func GetIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session, name xml.Name) (xmlstream.TokenReadCloser, error) {
	iq.Type = stanza.GetIQ
	resp, err := s.SendIQElement(ctx, xmlstream.Wrap(
		xmlstream.Wrap(
			nil,
			xml.StartElement{Name: name},
		),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "query"}},
	), iq)
	if err != nil {
		return nil, err
	}
	tok, err := resp.Token()
	start, ok := tok.(xml.StartElement)
	if !ok {
		/* #nosec */
		resp.Close()
		return nil, fmt.Errorf("privatexml: expected IQ start token, got %T %[1]v", tok)
	}
	_, err = stanza.UnmarshalIQError(resp, start)
	if err != nil {
		/* #nosec */
		resp.Close()
		return nil, err
	}

	tok, err = resp.Token()
	start, ok = tok.(xml.StartElement)
	if !ok || start.Name.Space != NS || start.Name.Local != "query" {
		/* #nosec */
		resp.Close()
		return nil, fmt.Errorf("privatexml: expected query payload, got %T %[1]v", tok)
	}

	return &readCloser{
		TokenReader: xmlstream.Inner(resp),
		Closer:      resp,
	}, nil
}
