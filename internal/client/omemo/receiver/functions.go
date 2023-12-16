package omemoreceiver

import (
	"fmt"
	"log"
	"strconv"

	b64 "encoding/base64"
	"encoding/xml"

	"google.golang.org/protobuf/proto"
	"mellium.im/communique/internal/client/doubleratchet"
	"mellium.im/communique/internal/client/omemo/protobuf"
	"mellium.im/communique/internal/client/x3dh"
)

func ReceiveKeyAgreement(keyElementB64, payloadB64, peerJid, peerDeviceId string, idPrivKey, spkPriv, dhPrivKey, dhPubKey []byte, opkList []PreKey, messageSession map[string]*doubleratchet.DoubleRatchet, logger *log.Logger) (string, string) {
	keyElement, err := b64.StdEncoding.DecodeString(keyElementB64)

	if err != nil {
		logger.Printf("Error decoding key element: %s", err)
	}

	keyExchangeMessage := &protobuf.OMEMOKeyExchange{}

	err = proto.Unmarshal(keyElement, keyExchangeMessage)

	if err != nil {
		logger.Printf("Error unmarshaling key element protobuf: %s", err)
	}

	opkId := strconv.FormatUint(uint64(*keyExchangeMessage.PkId), 10)

	peerIdPubKey := keyExchangeMessage.Ik
	ekPub := keyExchangeMessage.Ek
	var opkPriv []byte

	for _, opk := range opkList {
		if opk.ID == opkId {
			opkPriv = opk.PrivateKey
		}
	}

	if len(opkPriv) == 0 {
		logger.Print("OPK not found.")
	}

	fmt.Print()
	sharedKey, associatedData, err := x3dh.ReceiveInitialMessage(idPrivKey, opkPriv, peerIdPubKey, spkPriv, ekPub)

	if err != nil {
		logger.Printf("Failed performing X3DH: %s", err)
	}

	sess, err := doubleratchet.CreatePassive(sharedKey, associatedData, dhPubKey, dhPrivKey)

	if err != nil {
		logger.Printf("Failed setting up Double Ratchet session: %s", err)
	}

	jdid := peerJid + ":" + peerDeviceId

	messageSession[jdid] = sess

	payload, err := b64.RawStdEncoding.DecodeString(payloadB64)

	if err != nil {
		logger.Printf("Error decoding payload: %s", err)
	}

	envelope, err := sess.Decrypt(payload)

	if err != nil {
		logger.Printf("Error decrypting payload: %s", err)
	}

	return ParseEnvelope(string(envelope[:]), logger), opkId
}

func PublishKeyBundle(deviceId, fromJid string, idPubKey, spkPub, spkSig, tmpDhPubKey []byte, opkList []PreKey, logger *log.Logger) xml.TokenReader {
	keyBundleAnnouncementStanza := WrapKeyBundle(deviceId, fromJid, idPubKey, spkPub, spkSig, tmpDhPubKey, opkList)

	return keyBundleAnnouncementStanza.TokenReader()
}

func ReceiveEncryptedMessage(payloadB64, peerJid, peerDeviceId string, messageSession map[string]*doubleratchet.DoubleRatchet, logger *log.Logger) (string, error) {
	jdid := peerJid + ":" + peerDeviceId

	if _, prs := messageSession[jdid]; !prs {
		return "", fmt.Errorf("%s has no prior session with this user/device, and provides no key exchange", jdid)
	}

	sess := messageSession[jdid]

	payload, err := b64.RawStdEncoding.DecodeString(payloadB64)

	if err != nil {
		logger.Printf("Error decoding payload: %s", err)
	}

	envelope, err := sess.Decrypt(payload)

	if err != nil {
		logger.Printf("Error decrypting payload: %s", err)
	}

	return ParseEnvelope(string(envelope[:]), logger), nil
}

func ParseEnvelope(envelopeB64 string, logger *log.Logger) string {
	rawEnvelope, err := b64.RawStdEncoding.DecodeString(envelopeB64)

	if err != nil {
		logger.Printf("Error decoding envelope: %s", err)
	}

	var envelope Envelope
	ret := ""

	err = xml.Unmarshal([]byte(rawEnvelope), &envelope)
	if err != nil {
		// fmt.Printf("Error parsing envelope XML: %s", err)
	} else {
		ret = envelope.Content.Body.Text
	}

	return ret
}
