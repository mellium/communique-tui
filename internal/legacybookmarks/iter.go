// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package legacybookmarks

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/bookmarks"
	"mellium.im/xmpp/jid"
)

// Iter is an iterator over bookmarks.
type Iter struct {
	iter    *xmlstream.Iter
	current bookmarks.Channel
	err     error
}

// Next returns true if there are more items to decode.
func (i *Iter) Next() bool {
	if i.err != nil || !i.iter.Next() {
		return false
	}
	start, r := i.iter.Current()
	// If we encounter a lone token that doesn't begin with a start element (eg.
	// a comment) skip it. This should never happen with XMPP, but we don't want
	// to panic in case this somehow happens so just skip it.
	// Similarly, if we encounter a payload type we don't recognize, skip it (this
	// will likely happen as we don't support url bookmarks, so we'll skip those).
	if start == nil || start.Name.Local != "conference" {
		return i.Next()
	}
	var bookmarkJID jid.JID
	// We're decoding the old bookmarks struct into the new one since they're
	// similar. Instead of duplicating the struct, just change the namespace so it
	// decodes correctly.
	transformer := xmlstream.Map(func(t xml.Token) xml.Token {
		switch tok := t.(type) {
		case xml.StartElement:
			if tok.Name.Local == "conference" && tok.Name.Space == NS {
				tok.Name.Space = bookmarks.NS
				// Legacy bookmarks store JID as an attribute, unlike PEP Native Bookmarks
				// which store it as the ID of the PEP item.
				for _, attr := range start.Attr {
					if attr.Name.Local == "jid" {
						j, err := jid.Parse(attr.Value)
						if err != nil {
							i.err = err
							return false
						}
						bookmarkJID = j
						break
					}
				}
			}
			return tok
		case xml.EndElement:
			if tok.Name.Local == "conference" && tok.Name.Space == NS {
				tok.Name.Space = bookmarks.NS
			}
			return tok
		}
		return t
	})(xmlstream.MultiReader(xmlstream.Token(*start), r))
	d := xml.NewTokenDecoder(transformer)
	bookmark := bookmarks.Channel{}
	i.err = d.Decode(&bookmark)
	if i.err != nil {
		return false
	}
	i.current = bookmark
	i.current.JID = bookmarkJID
	return true
}

// Err returns the last error encountered by the iterator (if any).
func (i *Iter) Err() error {
	if i.err != nil {
		return i.err
	}

	return i.iter.Err()
}

// Bookmark returns the last bookmark parsed by the iterator.
func (i *Iter) Bookmark() bookmarks.Channel {
	return i.current
}

// Close indicates that we are finished with the given iterator and processing
// the stream may continue.
// Calling it multiple times has no effect.
func (i *Iter) Close() error {
	if i.iter == nil {
		return nil
	}
	return i.iter.Close()
}
