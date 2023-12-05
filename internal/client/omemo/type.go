package omemo

import (
	"encoding/xml"
)

type PublishOptions struct {
	XMLName xml.Name `xml:"publish-options"`
	X       *struct {
		XMLName xml.Name `xml:"jabber:x:data x"`
		Type    string   `xml:"type,attr"`
		Field   []*struct {
			Var   string `xml:"var,attr"`
			Type  string `xml:"type,attr,omitempty"`
			Value string `xml:"value"`
		} `xml:"field"`
	} `xml:"x"`
}

type Envelope struct {
	XMLName xml.Name `xml:"urn:xmpp:sce:1 envelope"`
	Content *struct {
		Body *struct {
			Text string `xml:",chardata"`
		} `xml:"jabber:client body"`
	} `xml:"content"`
	Rpad string `xml:"rpad"`
	From *struct {
		JID string `xml:"jid,attr"`
	} `xml:"from"`
}

type Encrypted struct {
	XMLName xml.Name `xml:"urn:xmpp:omemo:2 encrypted"`
	Header  *struct {
		Sid  string `xml:"sid,attr"`
		Keys []struct {
			JID string `xml:"jid,attr"`
			Key *struct {
				Kex  bool   `xml:"kex,attr,omitempty"`
				Rid  string `xml:"rid,attr"`
				Text string `xml:",chardata"`
			} `xml:"key"`
		} `xml:"keys"`
	} `xml:"header"`
	Payload string `xml:"payload"`
}
