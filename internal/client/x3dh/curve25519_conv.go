// SPDX-FileCopyrightText: 2019 Google LLC
// SPDX-FileCopyrightText: 2019-2020 Filippo Valsorda
//
// SPDX-License-Identifier: BSD-3-Clause

// This file contains two functions to convert Ed25519 public and private keys
// to their X25519 pedant. Those functions were extracted from Filippo
// Valsorda's age tool[0]. A more general explanation is available as a blog
// post[1]. Thanks a lot!
//
// [0] https://github.com/FiloSottile/age/blob/v1.0.0-beta5/agessh/agessh.go
// [1] https://blog.filippo.io/using-ed25519-keys-for-encryption/

package x3dh

import (
	"crypto/ed25519"
	"crypto/sha512"
	"math/big"

	"golang.org/x/crypto/curve25519"
)

func ed25519PrivateKeyToCurve25519(pk ed25519.PrivateKey) []byte {
	h := sha512.New()
	_, _ = h.Write(pk.Seed())
	out := h.Sum(nil)
	return out[:curve25519.ScalarSize]
}

var curve25519P, _ = new(big.Int).SetString("57896044618658097711785492504343953926634992332820282019728792003956564819949", 10)

func ed25519PublicKeyToCurve25519(pk ed25519.PublicKey) []byte {
	// ed25519.PublicKey is a little endian representation of the y-coordinate,
	// with the most significant bit set based on the sign of the x-coordinate.
	bigEndianY := make([]byte, ed25519.PublicKeySize)
	for i, b := range pk {
		bigEndianY[ed25519.PublicKeySize-i-1] = b
	}
	bigEndianY[0] &= 0b0111_1111

	// The Montgomery u-coordinate is derived through the bilinear map
	//
	//     u = (1 + y) / (1 - y)
	//
	// See https://blog.filippo.io/using-ed25519-keys-for-encryption.
	y := new(big.Int).SetBytes(bigEndianY)
	denom := big.NewInt(1)
	denom.ModInverse(denom.Sub(denom, y), curve25519P) // 1 / (1 - y)
	u := y.Mul(y.Add(y, big.NewInt(1)), denom)
	u.Mod(u, curve25519P)

	out := make([]byte, curve25519.PointSize)
	uBytes := u.Bytes()
	for i, b := range uBytes {
		out[len(uBytes)-i-1] = b
	}

	return out
}
