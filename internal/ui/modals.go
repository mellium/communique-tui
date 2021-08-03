// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mellium.im/xmpp/jid"
)

func modalClose(onEsc func()) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Key() == tcell.KeyESC {
			onEsc()
		}
		return event
	}
}

func addRosterModal(autocomplete func(s string) []string, onEsc func(), onAdd func(jid.JID)) *Modal {
	const (
		cancelButton = "Cancel"
		addButton    = "Add"
	)
	mod := NewModal().
		SetText(`Start Chat`)
	var inputJID jid.JID
	jidInput := tview.NewInputField().SetPlaceholder("me@example.net")
	mod.Form().AddFormItem(jidInput)
	jidInput.SetChangedFunc(func(text string) {
		var err error
		inputJID, err = jid.Parse(text)
		if err == nil {
			jidInput.SetLabel("✅")
		} else {
			jidInput.SetLabel("❌")
		}
	}).SetAutocompleteFunc(autocomplete)
	mod.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		AddButtons([]string{cancelButton, addButton}).
		SetDoneFunc(func(_ int, buttonLabel string) {
			if buttonLabel == addButton {
				onAdd(inputJID.Bare())
			}
			onEsc()
		})
	return mod
}

func delRosterModal(onEsc func(), onDel func()) *tview.Modal {
	const (
		cancelButton = "Cancel"
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

func infoModal(onEsc func()) *tview.Modal {
	mod := tview.NewModal().
		SetText(`Roster info:`).
		SetDoneFunc(func(int, string) {
			onEsc()
		}).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	mod.SetInputCapture(modalClose(onEsc))
	return mod
}

func helpModal(onEsc func()) *tview.Modal {
	// U+20E3 COMBINING ENCLOSING KEYCAP
	mod := tview.NewModal().
		SetText(`Global :

q⃣: quit or close
⎋⃣: close
K⃣: help


Navigation:

⇥⃣, ⇤⃣ focus to next/prev
g⃣ g⃣, ⇱⃣ scroll to top
G⃣, ⇲⃣ scroll to bottom
h⃣, ←⃣ move left
j⃣, ↓⃣ move down
k⃣, ↑⃣ move up
l⃣, →⃣ move right
⇞⃣, ⇟⃣ move up/down one page
1⃣ 0⃣ k⃣ move 10 lines up
1⃣ 0⃣ j⃣ move 10 lines down
/⃣ search forward
?⃣ search backward
n⃣ next search result
N⃣ previous search result


Roster:

c⃣ start chat
i⃣, ⏎⃣ open chat
I⃣ more info
o⃣, O⃣ open next/prev unread
d⃣ d⃣ remove contact
`).
		SetDoneFunc(func(int, string) {
			onEsc()
		}).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	mod.SetInputCapture(modalClose(onEsc))
	return mod
}
