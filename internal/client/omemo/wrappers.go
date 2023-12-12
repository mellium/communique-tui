package omemo

import (
	b64 "encoding/base64"
	"encoding/xml"
	"strings"

	"mellium.im/communique/internal/client"
	omemoreceiver "mellium.im/communique/internal/client/omemo/receiver"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

func WrapDeviceIds(deviceList []Device, c *client.Client) *DeviceAnnouncementIQ {
	iqStanza := &DeviceAnnouncementIQ{
		IQ: stanza.IQ{
			Type: stanza.SetIQ,
			From: c.LocalAddr().Bare(),
		},
		DeviceAnnouncement: &DeviceAnnouncement{
			Publish: &struct {
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
			}{
				Node: "urn:xmpp:omemo:2:devices",
				Item: &struct {
					XMLName xml.Name `xml:"item"`
					ID      string   `xml:"id,attr"`
					Devices *struct {
						XMLName xml.Name `xml:"urn:xmpp:omemo:2 devices"`
						Device  []Device
					} `xml:"devices,omitempty"`
				}{
					ID: "current",
					Devices: &struct {
						XMLName xml.Name `xml:"urn:xmpp:omemo:2 devices"`
						Device  []Device
					}{
						Device: deviceList,
					},
				},
			},
			PublishOptions: &PublishOptions{
				X: &struct {
					XMLName xml.Name `xml:"jabber:x:data x"`
					Type    string   `xml:"type,attr"`
					Field   []*struct {
						Var   string `xml:"var,attr"`
						Type  string `xml:"type,attr,omitempty"`
						Value string `xml:"value"`
					} `xml:"field"`
				}{
					Type: "submit",
					Field: []*struct {
						Var   string `xml:"var,attr"`
						Type  string `xml:"type,attr,omitempty"`
						Value string `xml:"value"`
					}{
						{Var: "FORM_TYPE", Type: "hidden", Value: "http://jabber.org/protocol/pubsub#publish-options"},
						{Var: "pubsub#access_model", Value: "open"},
					},
				},
			},
		},
	}

	return iqStanza
}

func WrapKeyBundle(c *client.Client) *KeyBundleAnnouncementIQ {
	var opks []PreKey

	for _, key := range c.OpkList {
		opks = append(opks, PreKey{ID: key.ID, Text: b64.StdEncoding.EncodeToString(key.PublicKey)})
	}

	iqStanza := &KeyBundleAnnouncementIQ{
		IQ: stanza.IQ{
			Type: stanza.SetIQ,
			From: c.LocalAddr().Bare(),
		},
		KeyBundleAnnouncement: &KeyBundleAnnouncement{
			Publish: &struct {
				XMLName xml.Name `xml:"http://jabber.org/protocol/pubsub publish"`
				Node    string   `xml:"node,attr"`
				Item    *struct {
					Id        string `xml:"id,attr"`
					KeyBundle *KeyBundle
				} `xml:"item"`
			}{
				Node: "urn:xmpp:omemo:2:bundles",
				Item: &struct {
					Id        string `xml:"id,attr"`
					KeyBundle *KeyBundle
				}{
					Id: c.DeviceId,
					KeyBundle: &KeyBundle{
						Spk: &struct {
							ID   string `xml:"id,attr"`
							Text string `xml:",chardata"`
						}{
							ID:   "0",
							Text: b64.StdEncoding.EncodeToString(c.SpkPub),
						},
						Spks: b64.StdEncoding.EncodeToString(c.SpkSig),
						Ik:   b64.StdEncoding.EncodeToString(c.IdPubKey),
						Dhk:  b64.StdEncoding.EncodeToString(c.TmpDhPubKey),
						Prekeys: &struct {
							Pks []PreKey
						}{
							Pks: opks,
						},
					},
				},
			},
			PublishOptions: &PublishOptions{
				X: &struct {
					XMLName xml.Name `xml:"jabber:x:data x"`
					Type    string   `xml:"type,attr"`
					Field   []*struct {
						Var   string `xml:"var,attr"`
						Type  string `xml:"type,attr,omitempty"`
						Value string `xml:"value"`
					} `xml:"field"`
				}{
					Type: "submit",
					Field: []*struct {
						Var   string `xml:"var,attr"`
						Type  string `xml:"type,attr,omitempty"`
						Value string `xml:"value"`
					}{
						{Var: "FORM_TYPE", Type: "hidden", Value: "http://jabber.org/protocol/pubsub#publish-options"},
						{Var: "pubsub#access_model", Value: "open"},
					},
				},
			},
		},
	}

	return iqStanza
}

func WrapNodeFetch(node string, itemId string, targetJid jid.JID, c *client.Client) *NodeFetchIQ {
	iqStanza := &NodeFetchIQ{
		IQ: stanza.IQ{
			Type: stanza.GetIQ,
			From: c.LocalAddr().Bare(),
			To:   targetJid,
		},

		NodeFetch: &NodeFetch{
			Items: &struct {
				XMLName xml.Name `xml:"items"`
				Node    string   `xml:"node,attr"`
				Item    []*struct {
					XMLName xml.Name `xml:"item"`
					ID      string   `xml:"id,attr"`
				} `xml:"item,omitempty"`
			}{
				Node: node,
				Item: []*struct {
					XMLName xml.Name `xml:"item"`
					ID      string   `xml:"id,attr"`
				}{
					{ID: itemId},
				},
			},
		},
	}

	return iqStanza
}

func WrapEnvelope(text string, c *client.Client) *omemoreceiver.Envelope {
	envelope := &omemoreceiver.Envelope{
		Content: &struct {
			Body *struct {
				Text string `xml:",chardata"`
			} `xml:"jabber:client body"`
		}{
			Body: &struct {
				Text string `xml:",chardata"`
			}{
				Text: text,
			},
		},
		Rpad: strings.Repeat("=", 64),
		From: &struct {
			JID string `xml:"jid,attr"`
		}{
			JID: c.LocalAddr().Bare().String(),
		},
	}

	return envelope
}

func WrapEncrypted(targetJid jid.JID, targetDeviceId, keyElement, encryptedEnvelope string, keyExchange bool, c *client.Client) (*EncryptedMessage, stanza.Message) {
	messageStanza := &EncryptedMessage{
		Header: &struct {
			Sid  string `xml:"sid,attr"`
			Keys []struct {
				JID string `xml:"jid,attr"`
				Key *struct {
					Kex  bool   `xml:"kex,attr,omitempty"`
					Rid  string `xml:"rid,attr"`
					Text string `xml:",chardata"`
				} `xml:"key"`
			} `xml:"keys"`
		}{
			Sid: c.DeviceId,
			Keys: []struct {
				JID string `xml:"jid,attr"`
				Key *struct {
					Kex  bool   `xml:"kex,attr,omitempty"`
					Rid  string `xml:"rid,attr"`
					Text string `xml:",chardata"`
				} `xml:"key"`
			}{
				{
					JID: targetJid.String(),
					Key: &struct {
						Kex  bool   `xml:"kex,attr,omitempty"`
						Rid  string `xml:"rid,attr"`
						Text string `xml:",chardata"`
					}{
						Kex:  keyExchange,
						Rid:  targetDeviceId,
						Text: keyElement,
					},
				},
			},
		},
		Payload: encryptedEnvelope,
	}

	stanzaMessage := stanza.Message{
		Type: stanza.ChatMessage,
		From: c.LocalAddr().Bare(),
		To:   targetJid,
	}

	return messageStanza, stanzaMessage
}
