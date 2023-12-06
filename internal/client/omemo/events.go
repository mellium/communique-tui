package omemo

import (
	"context"
	b64 "encoding/base64"
	"encoding/xml"
	"log"
	"math/rand"
	"time"

	"mellium.im/communique/internal/client"
	"mellium.im/communique/internal/client/x3dh"
	"mellium.im/xmpp/jid"
)

func SetupClient(c *client.Client, logger *log.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	keyBundleAnnouncementStanza := WrapKeyBundle(c)

	err := c.UnmarshalIQ(ctx, keyBundleAnnouncementStanza.TokenReader(), nil)

	if err != nil {
		logger.Printf("Error sending key bundle: %q", err)
	}

	// ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	// defer cancel()

	// deviceAnnouncementStanza := WrapDeviceIds([]Device{
	// 	{ID: "1", Label: "Acer Aspire 3"},
	// }, c)

	// _, err = c.SendIQ(ctx, deviceAnnouncementStanza.TokenReader())

	// if err != nil {
	// 	logger.Printf("Error sending device list: %q", err)
	// }

}

func InitiateKeyAgreement(c *client.Client, logger *log.Logger, targetJID jid.JID) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	logger.Printf("Fetching key bundle for " + targetJID.String() + " ...")
	fetchBundleStanza := WrapNodeFetch("urn:xmpp:omemo:2:bundles", c.DeviceId, targetJID, c)

	payload, err := c.SendIQ(ctx, fetchBundleStanza.TokenReader())

	if err != nil {
		logger.Printf("Error fetching key bundle: %v", err)
	}

	logger.Printf("Decoding key bundle for " + targetJID.String() + " ...")

	defer func() {
		payload := payload
		if err != nil {
			payload.Close()
		}
	}()

	var currentEl string
	var targetSpkPub, targetSpkSig, targetIdKeyPub []byte
	var opkList []PreKey
	var opkId string

	for t, _ := payload.Token(); t != nil; t, _ = payload.Token() {
		switch se := t.(type) {
		case xml.StartElement:
			currentEl = se.Name.Local
			if se.Name.Local == "pk" {
				opkId = se.Attr[0].Value
			}
		case xml.CharData:
			content := string(se[:])
			switch currentEl {
			case "spk":
				targetSpkPub, _ = b64.StdEncoding.DecodeString(content)
			case "spks":
				targetSpkSig, _ = b64.StdEncoding.DecodeString(content)
			case "ik":
				targetIdKeyPub, _ = b64.StdEncoding.DecodeString(content)
			case "pk":
				opkList = append(opkList, PreKey{ID: opkId, Text: content})
			}
		}
	}

	randomIndex := rand.Intn(len(opkList))
	opk := opkList[randomIndex]
	opkPub, _ := b64.StdEncoding.DecodeString(opk.Text)

	sharedKey, associatedData, ekPub, err := x3dh.CreateInitialMessage(c.IdPrivKey, targetIdKeyPub, opkPub, targetSpkPub, targetSpkSig)
	logger.Print(sharedKey)
	logger.Print(associatedData)
	logger.Print(ekPub)
}
