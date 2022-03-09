// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package legacybookmarks implements private XML based bookmarks.
package legacybookmarks

import (
	"context"
	"encoding/xml"
	"fmt"

	"mellium.im/communique/internal/privatexml"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/bookmarks"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// NS is the namespace used by this package.
const NS = `storage:bookmarks`

// Fetch returns an iterator over the list of bookmarks.
// The session may block until the iterator is closed.
func Fetch(ctx context.Context, s *xmpp.Session) *Iter {
	return FetchIQ(ctx, stanza.IQ{}, s)
}

// FetchIQ is like Fetch but it allows you to customize the IQ.
// Changing the type of the provided IQ has no effect.
func FetchIQ(ctx context.Context, iq stanza.IQ, s *xmpp.Session) *Iter {
	r, err := privatexml.GetIQ(ctx, iq, s, xml.Name{Space: NS, Local: "storage"})
	if err != nil {
		return &Iter{err: err}
	}
	tok, err := r.Token()
	if err != nil {
		/* #nosec */
		r.Close()
		return &Iter{err: err}
	}
	_, ok := tok.(xml.StartElement)
	if !ok {
		/* #nosec */
		r.Close()
		return &Iter{err: fmt.Errorf("legacybookmarks: expected start token, got %T %[1]v", tok)}
	}

	return &Iter{iter: xmlstream.NewIter(r)}
}

// Delete removes the provided bookmark if it exists.
// If the bookmark does not exist, no error is returned.
func Delete(ctx context.Context, s *xmpp.Session, j jid.JID) error {
	var c []xml.TokenReader
	iter := Fetch(ctx, s)
	var found bool
	for iter.Next() {
		cur := iter.Bookmark()
		if j.Equal(cur.JID) {
			found = true
			continue
		}
		c = append(c, channel{C: cur}.TokenReader())
	}
	err := iter.Err()
	if err != nil {
		return err
	}
	err = iter.Close()
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	return privatexml.Set(ctx, s, xmlstream.Wrap(
		xmlstream.MultiReader(c...),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "storage"}},
	))
}

// Set adds or updates a bookmark.
// Due to the nature of the legacy boomkarks spec, Set must first fetch the
// bookmarks then re-upload the entire list, making it very inefficient.
// There is also greater potential for race conditions if multiple cilents try
// to upload different bookmark lists at once.
func Set(ctx context.Context, s *xmpp.Session, b bookmarks.Channel) error {
	var c []xml.TokenReader
	iter := Fetch(ctx, s)
	for iter.Next() {
		cur := iter.Bookmark()
		if b.JID.Equal(cur.JID) {
			continue
		}
		c = append(c, channel{C: cur}.TokenReader())
	}
	err := iter.Err()
	if err != nil {
		return err
	}
	err = iter.Close()
	if err != nil {
		return err
	}
	c = append(c, channel{C: b}.TokenReader())

	return privatexml.Set(ctx, s, xmlstream.Wrap(
		xmlstream.MultiReader(c...),
		xml.StartElement{Name: xml.Name{Space: NS, Local: "storage"}},
	))
}
