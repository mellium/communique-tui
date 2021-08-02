// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package ui ties together various widgets to create the main Communiqué UI.
package ui // import "mellium.im/communique/internal/ui"

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mellium.im/communique/internal/client/event"
)

const (
	getPasswordPageName = "get_password"
	logsPageName        = "logs"
	chatPageName        = "chat"
	quitPageName        = "quit"
	helpPageName        = "help"
	infoPageName        = "info"
	setStatusPageName   = "set_status"
	uiPageName          = "ui"
)

type syncBool struct {
	b bool
	m sync.Mutex
}

func (b *syncBool) Set(v bool) {
	b.m.Lock()
	defer b.m.Unlock()
	b.b = v
}

func (b *syncBool) Get() bool {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b
}

// UI is a widget that combines other widgets to make the main UI.
type UI struct {
	app         *tview.Application
	flex        *tview.Flex
	pages       *tview.Pages
	buffers     *tview.Pages
	history     unreadTextView
	statusBar   *tview.TextView
	roster      *Roster
	rosterWidth int
	logWriter   *tview.TextView
	handler     func(interface{})
	redraw      func() *tview.Application
	addr        string
	passPrompt  chan string
	chatsOpen   *syncBool
	infoModal   *tview.Modal
}

// Run starts the application event loop.
func (ui *UI) Run() error {
	ui.logWriter.SetChangedFunc(func() {
		ui.app.Draw()
	})

	return ui.app.SetRoot(ui.pages, true).SetFocus(ui.pages).Run()
}

// Stop stops the application, causing Run() to return.
func (ui *UI) Stop() {
	ui.app.Stop()
}

// Option can be used to configure a new roster widget.
type Option func(*UI)

// ShowStatus returns an option that shows or hides the status line under
// contacts in the roster.
func ShowStatus(show bool) Option {
	return func(ui *UI) {
		ui.roster.ShowStatus(show)
	}
}

// InputCapture returns an option that overrides the default input handler for
// the application.
func InputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) Option {
	return func(ui *UI) {
		ui.app.SetInputCapture(capture)
	}
}

// Addr returns an option that sets the users address anywhere that it is
// displayed in the UI.
func Addr(addr string) Option {
	return func(ui *UI) {
		ui.addr = addr
	}
}

// Handle returns an option that configures an event handler which will be
// called when the user performs certain actions in the UI.
// Only one event handler can be registered, and subsequent calls to Handle will
// replace the handler.
// The function will be called synchronously on the UI goroutine, so don't do
// any intensive work (or, if you must, launch a new goroutine).
func Handle(handler func(event interface{})) Option {
	return func(ui *UI) {
		ui.handler = handler
	}
}

// RosterWidth returns an option that sets the width of the roster.
// It accepts a minimum of 2 and a max of 50 the default is 25.
func RosterWidth(width int) Option {
	return func(ui *UI) {
		if width == 0 {
			ui.roster.Width = 25
			ui.rosterWidth = 25
			return
		}
		if width < 2 {
			ui.roster.Width = 2
			ui.rosterWidth = 2
			return
		}
		if width > 50 {
			ui.roster.Width = 50
			ui.rosterWidth = 50
			return
		}
		ui.roster.Width = width
		ui.rosterWidth = width
	}
}

