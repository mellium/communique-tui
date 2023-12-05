package omemo

import (
	"encoding/xml"
	"strings"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stanza"
)

type NodeFetchIQ struct {
	stanza.IQ

	NodeFetch *NodeFetch `xml:"pubsub"`
}

func (iq NodeFetchIQ) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, iq.TokenReader())
}

func (iq NodeFetchIQ) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := iq.WriteXML(e)
	return err
}

func (iq NodeFetchIQ) TokenReader() xml.TokenReader {
	nodeFetchMarshaled, _ := xml.Marshal(iq.NodeFetch)
	var nodeFetchReader xml.TokenReader = xml.NewDecoder(strings.NewReader(
		string(nodeFetchMarshaled),
	))
	return iq.Wrap(nodeFetchReader)
}

// For fetching both devices and key bundles
type NodeFetch struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/pubsub pubsub"`
	Items   *struct {
		XMLName xml.Name `xml:"items"`
		Node    string   `xml:"node,attr"`
		Item    []*struct {
			XMLName xml.Name `xml:"item"`
			ID      string   `xml:"id,attr"`
		} `xml:"item,omitempty"`
	} `xml:"items,omitempty"`
}
