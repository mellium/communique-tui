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
				Device  []Device
			} `xml:"devices,omitempty"`
		} `xml:"item"`
	} `xml:"publish"`
	PublishOptions *PublishOptions
}

type Device struct {
	XMLName xml.Name `xml:"device"`
	ID      string   `xml:"id,attr"`
	Label   string   `xml:"label,attr,omitempty"`
}
