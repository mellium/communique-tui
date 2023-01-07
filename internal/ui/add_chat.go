// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.
package ui

import (
	"github.com/rivo/tview"
	"golang.org/x/text/message"

	"mellium.im/xmpp/jid"
)

type addRosterForm struct {
	addr jid.JID
	nick string
}

// addRoster creates a modal that asks for a JID to add to the roster.
func addRoster(p *message.Printer, addButton string, autocomplete []jid.JID, f func(addRosterForm, string)) *Modal {
	mod := NewModal()
	mod.SetText(p.Sprintf("Add Contact"))
	modForm := mod.Form()

	nickInput := tview.NewInputField()

	var inputJID jid.JID
	jidInput := jidInput(p, &inputJID, true, autocomplete, nil)
	jidInput.SetLabel(p.Sprintf("Address"))
	modForm.AddFormItem(jidInput)

	nickInput.SetLabel(p.Sprintf("Name"))
	modForm.AddFormItem(nickInput)

	var cancelButton = p.Sprintf("Cancel")
	mod.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		AddButtons([]string{cancelButton, addButton}).
		SetDoneFunc(func(_ int, buttonLabel string) {
			f(addRosterForm{
				addr: inputJID.Bare(),
				nick: nickInput.GetText(),
			}, buttonLabel)
		})
	return mod
}
