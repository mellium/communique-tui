// SPDX-FileCopyrightText: 2021 Alvar Penning
//
// SPDX-License-Identifier: GPL-3.0-or-later

// This file implements trivial PKCS#7 padding, as needed for the internal
// encryption in Double Ratchet's ENCRYPT/DECRYPT.

package doubleratchet

import (
	"bytes"
	"fmt"
)

// pkcs7Pad adds a PKCS#7 padding based on a given block size, RFC 5652.
func pkcs7Pad(data []byte, blockSize int) (paddedData []byte, err error) {
	if blockSize <= 0 || blockSize > 255 {
		return nil, fmt.Errorf("block size MUST be between 1 and 255")
	}

	padLen := blockSize - (len(data) % blockSize)
	padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
	paddedData = append(data, padding...)
	return
}

// pkcs7Unpad strips a PKCS#7 padding based on a given block size, RFC 5652.
func pkcs7Unpad(paddedData []byte, blockSize int) (data []byte, err error) {
	paddedDataLen := len(paddedData)

	if blockSize <= 0 || blockSize > 255 {
		return nil, fmt.Errorf("block size MUST be between 1 and 255")
	} else if paddedDataLen%blockSize != 0 || paddedDataLen == 0 {
		return nil, fmt.Errorf("padded data is not aligned")
	}

	padLen := int(paddedData[paddedDataLen-1])
	if padLen == 0 || padLen > blockSize {
		return nil, fmt.Errorf("invalid padding length %d", padLen)
	}

	padExpect := bytes.Repeat([]byte{byte(padLen)}, padLen)
	padData := paddedData[paddedDataLen-padLen:]
	if !bytes.Equal(padExpect, padData) {
		return nil, fmt.Errorf("padded data's suffix does not match")
	}

	data = paddedData[:paddedDataLen-padLen]
	return
}
