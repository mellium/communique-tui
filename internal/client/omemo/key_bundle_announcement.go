package omemo

import (
	"encoding/xml"
	"strings"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stanza"
)

type KeyBundleAnnouncementIQ struct {
	stanza.IQ

	KeyBundleAnnouncement *KeyBundleAnnouncement `xml:"pubsub"`
}

func (iq KeyBundleAnnouncementIQ) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, iq.TokenReader())
}

func (iq KeyBundleAnnouncementIQ) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := iq.WriteXML(e)
	return err
}

func (iq KeyBundleAnnouncementIQ) TokenReader() xml.TokenReader {
	keyBundleAnnouncementMarshaled, _ := xml.Marshal(iq.KeyBundleAnnouncement)
	var keyBundleAnnouncementReader xml.TokenReader = xml.NewDecoder(strings.NewReader(
		string(keyBundleAnnouncementMarshaled),
	))
	return iq.Wrap(keyBundleAnnouncementReader)
}

type KeyBundleAnnouncement struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/pubsub pubsub"`
	Publish *struct {
		XMLName xml.Name `xml:"http://jabber.org/protocol/pubsub publish"`
		Node    string   `xml:"node,attr"`
		Item    *struct {
			Id        string `xml:"id,attr"`
			KeyBundle *KeyBundle
		} `xml:"item"`
	} `xml:"http://jabber.org/protocol/pubsub publish"`
	PublishOptions *PublishOptions
}

type KeyBundle struct {
	XMLName xml.Name `xml:"urn:xmpp:omemo:2 bundle"`
	Spk     *struct {
		ID   string `xml:"id,attr"`
		Text string `xml:",chardata"`
	} `xml:"spk"`
	Spks    string `xml:"spks"`
	Ik      string `xml:"ik"`
	Dhk     string `xml:"dhk"`
	Prekeys *struct {
		Pks []PreKey
	} `xml:"prekeys"`
}

type PreKey struct {
	XMLName xml.Name `xml:"pk"`
	ID      string   `xml:"id,attr"`
	Text    string   `xml:",chardata"`
}
