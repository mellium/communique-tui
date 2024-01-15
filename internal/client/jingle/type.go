package jingle

import (
	"encoding/xml"
	"strings"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stanza"
)

type IQ struct {
	stanza.IQ

	Jingle *Jingle `xml:"urn:xmpp:jingle:1 jingle"`
}

func (iq IQ) WriteXML(w xmlstream.TokenWriter) (int, error) {
	return xmlstream.Copy(w, iq.TokenReader())
}

func (iq IQ) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	_, err := iq.WriteXML(e)
	return err
}

func (iq IQ) TokenReader() xml.TokenReader {
	jingleMarshaled, _ := xml.Marshal(iq.Jingle)
	var jingleReader xml.TokenReader = xml.NewDecoder(strings.NewReader(
		string(jingleMarshaled),
	))
	return iq.Wrap(jingleReader)
}

type Jingle struct {
	XMLName   xml.Name `xml:"urn:xmpp:jingle:1 jingle"`
	Action    string   `xml:"action,attr"`
	Initiator string   `xml:"initiator,attr,omitempty"`
	Responder string   `xml:"responder,attr,omitempty"`
	SID       string   `xml:"sid,attr"`
	Group     *struct {
		Semantics string `xml:"semantics,attr,omitempty"`
		Contents  []struct {
			Name string `xml:"name,attr,omitempty"`
		} `xml:"content,omitempty"`
	} `xml:"urn:xmpp:jingle:apps:grouping:0 group,omitempty"`
	Contents []*Content `xml:"content,omitempty"`
	Reason   *struct {
		Condition *struct {
			XMLName xml.Name `xml:",omitempty"`
			Details string   `xml:",chardata"`
		}
	} `xml:"reason,omitempty"`
}

func JingleFailed(sid string) *Jingle {
	return &Jingle{
		Action: "session-terminate",
		SID:    sid,
		Reason: &struct {
			Condition *struct {
				XMLName xml.Name "xml:\",omitempty\""
				Details string   "xml:\",chardata\""
			}
		}{
			Condition: &struct {
				XMLName xml.Name "xml:\",omitempty\""
				Details string   "xml:\",chardata\""
			}{
				XMLName: xml.Name{Local: "failed-application"},
			},
		},
	}
}

type Content struct {
	XMLName     xml.Name         `xml:"content,omitempty"`
	Creator     string           `xml:"creator,attr,omitempty"`
	Disposition string           `xml:"disposition,attr,omitempty"`
	Name        string           `xml:"name,attr,omitempty"`
	Senders     string           `xml:"senders,attr,omitempty"`
	Description *RTPDescription  `xml:"urn:xmpp:jingle:apps:rtp:1 description,omitempty"`
	Transport   *ICEUDPTransport `xml:"urn:xmpp:jingle:transports:ice-udp:1 transport,omitempty"`
}

type RTPDescription struct {
	XMLName      xml.Name       `xml:"urn:xmpp:jingle:apps:rtp:1 description,omitempty"`
	Media        string         `xml:"media,attr,omitempty"`
	SSRC         string         `xml:"ssrc,attr,omitempty"`
	PayloadTypes []*PayloadType `xml:"payload-type,omitempty"`
	Source       *struct {
		SSRC       string `xml:"ssrc,attr,omitempty"`
		Parameters []struct {
			Name  string `xml:"name,attr,omitempty"`
			Value string `xml:"value,attr,omitempty"`
		} `xml:"parameter,omitempty"`
	} `xml:"urn:xmpp:jingle:apps:rtp:ssma:0 source,omitempty"`
}

type PayloadType struct {
	XMLName   xml.Name `xml:"payload-type,omitempty"`
	Id        string   `xml:"id,attr,omitempty"`
	Name      string   `xml:"name,attr,omitempty"`
	ClockRate string   `xml:"clockrate,attr,omitempty"`
	Channels  string   `xml:"channels,attr,omitempty"`
	MaxPTime  string   `xml:"maxptime,attr,omitempty"`
	PTime     string   `xml:"ptime,attr,omitempty"`
	Parameter []*struct {
		Name  string `xml:"name,attr,omitempty"`
		Value string `xml:"value,attr,omitempty"`
	} `xml:"parameter,omitempty"`
	RTCPFB []*struct {
		Type    string `xml:"type,attr,omitempty"`
		SubType string `xml:"subtype,attr,omitempty"`
	} `xml:"rtcp-fb,omitempty"`
}

type ICEUDPTransport struct {
	XMLName     xml.Name        `xml:"urn:xmpp:jingle:transports:ice-udp:1 transport,omitempty"`
	PWD         string          `xml:"pwd,attr,omitempty"`
	UFrag       string          `xml:"ufrag,attr,omitempty"`
	FingerPrint *FingerPrint    `xml:"urn:xmpp:jingle:apps:dtls:0 fingerprint,omitempty"`
	Candidates  []*ICECandidate `xml:"candidate,omitempty"`
}

type FingerPrint struct {
	XMLName xml.Name `xml:"urn:xmpp:jingle:apps:dtls:0 fingerprint,omitempty"`
	Hash    string   `xml:"hash,attr,omitempty"`
	Setup   string   `xml:"setup,attr,omitempty"`
	Text    string   `xml:",chardata"`
}

type ICECandidate struct {
	XMLName    xml.Name `xml:"candidate,omitempty"`
	Component  string   `xml:"component,attr,omitempty"`
	Foundation string   `xml:"foundation,attr,omitempty"`
	Ip         string   `xml:"ip,attr,omitempty"`
	Port       string   `xml:"port,attr,omitempty"`
	Priority   string   `xml:"priority,attr,omitempty"`
	Protocol   string   `xml:"protocol,attr,omitempty"`
	Type       string   `xml:"type,attr,omitempty"`
	RelAddr    string   `xml:"rel-addr,attr,omitempty"`
	RelPort    string   `xml:"rel-port,attr,omitempty"`
}

func (candidate *ICECandidate) toSDP() string {
	candidateData := []string{
		candidate.Component,
		candidate.Protocol,
		candidate.Priority,
		candidate.Ip,
		candidate.Port,
		"typ",
		candidate.Type,
	}
	if candidate.Type == "srflx" || candidate.Type == "relay" {
		candidateData = append(candidateData, []string{
			"raddr",
			candidate.RelAddr,
			"rport",
			candidate.RelPort,
		}...)
	}
	candidateVal := &attributeField{
		name:      "candidate",
		value:     candidate.Foundation,
		extension: strings.Join(candidateData, " "),
	}
	return candidateVal.toString()
}