// New constructs a new UI.
func New(opts ...Option) *UI {
	app := tview.NewApplication()
	statusBar := tview.NewTextView()
	statusBar.
		SetTextColor(tview.Styles.PrimaryTextColor).
		SetBackgroundColor(tview.Styles.MoreContrastBackgroundColor).
		SetBorder(false).
		SetBorderPadding(0, 0, 2, 0)
	buffers := tview.NewPages()
	pages := tview.NewPages()

	rosterBox := newRoster(func() {
		pages.ShowPage(setStatusPageName)
		pages.SendToFront(setStatusPageName)
		app.SetFocus(pages)
	})
	rosterBox.OnChanged(func(idx int, main string, secondary string, shortcut rune) {
		if idx == 0 {
			statusBar.SetText("Status: " + main)
			return
		}
		main = strings.TrimPrefix(main, highlightTag)
		statusBar.SetText(fmt.Sprintf("Chat: %q (%s)", main, secondary))
	})

	ui := &UI{
		app:         app,
		roster:      rosterBox,
		rosterWidth: 25,
		statusBar:   statusBar,
		handler:     func(interface{}) {},
		redraw:      app.Draw,
		buffers:     buffers,
		pages:       pages,
		passPrompt:  make(chan string),
		chatsOpen:   &syncBool{},
	}
	ui.infoModal = infoModal(func() {
		ui.pages.HidePage(infoPageName)
	})
	for _, o := range opts {
		o(ui)
	}

	chats, history := newChats(ui)
	ui.history = history
	buffers.AddPage(chatPageName, chats, true, false)

	logs := newLogs(app, func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyESC, tcell.KeyTAB, tcell.KeyBacktab:
			ui.chatsOpen.Set(false)
			ui.SelectRoster()
			return nil
		}
		return event
	})
	buffers.AddPage(logsPageName, logs, true, true)
	ui.logWriter = logs

	innerCapture := rosterBox.GetInputCapture()
	rosterBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		key := event.Key()
		eventRune := event.Rune()
		switch {
		case key == tcell.KeyTAB || key == tcell.KeyBacktab:
			buffers.SwitchToPage(logsPageName)
			app.SetFocus(buffers)
			return nil
		case eventRune == 'q':
			ui.ShowQuitPrompt()
			return nil
		case eventRune == 'K' || key == tcell.KeyF1 || key == tcell.KeyHelp:
			ui.ShowHelpPrompt()
			return nil
		case eventRune == 'I':
			ui.ShowRosterInfo()
			return nil
		}

		if innerCapture != nil {
			return innerCapture(event)
		}

		return event
	})

	setStatusPage := statusModal(func(buttonIndex int, buttonLabel string) {
		switch buttonIndex {
		case 0:
			ui.handler(event.StatusOnline{})
		case 1:
			ui.handler(event.StatusAway{})
		case 2:
			ui.handler(event.StatusBusy{})
		case 3:
			ui.handler(event.StatusOffline{})
		}
		ui.pages.HidePage(setStatusPageName)
	})

	getPasswordPage := passwordModal(ui.addr, func(getPasswordPage *tview.Form) {
		ui.passPrompt <- getPasswordPage.GetFormItem(0).(*tview.InputField).GetText()
		ui.pages.HidePage(getPasswordPageName)
	})

	ltrFlex := tview.NewFlex().
		AddItem(rosterBox, ui.rosterWidth, 1, true).
		AddItem(buffers, 0, 1, false)
	ui.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ltrFlex, 0, 1, true).
		AddItem(statusBar, 1, 1, false)

	ui.pages.AddPage(setStatusPageName, setStatusPage, true, false)
	ui.pages.AddPage(uiPageName, ui.flex, true, true)
	ui.pages.AddPage(quitPageName, quitModal(func(buttonIndex int, buttonLabel string) {
		if buttonIndex == 0 {
			app.Stop()
		}
		ui.pages.HidePage(quitPageName)
	}), true, false)
	ui.pages.AddPage(helpPageName, helpModal(func() {
		ui.pages.HidePage(helpPageName)
	}), true, false)
	ui.pages.AddPage(infoPageName, ui.infoModal, true, false)
	ui.pages.AddPage(getPasswordPageName, getPasswordPage, true, false)

	return ui
}

// RosterLen returns the length of the roster.
func (ui *UI) RosterLen() int {
	return ui.roster.Len()
}

// UpdateRoster adds an item to the roster.
func (ui *UI) UpdateRoster(item RosterItem) {
	ui.roster.Upsert(item, func() {
		ui.buffers.SwitchToPage(chatPageName)
		ui.chatsOpen.Set(true)
		item, ok := ui.roster.GetSelected()
		if ok {
			ui.handler(event.OpenChat(item.Item))
		}
		ui.app.SetFocus(ui.buffers)
	})
	ui.redraw()
}

