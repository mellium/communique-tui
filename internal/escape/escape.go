// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package escape contains a transformer that escapes tview IDs.
package escape // import "mellium.im/communique/internal/escape"

import (
	"strings"

	"github.com/mpvl/textutil"
	"golang.org/x/text/transform"
)

type escapeRewriter struct {
	inTag   bool
	hasBody bool
}

func (er *escapeRewriter) Rewrite(c textutil.State) {
	r, _ := c.ReadRune()
	switch {
	case r == '[':
		if !er.inTag {
			er.inTag = true
			er.hasBody = false
		}
	case r == ']':
		if er.inTag && er.hasBody {
			c.WriteRune('[')
		}
		er.inTag = false
	case !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !strings.ContainsRune(`_,;: -."#`, r):
		if er.inTag {
			er.inTag = false
		}
	default:
		if er.inTag {
			er.hasBody = true
		}
	}
	c.WriteRune(r)
}

func (er *escapeRewriter) Reset() {
	er.inTag = false
	er.hasBody = false
}

// Transformer returns a transformer that escapes color and/or region tags.
//
// For more information see Escape.
func Transformer() transform.Transformer {
	return textutil.NewTransformer(&escapeRewriter{})
}
