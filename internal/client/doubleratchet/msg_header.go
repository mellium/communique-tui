// SPDX-FileCopyrightText: 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

package doubleratchet

// This file implements the encrypted message's header, including both
// marshalling and parsing.

import (
	"fmt"
)

// header represents an unencrypted Double Ratchet message header.
//
// A header contains the sender's current DH ratchet public key, the previous
// chain length (PN), and the this message's chain number (N). The Double
// Ratchet Algorithm specification names this as HEADER.
type header struct {
	dhPub  []byte
	prevNo int
	msgNo  int
}

// headerLen is the summed length of an ECDH public key and two uint16s.
const headerLen = 32 + 2 + 2

// marshal this header into bytes.
func (h header) marshal() (data []byte, err error) {
	if h.prevNo >= 1<<16 || h.msgNo >= 1<<16 {
		return nil, fmt.Errorf("header numbers MUST be uint16")
	}

	data = make([]byte, headerLen)
	copy(data[:32], h.dhPub)
	copy(data[32:34], []byte{byte(h.prevNo >> 8), byte(h.prevNo)})
	copy(data[34:36], []byte{byte(h.msgNo >> 8), byte(h.msgNo)})
	return
}

// parseHeader from bytes.
func parseHeader(data []byte) (h header, err error) {
	if len(data) < headerLen {
		err = fmt.Errorf("header MUST be of %d bytes", headerLen)
		return
	}

	h.dhPub = make([]byte, 32)
	copy(h.dhPub[:], data[:32])
	h.prevNo = int(data[32])<<8 | int(data[33])
	h.msgNo = int(data[34])<<8 | int(data[35])
	return
}
