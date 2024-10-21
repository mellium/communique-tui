// Copyright 2019 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mellium.im/communique/internal/client/event"
	"mellium.im/filechooser"
	"mellium.im/xmpp/stanza"
)

// UnreadRegion is a tview region tag that will draw an unread marker.
const UnreadRegion = "unreadMarker"

// ConversationView is a wrapper around TextView that adds other functionality
// important for displaying chats.
type ConversationView struct {
	*tview.Flex
	TextView   *tview.TextView
	inputPages *tview.Pages
	ui         *UI
}

const (
	pageFilePicker  = "page_filepick"
	pageInput       = "page_input"
	filePickerLabel = "📎"
)

// NewConversationView configures and creates a new chat view.
func NewConversationView(ui *UI) *ConversationView {
	p := ui.Printer()
	cv := ConversationView{
		Flex: tview.NewFlex().SetDirection(tview.FlexRow),
		TextView: tview.NewTextView().
			SetDynamicColors(true).
			SetRegions(true).
			ScrollToEnd().
			Highlight(UnreadRegion),
		inputPages: tview.NewPages(),
		ui:         ui,
	}
	filePicker := filechooser.NewPathInputField()
	filePicker.SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	filePicker.SetBorder(true)
	filePicker.SetLabel(filePickerLabel)
	filePicker.SetDoneFunc(func(key tcell.Key) {
		files := filePicker.Files()
		if key == tcell.KeyEnter && len(files) > 0 {
			cv.uploadFiles(files)
		}
		filePicker.SetText("")
		cv.ShowInput()
	})
	cv.TextView.SetBorder(true).SetTitle(p.Sprintf("Conversation"))
	input := tview.NewInputField()
	input.SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	input.SetBorder(true)
	cv.inputPages.AddPage(pageFilePicker, filePicker, true, false)
	cv.inputPages.AddPage(pageInput, input, true, true)
	cv.Flex.SetBorder(false)
	cv.Flex.AddItem(unreadTextView{TextView: cv.TextView}, 0, 100, false)
	cv.Flex.AddItem(cv.inputPages, 3, 1, true)
	cv.TextView.SetChangedFunc(func() {
		ui.app.Draw()
	})
	return &cv
}

// ShowFilePicker shows the file picker field.
func (cv *ConversationView) ShowFilePicker() {
	cv.inputPages.SwitchToPage(pageFilePicker)
	_, prim := cv.inputPages.GetFrontPage()
	prim.(*filechooser.PathInputField).SetText("")
}

// ShowInput shows the text input field.
func (cv *ConversationView) ShowInput() {
	cv.inputPages.SwitchToPage(pageInput)
}

func checkScroll(cv *ConversationView, f func()) {
	oldRow, _ := cv.TextView.GetScrollOffset()
	f()
	newRow, _ := cv.TextView.GetScrollOffset()
	if newRow == 0 && oldRow != newRow {
		item, ok := cv.ui.sidebar.GetSelected()
		if ok {
			// TODO: work with other types?
			if rosterItem, ok := item.(*RosterItem); ok {
				cv.ui.handler(event.PullToRefreshChat(rosterItem.Item))
			}
		}
	}
}

// ScrollTo scrolls to the specified row and column (both starting with 0).
func (cv *ConversationView) ScrollTo(row, column int) {
	checkScroll(cv, func() {
		cv.TextView.ScrollTo(row, column)
	})
}

// ScrollToBeginning scrolls to the top left corner of the text if the text view
// is scrollable.
func (cv *ConversationView) ScrollToBeginning() {
	checkScroll(cv, func() {
		cv.TextView.ScrollToBeginning()
	})
}

// ScrollToEnd scrolls to the bottom left corner of the text if the text view is
// scrollable.
// Adding new rows to the end of the text view will cause it to scroll with the
// new data.
func (cv *ConversationView) ScrollToEnd() {
	checkScroll(cv, func() {
		cv.TextView.ScrollToEnd()
	})
}

