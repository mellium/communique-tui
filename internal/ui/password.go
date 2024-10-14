// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/rivo/tview"
	"golang.org/x/text/message"
)

func passwordModal(p *message.Printer, addr string, done func(*tview.Form)) tview.Primitive {
	getPasswordPage := tview.NewForm().
		AddPasswordField(p.Sprintf("Password"), "", 0, 0, nil).
		SetButtonsAlign(tview.AlignRight)
	getPasswordPage.AddButton(p.Sprintf("Login"), func() {
		done(getPasswordPage)
	})
	getPasswordPage.
		SetBorder(true).
		SetTitle(p.Sprintf("Enter password for: %q", addr))

	return tview.NewGrid().
		SetColumns(0, 50, 0).
		SetRows(0, 7, 0).
		AddItem(getPasswordPage, 1, 1, 1, 1, 0, 0, true)
}
