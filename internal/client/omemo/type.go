package omemo

import (
	"encoding/xml"
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
