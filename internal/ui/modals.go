// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mellium.im/xmpp/jid"
)

const (
	cancelButton = "Cancel"
	execButton   = "Execute"
)

func modalClose(onEsc func()) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Key() == tcell.KeyESC {
			onEsc()
		}
		return event
	}
}

func delRosterModal(onEsc func(), onDel func()) *tview.Modal {
	const (
		removeButton = "Remove"
	)
	mod := tview.NewModal().
		SetText(`Remove this contact from your roster?`).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		AddButtons([]string{cancelButton, removeButton}).
		SetDoneFunc(func(_ int, buttonLabel string) {
			if buttonLabel == removeButton {
				onDel()
			}
			onEsc()
		})
	mod.SetInputCapture(modalClose(onEsc))
	return mod
}

func delBookmarkModal(onEsc func(), onDel func()) *tview.Modal {
	const (
		removeButton = "Remove"
	)
	mod := tview.NewModal().
		SetText(`Remove this channel?`).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		AddButtons([]string{cancelButton, removeButton}).
		SetDoneFunc(func(_ int, buttonLabel string) {
			if buttonLabel == removeButton {
				onDel()
			}
			onEsc()
		})
	mod.SetInputCapture(modalClose(onEsc))
	return mod
}

// getJID creates a modal that asks for a JID. Eg. to add a bookmark or start a
// new conversation.
func getJID(title, addButton string, f func(jid.JID, string), autocomplete func(string) []string) *Modal {
	mod := NewModal().
		SetText(title)
	var inputJID jid.JID
	jidInput := tview.NewInputField().SetPlaceholder("me@example.net")
	modForm := mod.Form()
	modForm.AddFormItem(jidInput)
	jidInput.SetChangedFunc(func(text string) {
		var err error
		inputJID, err = jid.Parse(text)
		if err == nil {
			jidInput.SetLabel("✅")
		} else {
			jidInput.SetLabel("❌")
		}
	})
	mod.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		AddButtons([]string{cancelButton, addButton}).
		SetDoneFunc(func(_ int, buttonLabel string) {
			f(inputJID.Bare(), buttonLabel)
		})
	return mod
}
