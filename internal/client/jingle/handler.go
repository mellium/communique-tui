package jingle

import (
	"encoding/xml"

	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

const NS = "urn:xmpp:jingle:1"

func Handle(h mux.IQHandlerFunc) mux.Option {
	return mux.IQ(stanza.SetIQ, xml.Name{Local: "jingle", Space: NS}, h)
}
