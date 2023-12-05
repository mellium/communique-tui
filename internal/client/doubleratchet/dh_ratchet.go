// SPDX-FileCopyrightText: 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

// This file implements the Diffie-Hellman ratchet with a linked root chain.

package doubleratchet

// dhRatchet represents a Diffie-Hellman ratchet including a KDF based on a
// common secret / root key to derive the sending and receiving chain keys.
//
// The integrated root chain prevents a MITM from announcing new DH parameters.
// Thus, at most the communication can be broken, but not taken over. The
// verification takes place in the DECRYPT function, which re-calculates an HMAC
// based on the sending/receiving chain's parameters, derived from the KDF_RK.
type dhRatchet struct {
	rootKey   []byte
	dhPub     []byte
	dhPriv    []byte
	peerDhPub []byte

	isActive      bool
	isInitialized bool
}

// dhRatchetActive creates a DH ratchet for the active peer, Alice.
func dhRatchetActive(rootKey, peerDhPub []byte) (r *dhRatchet, err error) {
	r = &dhRatchet{
		isActive:  true,
		rootKey:   rootKey,
		peerDhPub: peerDhPub,
	}

	r.dhPub, r.dhPriv, err = DhKeyPair()
	return
}

// dhRatchetPassive creates a DH ratchet for the passive peer, Bob.
func dhRatchetPassive(rootKey, dhPub, dhPriv []byte) (r *dhRatchet, err error) {
	r = &dhRatchet{
		isActive: false,
		rootKey:  rootKey,
		dhPub:    dhPub,
		dhPriv:   dhPriv,
	}
	return
}

// step performs a DH ratchet step.
//
// First, the other party's secret will be calculated. Second, a new DH key pair
// will be generated with its subsequent secret.
//
// For the active peer's initial step, peerDhPub might be nil. The previously
// set value will not be overwritten.
func (r *dhRatchet) step(peerDhPub []byte) (dhPub, sendKey, recvKey []byte, err error) {
	// The active peer needs to perform a special initial step exactly once.
	if r.isActive && !r.isInitialized {
		dhPub = r.dhPub

		sendKey, err = dh(r.dhPriv, r.peerDhPub)
		if err != nil {
			return
		}
		r.rootKey, sendKey, err = rootKdf(r.rootKey, sendKey)
		if err != nil {
			return
		}

		r.isInitialized = true
		return
	}

	r.peerDhPub = peerDhPub

	// Close up to the other party's state..
	recvKey, err = dh(r.dhPriv, r.peerDhPub)
	if err != nil {
		return
	}
	r.rootKey, recvKey, err = rootKdf(r.rootKey, recvKey)
	if err != nil {
		return
	}

	// ..and proceed ourselves.
	r.dhPub, r.dhPriv, err = DhKeyPair()
	if err != nil {
		return
	}
	dhPub = r.dhPub

	sendKey, err = dh(r.dhPriv, r.peerDhPub)
	if err != nil {
		return
	}
	r.rootKey, sendKey, err = rootKdf(r.rootKey, sendKey)
	if err != nil {
		return
	}

	return
}
