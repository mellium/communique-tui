package omemo

import (
	"encoding/xml"
	"strings"
)

type EncryptedMessage struct {
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
	Store   string `xml:"urn:xmpp:hints store"`
}

func (message EncryptedMessage) TokenReader() xml.TokenReader {
	encryptedMessageMarshaled, _ := xml.Marshal(message)
	var encryptedMessageReader xml.TokenReader = xml.NewDecoder(strings.NewReader(
		string(encryptedMessageMarshaled),
	))
	return encryptedMessageReader
}
