// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

func newChats(app *tview.Application, onEsc func()) *tview.Flex {
	chats := tview.NewFlex().
		SetDirection(tview.FlexRow)

	const nyi = "TODO: Not yet implemented."
	history := tview.NewTextView().SetText(nyi)
	history.SetBorder(true).SetTitle("Conversation")
	inputField := tview.NewInputField().SetText(nyi)
	inputField.SetBorder(true)
	chats.AddItem(history, 0, 100, false)
	chats.AddItem(inputField, 3, 1, false)

	history.SetChangedFunc(func() {
		app.Draw()
	})
	chats.SetBorder(false)
	chats.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// If escape is pressed, call the escape handler.
		if event.Key() == tcell.KeyESC {
			onEsc()
			return nil
		}

		// If anythig but Esc is pressed, pass input to the text box.
		capt := inputField.InputHandler()
		if capt != nil {
			capt(event, nil)
		}
		return nil
	})

	return chats
}
