// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mellium.im/communique/internal/client/event"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// UnreadRegion is a tview region tag that will draw an unread marker.
const UnreadRegion = "unreadMarker"

// unreadTextView wraps a text view and draws the unread marker on any line that
// starts with a '─'.
type unreadTextView struct {
	*tview.TextView
}

func (t unreadTextView) Draw(screen tcell.Screen) {
	t.TextView.Draw(screen)

	t.TextView.Lock()
	defer t.TextView.Unlock()

	x, y, width, height := t.GetInnerRect()
	top := y + height

	var found bool
	for y < top {
		mainc, combc, _, width := screen.GetContent(x, y)
		// Scan for a line that starts with ─, and then draw the unread marker on
		// that line.
		if mainc == '─' && len(combc) == 0 && width == 1 {
			found = true
			break
		}
		y++
	}

	if !found {
		return
	}

	// TODO: set the style to something other than bold.
	screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
	screen.SetContent(x+1, y, ' ', nil, tcell.StyleDefault)
	for i := x + 2; i < x+width-2; i++ {
		screen.SetContent(i, y, '─', nil,
			tcell.StyleDefault.
				Bold(true).
				Foreground(tview.Styles.ContrastSecondaryTextColor),
		)
	}
}

func newChats(ui *UI) (*tview.Flex, unreadTextView) {
	chats := tview.NewFlex().
		SetDirection(tview.FlexRow)

	history := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		Highlight(UnreadRegion)
	history.SetBorder(true).SetTitle("Conversation")
	history.SetChangedFunc(func() {
		ui.app.Draw()
	})
	inputField := tview.NewInputField().SetFieldBackgroundColor(tcell.ColorDefault)
	inputField.SetBorder(true)
	unreadHistory := unreadTextView{
		TextView: history,
	}
	chats.AddItem(unreadHistory, 0, 100, false)
	chats.AddItem(inputField, 3, 1, true)

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

	return chats, unreadHistory
}
