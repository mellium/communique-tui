// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package logwriter implements writing to log.Logger's.
package logwriter

import (
	"io"
	"log"
)

type logWriter struct {
	logger *log.Logger
}

func (lw logWriter) Write(p []byte) (int, error) {
	lw.logger.Println(string(p))
	return len(p), nil
}

// New returns a writer that mirrors all writes to the provided logger.
func New(logger *log.Logger) io.Writer {
	return logWriter{
		logger: logger,
	}
}
