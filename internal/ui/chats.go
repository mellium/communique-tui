// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"

	"mellium.im/communiqué/internal/client/event"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

func newChats(ui *UI) (*tview.Flex, *tview.TextView) {
	chats := tview.NewFlex().
		SetDirection(tview.FlexRow)

	history := tview.NewTextView()
	history.SetBorder(true).SetTitle("Conversation")
	inputField := tview.NewInputField().SetFieldBackgroundColor(tcell.ColorDefault)
	inputField.SetBorder(true)
	chats.AddItem(history, 0, 100, false)
	chats.AddItem(inputField, 3, 1, false)

	history.SetChangedFunc(func() {
		ui.app.Draw()
	})
	chats.SetBorder(false)
	chats.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		// If escape is pressed, call the escape handler.
		switch ev.Key() {
		case tcell.KeyESC:
			ui.SelectRoster()
			return nil
		case tcell.KeyEnter:
			body := inputField.GetText()
			if body == "" {
				return nil
			}
			item, ok := ui.roster.GetSelected()
			if !ok {
				return nil
			}
			ui.handler(event.ChatMessage{
				Message: stanza.Message{
					To: item.Item.JID,
					// TODO: shouldn't this be automatically set by the library?
					From: jid.MustParse(ui.addr),
					Type: stanza.ChatMessage,
				},
				Body: body,
			})
			inputField.SetText("")
			return nil
		}

		// If anythig but Esc is pressed, pass input to the text box.
		capt := inputField.InputHandler()
		if capt != nil {
			capt(ev, nil)
		}
		return nil
	})

	return chats, history
}
