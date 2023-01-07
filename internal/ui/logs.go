// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/text/message"
)

func newLogs(p *message.Printer, app *tview.Application, input func(event *tcell.EventKey) *tcell.EventKey) *tview.TextView {
	logs := tview.NewTextView()
	logs.SetText("")
	logs.SetBorder(true).SetTitle(p.Sprintf("Logs"))
	logs.SetInputCapture(input)

	return logs
}
