package omemo

import (
	"encoding/xml"

	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

const PUBSUB_NS = "http://jabber.org/protocol/pubsub"

func PubsubHandle(h mux.IQHandlerFunc) mux.Option {
	return mux.IQ(stanza.SetIQ, xml.Name{Local: "pubsub", Space: PUBSUB_NS}, h)
}
