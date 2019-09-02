// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package ui ties together various widgets to create the main Communiqué UI.
package ui // import "mellium.im/communiqué/internal/ui"

import (
	"fmt"
	"io"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"

	"mellium.im/communiqué/internal/client/event"
)

const (
	getPasswordPageName = "get_password"
	logsPageName        = "logs"
	chatPageName        = "chat"
	quitPageName        = "quit"
	setStatusPageName   = "set_status"
	uiPageName          = "ui"
)

// UI is a widget that combines other widgets to make the main UI.
type UI struct {
	app         *tview.Application
	flex        *tview.Flex
	pages       *tview.Pages
	buffers     *tview.Pages
	history     *tview.TextView
	statusBar   *tview.TextView
	roster      Roster
	hideJIDs    bool
	rosterWidth int
	defaultLog  string
	logWriter   io.Writer
	handler     func(interface{})
	redraw      func() *tview.Application
	addr        string
	passPrompt  chan string
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

// Log returns an option that sets the default string to show in the log window
// on startup.
func Log(s string) Option {
	return func(ui *UI) {
		ui.defaultLog = s
	}
}

// New constructs a new UI.
func New(app *tview.Application, opts ...Option) *UI {
	statusBar := tview.NewTextView()
	statusBar.SetChangedFunc(func() {
		app.Draw()
	})
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
		app.Draw()
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
	}
	for _, o := range opts {
		o(ui)
	}

	innerCapture := rosterBox.GetInputCapture()
	rosterBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyTAB:
			buffers.SwitchToPage(logsPageName)
			app.SetFocus(buffers)
			app.Draw()
			return nil
		case event.Rune() == 'q':
			ui.ShowQuitPrompt()
			return nil
		}

		if innerCapture != nil {
			return innerCapture(event)
		}

		return event
	})

	chats, history := newChats(ui)
	ui.history = history
	buffers.AddPage(chatPageName, chats, true, false)

	logs := newLogs(app, func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyESC {
			ui.SelectRoster()
			return nil
		}
		return event
	})
	buffers.AddPage(logsPageName, logs, true, true)
	logs.SetText(ui.defaultLog)
	ui.logWriter = logs

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
	ui.pages.AddPage(getPasswordPageName, getPasswordPage, true, false)

	return ui
}

// UpdateRoster adds an item to the roster.
func (ui *UI) UpdateRoster(item RosterItem) {
	ui.roster.Upsert(item, func() {
		ui.buffers.ShowPage(chatPageName)
		ui.buffers.SendToFront(chatPageName)
		item, ok := ui.roster.GetSelected()
		if ok {
			ui.handler(event.OpenChat(item.Item))
		}
		ui.app.SetFocus(ui.buffers)
		ui.app.Draw()
	})
	ui.redraw()
}

// Write writes to the logging text view.
func (ui *UI) Write(p []byte) (n int, err error) {
	return ui.logWriter.Write(p)
}

// Roster returns the underlying roster pane widget.
func (ui *UI) Roster() Roster {
	return ui.roster
}

// Draw implements tview.Primitive for UI.
func (ui *UI) Draw(screen tcell.Screen) {
	ui.pages.Draw(screen)
}

// GetRect implements tview.Primitive for UI.
func (ui *UI) GetRect() (int, int, int, int) {
	return ui.pages.GetRect()
}

// SetRect implements tview.Primitive for UI.
func (ui *UI) SetRect(x, y, width, height int) {
	ui.pages.SetRect(x, y, width, height)
}

// InputHandler implements tview.Primitive for UI.
func (ui *UI) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return ui.pages.InputHandler()
}

// Focus implements tview.Primitive for UI.
func (ui *UI) Focus(delegate func(p tview.Primitive)) {
	ui.pages.Focus(delegate)
}

// Blur implements tview.Primitive for UI.
func (ui *UI) Blur() {
	ui.pages.Blur()
}

// GetFocusable implements tview.Primitive for UI.
func (ui *UI) GetFocusable() tview.Focusable {
	return ui.pages.GetFocusable()
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
	ui.app.Draw()
	return <-ui.passPrompt
}

// ShowQuitPrompt asks if the user wants to quit the application.
func (ui *UI) ShowQuitPrompt() {
	ui.pages.ShowPage(quitPageName)
	ui.pages.SendToFront(quitPageName)
	ui.app.SetFocus(ui.pages)
	ui.app.Draw()
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
	return ui.history
}

// Redraw redraws the UI.
func (ui *UI) Redraw() {
	ui.redraw()
}
