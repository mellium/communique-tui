package omemoresponse

import (
	"log"
	"strconv"

	b64 "encoding/base64"

	"google.golang.org/protobuf/proto"
	"mellium.im/communique/internal/client/doubleratchet"
	"mellium.im/communique/internal/client/omemo/protobuf"
	"mellium.im/communique/internal/client/x3dh"
)

func ReceiveKeyAgreement(keyElementB64, payloadB64, peerJid, deviceId string, idPrivKey, spkPriv, dhPrivKey, dhPubKey []byte, opkList []PreKey, messageSession map[string]*doubleratchet.DoubleRatchet, logger *log.Logger) {
	keyElement, err := b64.StdEncoding.DecodeString(keyElementB64)

	if err != nil {
		logger.Printf("Error decoding key element: %s", err)
		return
	}

	keyExchangeMessage := &protobuf.OMEMOKeyExchange{}

	err = proto.Unmarshal(keyElement, keyExchangeMessage)

	if err != nil {
		logger.Printf("Error unmarshaling key element protobuf: %s", err)
		return
	}

	opkId := strconv.FormatUint(uint64(*keyExchangeMessage.PkId), 10)

	peerIdPubKey := keyExchangeMessage.Ik
	ekPub := keyExchangeMessage.Ek
	var opkPriv []byte
	var opkPub []byte

	for _, opk := range opkList {
		if opk.ID == opkId {
			opkPriv = opk.PrivateKey
			opkPub = opk.PublicKey
		}
	}

	logger.Print("CHOSEN OPK")
	logger.Print(opkId)
	logger.Print(keyExchangeMessage.PkId)
	logger.Print("CHOSEN OPK VALUE")
	logger.Print(opkPub)

	if len(opkPriv) == 0 {
		logger.Print("OPK not found.")
		return
	}

	sharedKey, associatedData, err := x3dh.ReceiveInitialMessage(idPrivKey, opkPriv, peerIdPubKey, spkPriv, ekPub)

	if err != nil {
		logger.Printf("Failed performing X3DH: %s", err)
	}

	logger.Print("SHARED KEY")
	logger.Print(sharedKey)

	logger.Print("ASSOCIATED DATA")
	logger.Print(associatedData)

	sess, err := doubleratchet.CreatePassive(sharedKey, associatedData, dhPubKey, dhPrivKey)

	if err != nil {
		logger.Printf("Failed setting up Double Ratchet session: %s", err)
	}

	jdid := peerJid + ":" + deviceId

	messageSession[jdid] = sess

	payload, err := b64.RawStdEncoding.DecodeString(payloadB64)

	logger.Print("CIPHERTEXT")
	logger.Print(payload)

	if err != nil {
		logger.Printf("Error decoding payload: %s", err)
	}

	logger.Print("IDPRIVKEY 2")
	logger.Print(idPrivKey)

	envelope, err := sess.Decrypt(payload)

	if err != nil {
		logger.Printf("Error decrypting payload: %s", err)
	}

	logger.Print(string(envelope[:]))

}
