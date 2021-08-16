// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package storage

import (
	"context"
	"database/sql"
)

// Iter is a generic iterator that can be returned to marshal database rows into
// values.
type Iter struct {
	err    error
	rows   *sql.Rows
	cur    interface{}
	f      func(*sql.Rows) (interface{}, error)
	cancel context.CancelFunc
}

// Next advances the iterator and returns whether the next row is ready to be
// read.
func (i *Iter) Next() bool {
	switch {
	case i == nil:
		return false
	case i.err != nil || i.rows == nil:
		i.cancel()
		return false
	}
	next := i.rows.Next()
	if !next {
		i.cancel()
		return next
	}

	if i.f != nil {
		i.cur, i.err = i.f(i.rows)
	}
	if i.err != nil {
		i.cancel()
		return false
	}
	return true
}

// Current returns the last parsed row.
func (i *Iter) Current() interface{} {
	if i == nil {
		return nil
	}
	return i.cur
}

// Err returns the error, if any, that was encountered during iteration.
// Err may be called after an explicit or implicit Close.
func (i *Iter) Err() error {
	if i == nil {
		return nil
	}
	switch i.err {
	case sql.ErrNoRows:
		return nil
	case nil:
		return i.rows.Err()
	}
	return i.err
}

// Close stops iteration.
// If Next is called and returns false, Close is called automatically.
// Close is idempotent and does not affect the result of Err.
func (i *Iter) Close() error {
	if i == nil {
		return nil
	}
	i.cancel()
	return i.rows.Close()
}
