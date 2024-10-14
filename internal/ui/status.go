// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/rivo/tview"
	"golang.org/x/text/message"
)

func statusModal(p *message.Printer, done func(buttonIndex int, buttonLabel string)) *Modal {
	mod := NewModal().
		SetText(p.Sprintf("Set Status")).
		AddButtons([]string{
			p.Sprintf("Online %s", "[green]●"),
			p.Sprintf("Away %s", "[orange]◓"),
			p.Sprintf("Busy %s", "[red]◑"),
			p.Sprintf("Offline %s", "○"),
		}).
		SetDoneFunc(done).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	mod.SetInputCapture(modalClose(func() {
		done(-1, "")
	}))
	return mod
}
