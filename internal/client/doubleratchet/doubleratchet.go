// SPDX-FileCopyrightText: 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

// Package doubleratchet implements a variant of the Double Ratchet Algorithm.
//
// The Double Ratchet Algorithm by Perrin and Marlinspike[0] is to be used by
// two parties to exchange encrypted messages based on a shared secret. Thus, a
// key agreement protocol needs to be used first, e.g., X3DH as also implemented
// in this repository.
//
// The following implementation decisions were made; always following the
// recommendations within the specification.
//
// For the Diffie-Hellman key exchange, X25519 based on Curve25519 is used.
//
// The key derivation functions, both for the root key chain as well as the
// sending and receiving chains, are based on SHA-256. This hash algorithm was
// chosen over SHA-512 to save 32 bytes. The root key's KDF is an HKDF and the
// other KDF uses an HMAC, as recommended by the specification.
//
// For AEAC encryption, AES-256 is used in CBC mode. To fit the AES block size,
// the input is padded with PKCS#7. Finally, an HMAC is attached.
//
// This implementation does not perform the optional header encryption. Each
// message header contains the sender's current DH public key next to its
// position within the current sending chain and the previous chain length.
// This data is not confidential.
//
// A buffer to allow skipping lost or out-of-order messages is implemented to
// support up to eight previous chains, with 32 skipped messages in each. This
// is a compromise between a very lossy link and the possibility for an attacker
// to reserve lots of memory on the victim's machine.
//
//	[0] https://signal.org/docs/specifications/doubleratchet/
package doubleratchet

import (
	"crypto/subtle"
	"fmt"
)

// DoubleRatchet implements the Double Ratchet Algorithm.
//
// This allows sending and receiving PFS encrypted messages. It is also possible
// to skipp a certain amount of messages or decrypt them out of order.
//
// For the Double Ratchet's initialization, the roles of an active and passive
// part (Alice and Bob) needs to be assigned the used application protocol.
// After the first two messages - starting with Alice, bob follows - the order
// of encryption and decryption calls is no longer relevant. However, Alice MUST
// send her first message to Bob in order to synchronize the internal state.
//
// The implementation design and chosen algorithms were documented in this
// package's documentation.
type DoubleRatchet struct {
	associatedData []byte

	dhr *dhRatchet

	peerDhPub    []byte
	chainKeySend []byte
	chainKeyRecv []byte

	sendNo     int
	recvNo     int
	prevSendNo int

	msgKeyBuffer *keyBuffer
}

// CreateActive creates a Double Ratchet for the active part, Alice.
//
// The active peer MUST send the first message.
func CreateActive(sessKey, associatedData, peerDhPub []byte) (dr *DoubleRatchet, err error) {
	dhr, err := dhRatchetActive(sessKey, peerDhPub)
	if err != nil {
		return
	}

	dr = &DoubleRatchet{
		associatedData: associatedData,
		dhr:            dhr,
		peerDhPub:      peerDhPub,
		msgKeyBuffer:   newKeyBuffer(),
	}
	return
}

// CreatePassive creates a Double Ratchet for the passive part, Bob.
//
// The passive peer MUST receive a message first.
func CreatePassive(sessKey, associatedData, dhPub, dhPriv []byte) (dr *DoubleRatchet, err error) {
	dhr, err := dhRatchetPassive(sessKey, dhPub, dhPriv)
	if err != nil {
		return
	}

	dr = &DoubleRatchet{
		associatedData: associatedData,
		dhr:            dhr,
		msgKeyBuffer:   newKeyBuffer(),
	}
	return
}

// dhStep performs a Diffie-Hellman ratchet step.
//
// This is performed automatically if the other party's DH ratchet has proceeded
// or for the active part's initial encrypted message.
func (dr *DoubleRatchet) dhStep() (err error) {
	dr.prevSendNo = dr.sendNo
	dr.sendNo = 0
	dr.recvNo = 0

	_, dr.chainKeySend, dr.chainKeyRecv, err = dr.dhr.step(dr.peerDhPub)
	return
}

// Encrypt a plaintext message for the other party.
//
// The resulting ciphertext will include the necessary header.
func (dr *DoubleRatchet) Encrypt(plaintext []byte) (ciphertext, authKey []byte, err error) {
	if dr.chainKeySend == nil {
		err = dr.dhStep()
		if err != nil {
			return
		}
	}

	var msgKey []byte
	dr.chainKeySend, msgKey, err = chainKdf(dr.chainKeySend)
	if err != nil {
		return
	}

	h := header{
		dhPub:  dr.dhr.dhPub,
		prevNo: dr.prevSendNo,
		msgNo:  dr.sendNo,
	}
	dr.sendNo++

	hData, err := h.marshal()
	if err != nil {
		return
	}

	ciphertext, authKey, err = encrypt(msgKey, plaintext, dr.associatedData)
	if err != nil {
		return
	}

	ciphertext = append(hData, ciphertext...)
	return
}

// skipMsgKeys caches future message keys in the current receiving chain.
//
// This might be necessary if received messages are either lost or out of order.
func (dr *DoubleRatchet) skipMsgKeys(until int) (err error) {
	if dr.recvNo+maxSkipElements < until {
		return fmt.Errorf("cannot skip until %d, maximum is %d", until, dr.recvNo+maxSkipElements)
	}

	// Cannot skip messages without an existing receiving chain. This happens in
	// an initial state before the first complete key exchange.
	if dr.chainKeyRecv == nil {
		return
	}

	for ; dr.recvNo < until; dr.recvNo++ {
		var msgKey []byte
		dr.chainKeyRecv, msgKey, err = chainKdf(dr.chainKeyRecv)
		if err != nil {
			return
		}

		dr.msgKeyBuffer.insert(dr.peerDhPub, dr.recvNo, msgKey)
	}

	return
}

// Decrypt a ciphertext from the other party.
//
// The encryption is an AEAD encryption. Thus, a changed message should be
// detected and result in an error.
func (dr *DoubleRatchet) Decrypt(ciphertext []byte) (plaintext []byte, err error) {
	if len(ciphertext) <= headerLen {
		return nil, fmt.Errorf("ciphertext is too short")
	}

	h, err := parseHeader(ciphertext[:headerLen])
	if err != nil {
		return
	}

	if subtle.ConstantTimeCompare(h.dhPub, dr.peerDhPub) != 1 {
		err = dr.skipMsgKeys(h.prevNo)
		if err != nil {
			return
		}

		dr.peerDhPub = h.dhPub

		err = dr.dhStep()
		if err != nil {
			return
		}
	}

	var msgKey []byte
	switch {
	case h.msgNo < dr.recvNo:
		msgKey, err = dr.msgKeyBuffer.find(h.dhPub, h.msgNo)
		if err != nil {
			return
		}

	case h.msgNo > dr.recvNo:
		err = dr.skipMsgKeys(h.msgNo)
		if err != nil {
			return
		}
		fallthrough

	case h.msgNo == dr.recvNo:
		dr.chainKeyRecv, msgKey, err = chainKdf(dr.chainKeyRecv)
		if err != nil {
			return
		}
		dr.recvNo++
	}

	plaintext, err = decrypt(msgKey, ciphertext[headerLen:], dr.associatedData)
	return
}
