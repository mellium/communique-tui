// SPDX-FileCopyrightText: 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package x3dh implements a variant of the Extended Triple Diffie-Hellman
// (X3DH) key agreement protocol.
//
// The original X3DH algorithm by Marlinspike and Perrin[0] provides a certain
// amount of leeway. Thus, the following decisions were made. The used curve is
// the Curve25519 resp. X25519 as the ECDH function. SHA-256 is the used hash
// function.
//
// This implementation does not contain support for one-time prekeys. Thus, the
// published signed prekey (SPK) needs to be rotated.
//
// However, a serious difference from the standard is the choice of Ed25519 for
// the identity keys (IK). Originally, X25519 is used for all keys. As a
// drawback, Signal requires its XEdDSA[1] specification to allow signatures
// based on X25519. This implementation has chosen the "other way" and map
// Ed25519 keys to their X25519 equivalent, as described in RFC 7748[2] or this
// nice blog post by Filippo Valsorda[3]. This breaks compatibility!
//
//	[0] https://signal.org/docs/specifications/x3dh/
//	[1] https://signal.org/docs/specifications/xeddsa/
//	[2] https://tools.ietf.org/html/rfc7748#section-4.1
//	[3] https://blog.filippo.io/using-ed25519-keys-for-encryption/
//
// The normal procedure is:
//
//  1. Bob creates a signed prekey (SPK) and publishes it; CreateNewSpk.
//  2. Alice fetches Bob's SPK including the signature and crafts an initial
//     message; CreateInitialMessage.
//  3. Bob receives this message and calculates the same session parameters.
package x3dh

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

// CreateNewSpk creates a new X25519 signed prekey (SPK), both the public and
// private part. The public part is signed by the identity key.
//
// The resulting triple (public IK, public SPK, signed public SPK) should be
// either sent to a peer or published on some keyserver. Based on this data,
// another peer can initiate a session by the CreateInitialMessage function.
func CreateNewSpk(idKey ed25519.PrivateKey) (spkPub, spkPriv, spkSig []byte, err error) {
	spkPriv = make([]byte, curve25519.ScalarSize)
	if _, err = rand.Read(spkPriv); err != nil {
		return
	}

	spkPub, err = curve25519.X25519(spkPriv, curve25519.Basepoint)
	if err != nil {
		return
	}

	spkSig = ed25519.Sign(idKey, spkPub)
	return
}

// kdf derives a session key based on a SHA-256 HKDF from the DH parameters.
//
// Internally the input key material (concatenated DH values) are appended to 32
// zero bytes, as the X3DH specification demands. For the HKDF, 0xff is used as
// the info.
func kdf(km []byte) (out []byte, err error) {
	in := append(bytes.Repeat([]byte{0x00}, 32), km...)
	kdf := hkdf.New(sha256.New, in, bytes.Repeat([]byte{0x00}, sha256.Size), []byte{0xff})

	out = make([]byte, 32)
	if _, err = io.ReadFull(kdf, out); err != nil {
		return
	}

	return
}

// CreateInitialMessage based on the peer's published signed prekey.
//
// This function must be called by the active opening party. Internally an
// ephemeral key (X25519) will be generated and used with the X25519 equivalent
// of the two identity keys to establish an ECDH secret. The associated data are
// the concatenation of the two public keys.
func CreateInitialMessage(
	idKey ed25519.PrivateKey, peerIdKey, opkPub ed25519.PublicKey, spkPub, spkSig []byte,
) (sessKey, associatedData, ekPub []byte, err error) {
	if len(peerIdKey) != ed25519.PublicKeySize {
		err = fmt.Errorf("invalid peer public key size")
		return
	}

	if !ed25519.Verify(peerIdKey, spkPub, spkSig) {
		err = fmt.Errorf("invalid SPK signature")
		return
	}

	ekPriv := make([]byte, curve25519.ScalarSize)
	if _, err = rand.Read(ekPriv); err != nil {
		return
	}
	ekPub, err = curve25519.X25519(ekPriv, curve25519.Basepoint)
	if err != nil {
		return
	}

	idXKey := ed25519PrivateKeyToCurve25519(idKey)
	peerIdXKey := ed25519PublicKeyToCurve25519(peerIdKey)
	opkPubXKey := ed25519PublicKeyToCurve25519(opkPub)

	dhOut := make([]byte, 4*curve25519.ScalarSize)
	dhSteps := [][][]byte{{idXKey, spkPub}, {ekPriv, peerIdXKey}, {ekPriv, spkPub}, {ekPriv, opkPubXKey}}

	for _, dhStep := range dhSteps {
		var dhTmp []byte
		dhTmp, err = curve25519.X25519(dhStep[0], dhStep[1])
		if err != nil {
			return
		}

		dhOut = append(dhOut, dhTmp...)
	}

	sessKey, err = kdf(dhOut)
	if err != nil {
		return
	}

	associatedData = append(idKey.Public().(ed25519.PublicKey), []byte(peerIdKey)...)
	return
}

// ReceiveInitialMessage handles the initial message from the passive party.
//
// Therefore the same calculation is performed as for CreateInitialMessage,
// just in reverse.
func ReceiveInitialMessage(
	idKey ed25519.PrivateKey, opkPriv []byte, peerIdKey ed25519.PublicKey, spkPriv, ekPub []byte,
) (sessKey, associatedData []byte, err error) {
	if len(peerIdKey) != ed25519.PublicKeySize {
		err = fmt.Errorf("invalid peer public key size")
		return
	}

	isOpkValid := len(opkPriv) > 0

	var opkPrivXKey []byte

	idXKey := ed25519PrivateKeyToCurve25519(idKey)
	if isOpkValid {
		opkPrivXKey = ed25519PrivateKeyToCurve25519(opkPriv)
	}

	peerIdXKey := ed25519PublicKeyToCurve25519(peerIdKey)

	dhOut := make([]byte, 4*curve25519.ScalarSize)

	var dhSteps [][][]byte

	if isOpkValid {
		dhSteps = [][][]byte{
			{spkPriv, peerIdXKey},
			{idXKey, ekPub},
			{spkPriv, ekPub},
			{opkPrivXKey, ekPub},
		}
	} else {
		dhSteps = [][][]byte{
			{spkPriv, peerIdXKey},
			{idXKey, ekPub},
			{spkPriv, ekPub},
		}
	}

	for _, dhStep := range dhSteps {
		var dhTmp []byte
		dhTmp, err = curve25519.X25519(dhStep[0], dhStep[1])
		if err != nil {
			return
		}

		dhOut = append(dhOut, dhTmp...)
	}

	sessKey, err = kdf(dhOut)
	if err != nil {
		return
	}

	associatedData = append([]byte(peerIdKey), idKey.Public().(ed25519.PublicKey)...)
	return
}
