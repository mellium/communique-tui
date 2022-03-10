// Copyright 2020 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.
package ui

import (
	"strings"

	"github.com/rivo/tview"

	"mellium.im/xmpp/jid"
)

type addRosterForm struct {
	addr jid.JID
	nick string
}

// addRoster creates a modal that asks for a JID to add to the roster.
func addRoster(addButton string, autocomplete []jid.JID, f func(addRosterForm, string)) *Modal {
	mod := NewModal()
	mod.SetText("Add Contact")
	modForm := mod.Form()

	nickInput := tview.NewInputField()

	var inputJID jid.JID
	jidInput := jidInput(&inputJID, true, autocomplete, func(text string) {
		if idx := strings.IndexByte(text, '@'); idx > -1 {
			text = text[:idx]
		}
	})
	jidInput.SetLabel("Address")
	modForm.AddFormItem(jidInput)

	nickInput.SetLabel("Name")
	modForm.AddFormItem(nickInput)

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
