// Copyright 2024 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package localerr contains localizable errors.
package localerr // import "mellium.im/communique/internal/localerr"

import (
	"errors"

	"golang.org/x/text/message"
)

// Wrap returns an error that translates its reference string and wraps any
// arguments that are also errors.
// Unlike fmt.Errorf it does not check the verbs and will wrap errors even if
// they do not use %w.
func Wrap(p *message.Printer, ref message.Reference, a ...any) error {
	var errs []error
	for _, v := range a {
		if err, ok := v.(error); ok {
			errs = append(errs, err)
		}
	}
	switch len(errs) {
	case 0:
		return errors.New(p.Sprintf(ref, a...))
	case 1:
		return &wrapError{
			msg: p.Sprintf(ref, a...),
			err: errs[0],
		}
	default:
		return &wrapErrors{
			msg:  p.Sprintf(ref, a...),
			errs: errs,
		}
	}
}

type wrapError struct {
	msg string
	err error
}

func (e *wrapError) Error() string {
	return e.msg
}

func (e *wrapError) Unwrap() error {
	return e.err
}

type wrapErrors struct {
	msg  string
	errs []error
}

func (e *wrapErrors) Error() string {
	return e.msg
}

func (e *wrapErrors) Unwrap() []error {
	return e.errs
}
