package omemoreceiver

import "encoding/xml"

type PreKey struct {
	ID         string
	PublicKey  []byte
	PrivateKey []byte
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