// Write writes to the logging text view.
func (ui *UI) Write(p []byte) (n int, err error) {
	return ui.logWriter.Write(p)
}

// Roster returns the underlying roster pane widget.
func (ui *UI) Roster() *Roster {
	return ui.roster
}

// ChatsOpen returns true if the chat pane is open.
func (ui *UI) ChatsOpen() bool {
	return ui.chatsOpen.Get()
}

// Offline sets the state of the roster to show the user as offline.
func (ui *UI) Offline() {
	ui.roster.Offline()
	ui.redraw()
}

// Online sets the state of the roster to show the user as online.
func (ui *UI) Online() {
	ui.roster.Online()
	ui.redraw()
}

// Away sets the state of the roster to show the user as away.
func (ui *UI) Away() {
	ui.roster.Away()
	ui.redraw()
}

// Busy sets the state of the roster to show the user as busy.
func (ui *UI) Busy() {
	ui.roster.Busy()
	ui.redraw()
}

// Handle configures an event handler which will be called when the user
// performs certain actions in the UI.
// Only one event handler can be registered, and subsequent calls to Handle will
// replace the handler.
// The function will be called synchronously on the UI goroutine, so don't do
// any intensive work (or launch a new goroutine if you must).
func (ui *UI) Handle(handler func(interface{})) {
	ui.handler = handler
}

// ShowPasswordPrompt displays a modal and blocks until the user enters a
// password and submits it.
func (ui *UI) ShowPasswordPrompt() string {
	ui.pages.ShowPage(getPasswordPageName)
	ui.pages.SendToFront(getPasswordPageName)
	ui.app.SetFocus(ui.pages)
	return <-ui.passPrompt
}

// ShowQuitPrompt asks if the user wants to quit the application.
func (ui *UI) ShowQuitPrompt() {
	ui.pages.ShowPage(quitPageName)
	ui.pages.SendToFront(quitPageName)
	ui.app.SetFocus(ui.pages)
}

// ShowHelpPrompt shows a list of keyboard shortcuts..
func (ui *UI) ShowHelpPrompt() {
	ui.pages.ShowPage(helpPageName)
	ui.pages.SendToFront(helpPageName)
	ui.app.SetFocus(ui.pages)
}

func (ui *UI) ShowRosterInfo() {
	item, ok := ui.roster.GetSelected()
	if !ok {
		idx := ui.roster.list.GetCurrentItem()
		main, secondary := ui.roster.list.GetItemText(idx)
		ui.infoModal.SetText(fmt.Sprintf(`%s
%s
`, main, secondary))
	} else {
		subscriptionIcon := "✘"
		switch item.Subscription {
		case "both":
			subscriptionIcon = "⇆"
		case "to":
			subscriptionIcon = "→"
		case "from":
			subscriptionIcon = "←"
		}
		name := item.Name
		if name == "" {
			name = item.JID.Localpart()
		}
		ui.infoModal.SetText(fmt.Sprintf(`🛈

%s
%s

Subscription: %s
Groups: %v
`, name, item.JID, subscriptionIcon, item.Group))
	}
	ui.pages.ShowPage(infoPageName)
	ui.pages.SendToFront(infoPageName)
	ui.app.SetFocus(ui.pages)
}

// SelectRoster moves the input selection back to the roster and shows the logs
// view.
func (ui *UI) SelectRoster() {
	item, ok := ui.roster.GetSelected()
	if ok {
		ui.handler(event.CloseChat(item.Item))
	}
	ui.buffers.SwitchToPage(logsPageName)
	ui.app.SetFocus(ui.roster)
}

// History returns the chat history view.
// To flush any remaining data to the buffer, the writer must be closed after
// use.
func (ui *UI) History() *tview.TextView {
	return ui.history.TextView
}

// Redraw redraws the UI.
func (ui *UI) Redraw() {
	ui.redraw()
}
