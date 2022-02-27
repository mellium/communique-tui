// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package legacybookmarks

import (
	"encoding/xml"
	"strconv"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/bookmarks"
	"mellium.im/xmpp/jid"
)

// channel represents a single chat room.
// It wraps the newer bookmarks.Channel to change the XML output to the legacy
// format.
type channel struct {
	C bookmarks.Channel
}

// TokenReader satisfies the xmlstream.Marshaler interface.
func (c channel) TokenReader() xml.TokenReader {
	var payloads []xml.TokenReader
	if c.C.Nick != "" {
		payloads = append(payloads, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(c.C.Nick)),
			xml.StartElement{
				Name: xml.Name{Local: "nick"},
			},
		))
	}
	if c.C.Password != "" {
		payloads = append(payloads, xmlstream.Wrap(
			xmlstream.Token(xml.CharData(c.C.Password)),
			xml.StartElement{
				Name: xml.Name{Local: "password"},
			},
		))
	}
	conferenceAttrs := []xml.Attr{{
		Name:  xml.Name{Local: "jid"},
		Value: c.C.JID.String(),
	}, {
		Name:  xml.Name{Local: "autojoin"},
		Value: strconv.FormatBool(c.C.Autojoin),
	}}
	if c.C.Name != "" {
		conferenceAttrs = append(conferenceAttrs, xml.Attr{
			Name:  xml.Name{Local: "name"},
			Value: c.C.Name,
		})
	}

	return xmlstream.Wrap(
		xmlstream.MultiReader(payloads...),
		xml.StartElement{
			Name: xml.Name{Local: "conference", Space: NS},
			Attr: conferenceAttrs,
		},
	)
}

// UnmarshalXML implements xml.Unmarshaler.
func (c *channel) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	data := struct {
		XMLName  xml.Name `xml:"storage:bookmarks conference"`
		JID      jid.JID  `xml:"jid,attr"`
		Name     string   `xml:"name,attr"`
		Autojoin bool     `xml:"autojoin,attr"`
		Nick     string   `xml:"nick"`
		Password string   `xml:"password"`
	}{}
	err := d.DecodeElement(&data, &start)
	if err != nil {
		return err
	}

	c.C.Autojoin = data.Autojoin
	c.C.Name = data.Name
	c.C.Nick = data.Nick
	c.C.Password = data.Password
	c.C.JID = data.JID
	return nil
}
