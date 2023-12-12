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
