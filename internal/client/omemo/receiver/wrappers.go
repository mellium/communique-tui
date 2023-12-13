package omemoreceiver

import (
	b64 "encoding/base64"
	"encoding/xml"

	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

func WrapKeyBundle(deviceId, fromJid string, idPubKey, spkPub, spkSig, tmpDhPubKey []byte, opkList []PreKey) *KeyBundleAnnouncementIQ {
	var opks []OPreKey

	for _, key := range opkList {
		opks = append(opks, OPreKey{ID: key.ID, Text: b64.StdEncoding.EncodeToString(key.PublicKey)})
	}

	iqStanza := &KeyBundleAnnouncementIQ{
		IQ: stanza.IQ{
			Type: stanza.SetIQ,
			From: jid.MustParse(fromJid),
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
					Id: deviceId,
					KeyBundle: &KeyBundle{
						Spk: &struct {
							ID   string `xml:"id,attr"`
							Text string `xml:",chardata"`
						}{
							ID:   "0",
							Text: b64.StdEncoding.EncodeToString(spkPub),
						},
						Spks: b64.StdEncoding.EncodeToString(spkSig),
						Ik:   b64.StdEncoding.EncodeToString(idPubKey),
						Dhk:  b64.StdEncoding.EncodeToString(tmpDhPubKey),
						Prekeys: &struct {
							Pks []OPreKey
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
