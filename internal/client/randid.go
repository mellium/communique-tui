// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package client

import (
	"crypto/rand"
	"fmt"
)

// idLen is the standard length of stanza identifiers in bytes.
const idLen = 16

// randomID generates a new random identifier of length IDLen. If the OS's
// entropy pool isn't initialized, or we can't generate random numbers for some
// other reason, panic.
func randomID() string {
	b := make([]byte, (idLen/2)+(idLen&1))
	switch n, err := rand.Reader.Read(b); {
	case err != nil:
		panic(err)
	case n != len(b):
		panic("Could not read enough randomness")
	}

	return fmt.Sprintf("%x", b)[:idLen]
}
