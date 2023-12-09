package omemo

import (
	"encoding/xml"
	"strings"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stanza"
)

type EncryptedMessage struct {
	stanza.Message

	Encrypted *Encrypted `xml:"urn:xmpp:omemo:2 encrypted"`
}

func (message EncryptedMessage) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, message.TokenReader())
}

func (message EncryptedMessage) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := message.WriteXML(e)
	return err
}

func (message EncryptedMessage) TokenReader() xml.TokenReader {
	encryptedMessageMarshaled, _ := xml.Marshal(message.Encrypted)
	var encryptedMessageReader xml.TokenReader = xml.NewDecoder(strings.NewReader(
		string(encryptedMessageMarshaled),
	))
	return message.Wrap(encryptedMessageReader)
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
	Store   string `xml:"urn:xmpp:hints store"`
}
