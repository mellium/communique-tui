package omemo

import (
	"encoding/xml"
	"strings"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stanza"
)

type DeviceAnnouncementIQ struct {
	stanza.IQ

	DeviceAnnouncement *DeviceAnnouncement `xml:"pubsub"`
}

func (iq DeviceAnnouncementIQ) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, iq.TokenReader())
}

func (iq DeviceAnnouncementIQ) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := iq.WriteXML(e)
	return err
}

func (iq DeviceAnnouncementIQ) TokenReader() xml.TokenReader {
	deviceAnnouncementMarshaled, _ := xml.Marshal(iq.DeviceAnnouncement)
	var deviceAnnouncementReader xml.TokenReader = xml.NewDecoder(strings.NewReader(
		string(deviceAnnouncementMarshaled),
	))
	return iq.Wrap(deviceAnnouncementReader)
}

type KeyBundleAnnouncement struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/pubsub pubsub"`
	Publish *struct {
		Node string `xml:"node,attr"`
		Item *struct {
			Id        string `xml:"id,attr"`
			KeyBundle *KeyBundle
		}
	}
	PublishOptions *PublishOptions
}

type DeviceAnnouncement struct {
	XMLName xml.Name `xml:"http://jabber.org/protocol/pubsub pubsub"`
	Publish *struct {
		XMLName xml.Name `xml:"publish"`
		Node    string   `xml:"node,attr"`
		Item    *struct {
			XMLName xml.Name `xml:"item"`
			ID      string   `xml:"id,attr"`
			Devices *struct {
				XMLName xml.Name `xml:"urn:xmpp:omemo:2 devices"`
				Device  []*struct {
					XMLName xml.Name `xml:"device"`
					ID      string   `xml:"id,attr"`
					Label   string   `xml:"label,attr,omitempty"`
				} `xml:"device"`
			} `xml:"devices,omitempty"`
		} `xml:"item"`
	} `xml:"publish,omitempty"`
	PublishOptions *PublishOptions
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

type KeyBundle struct {
	XMLName xml.Name `xml:"urn:xmpp:omemo:2 bundle"`
	Spk     *struct {
		ID   string `xml:"id,attr"`
		Text string `xml:",chardata"`
	} `xml:"spk"`
	Spks    string `xml:"spks"`
	Ik      string `xml:"ik"`
	Prekeys struct {
		Pks []*struct {
			ID   string `xml:"id,attr"`
			Text string `xml:",chardata"`
		} `xml:"pk"`
	} `xml:"prekeys"`
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