// ScrollToHighlight will cause the visible area to be scrolled so that the
// highlighted regions appear in the visible area of the text view.
// This repositioning happens the next time the text view is drawn.
// It happens only once so you will need to call this function repeatedly to
// always keep highlighted regions in view.
//
// Nothing happens if there are no highlighted regions or if the text view is
// not scrollable.
func (cv *ConversationView) ScrollToHighlight() {
	checkScroll(cv, func() {
		cv.TextView.ScrollToHighlight()
	})
}

// InputHandler returns the handler for this primitive.
func (cv *ConversationView) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return func(ev *tcell.EventKey, setFocus func(p tview.Primitive)) {

		// If the file upload dialog is selected, always pass through control to it.
		pageName, _ := cv.inputPages.GetFrontPage()
		if cv.inputPages.HasFocus() && pageName == pageFilePicker {
			cv.inputPages.InputHandler()(ev, setFocus)
			return
		}

		switch ev.Key() {
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyRight, tcell.KeyLeft, tcell.KeyPgUp, tcell.KeyPgDn:
			cv.TextView.InputHandler()(ev, setFocus)
		case tcell.KeyTAB, tcell.KeyBacktab:
			if cv.inputPages.HasFocus() {
				setFocus(cv.TextView)
			} else {
				setFocus(cv.inputPages)
			}
		case tcell.KeyESC:
			cv.ui.SelectRoster()
		case tcell.KeyEnter:
			if !cv.inputPages.HasFocus() {
				break
			}
			sendMsg(cv, ev, setFocus)
		case tcell.KeyCtrlU:
			if cv.ui.FilePickerConfigured() {
				p := cv.ui.Printer()
				files, err := cv.ui.FilePicker()
				if err != nil {
					cv.ui.logger.Print(p.Sprintf("error while picking files: %v", err))
					return
				}
				cv.uploadFiles(files)
				return
			}
			// If an external file picker isn't configured, launch the built-in one.
			cv.ShowFilePicker()
			setFocus(cv.inputPages)
		default:
			// Pass anything else to the input handler.
			if cv.inputPages.HasFocus() {
				cv.inputPages.InputHandler()(ev, setFocus)
			} else {
				checkScroll(cv, func() {
					cv.TextView.InputHandler()(ev, setFocus)
				})
			}
		}
	}
}

func sendMsg(cv *ConversationView, ev *tcell.EventKey, setFocus func(p tview.Primitive)) {
	_, prim := cv.inputPages.GetFrontPage()
	body := prim.(*tview.InputField).GetText()
	if body == "" {
		return
	}
	c, ok := cv.ui.sidebar.conversations.GetSelected()
	if !ok {
		return
	}
	typ := stanza.ChatMessage
	to := c.JID
	if c.Room {
		typ = stanza.GroupChatMessage
		to = to.Bare()
	}
	cv.ui.handler(event.ChatMessage{
		Message: stanza.Message{
			To:   to,
			Type: typ,
		},
		Body: body,
		Sent: true,
	})
	prim.(*tview.InputField).SetText("")
}

func (cv *ConversationView) uploadFiles(files []string) {
	p := cv.ui.Printer()

	rcpt, ok := cv.ui.sidebar.conversations.GetSelected()
	if !ok {
		cv.ui.logger.Print(p.Sprintf("failed to get the recipient"))
		return
	}
	typ := stanza.ChatMessage
	to := rcpt.JID
	if rcpt.Room {
		typ = stanza.GroupChatMessage
		to = to.Bare()
	}
	for _, file := range files {
		cv.ui.handler(event.UploadFile{
			Message: event.ChatMessage{
				Message: stanza.Message{
					To:   to,
					Type: typ,
				},
				Sent: true,
			},
			Path: file,
		})
	}
}

type unreadTextView struct {
	*tview.TextView
}

func (cv unreadTextView) Draw(screen tcell.Screen) {
	cv.TextView.Draw(screen)

	cv.TextView.Lock()
	defer cv.TextView.Unlock()

	x, y, width, height := cv.TextView.GetInnerRect()
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
