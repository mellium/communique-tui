// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path"
	"time"

	"mellium.im/communiqué/internal/client/event"
)

func writeMessage(configPath string, msg event.ChatMessage) error {
	historyDir := path.Join(configPath, "history")
	err := os.MkdirAll(historyDir, 0755)
	if err != nil {
		return err
	}
	configPath = path.Join(historyDir, msg.From.Bare().String())
	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	/* #nosec */
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s %s\n", time.Now().UTC().Format(time.RFC3339), msg.Body)
	return err
}
