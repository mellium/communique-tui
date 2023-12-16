package omemo

import (
	"encoding/xml"
	"strings"
)

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

type ReceiptRequest struct {
	XMLName xml.Name `xml:"urn:xmpp:receipts request"`
}

func (rr *ReceiptRequest) TokenReader() xml.TokenReader {
	xmlData, err := xml.Marshal(rr)
	if err != nil {
		panic(err)
	}

	reader := strings.NewReader(string(xmlData))

	decoder := xml.NewDecoder(reader)

	return decoder
}
