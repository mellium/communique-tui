// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package ui ties together various widgets to create the main Communiqué UI.
package ui // import "mellium.im/communiqué/internal/ui"

import (
	"fmt"
	"io"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

const (
	uiPageName          = "ui"
	quitPageName        = "quit"
	getPasswordPageName = "get_password"
	setStatusPageName   = "set_status"
	statusPageName      = "status"
)

// Event is any UI event.
type Event string

// A list of events.
const (
	// The user has indicated that they want to change their status.
	GoOnline  Event = "go-online"
	GoOffline Event = "go-offline"
	GoAway    Event = "go-away"
	GoBusy    Event = "go-busy"
)

// UI is a widget that combines other widgets to make the main UI.
type UI struct {
	app         *tview.Application
	flex        *tview.Flex
	pages       *tview.Pages
	buffers     *tview.Pages
	statusBar   *tview.TextView
	roster      Roster
	hideJIDs    bool
	rosterWidth int
	defaultLog  string
	logWriter   io.Writer
	handler     func(Event)
	redraw      func() *tview.Application
	mainFocus   func()
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

// Handle returns an option that configures an event handler which will be called
// when the user performs certain actions in the UI.
// Only one event handler can be registered, and subsequent calls to Event will
// replace the handler.
// The function will be called synchronously on the UI goroutine, so don't do
// any intensive work (or launch a new goroutine if you must).
func Handle(handler func(Event)) Option {
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
		SetBackgroundColor(tcell.ColorGreen).
		SetBorder(false).
		SetBorderPadding(0, 0, 2, 0)
	buffers := tview.NewPages()
	pages := tview.NewPages()

	rosterBox := NewRoster(func() {
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
		statusBar.SetText(fmt.Sprintf("Chat: %q (%s)", main, secondary))
	})

	logs := newLogs(app, func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyESC {
			app.SetFocus(rosterBox)
			return nil
		}
		return event
	})
	buffers.AddPage(statusPageName, logs, true, true)
	ui := &UI{
		app:         app,
		roster:      rosterBox,
		rosterWidth: 25,
		statusBar:   statusBar,
		logWriter:   logs,
		handler:     func(Event) {},
		redraw:      app.Draw,
		buffers:     buffers,
		pages:       pages,
		passPrompt:  make(chan string),
	}
	for _, o := range opts {
		o(ui)
	}
	logs.SetText(ui.defaultLog)

	setStatusPage := statusModal(func(buttonIndex int, buttonLabel string) {
		switch buttonIndex {
		case 0:
			ui.handler(GoOnline)
		case 1:
			ui.handler(GoAway)
		case 2:
			ui.handler(GoBusy)
		case 3:
			ui.handler(GoOffline)
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

// AddRoster adds an item to the roster.
func (ui *UI) AddRoster(item RosterItem) {
	ui.roster.Upsert(item, ui.mainFocus)
}

// Write writes to the logging text view.
func (ui *UI) Write(p []byte) (n int, err error) {
	return ui.logWriter.Write(p)
}

// Roster returns the underlying roster pane widget.
func (ui *UI) Roster() Roster {
	return ui.roster
}

// Draw implements tview.Primitive foui UI.
func (ui *UI) Draw(screen tcell.Screen) {
	ui.pages.Draw(screen)
}

// GetRect implements tview.Primitive foui UI.
func (ui *UI) GetRect() (int, int, int, int) {
	return ui.pages.GetRect()
}

// SetRect implements tview.Primitive foui UI.
func (ui *UI) SetRect(x, y, width, height int) {
	ui.pages.SetRect(x, y, width, height)
}

// InputHandler implements tview.Primitive foui UI.
func (ui *UI) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return ui.pages.InputHandler()
}

// Focus implements tview.Primitive foui UI.
func (ui *UI) Focus(delegate func(p tview.Primitive)) {
	ui.pages.Focus(delegate)
}

// Blur implements tview.Primitive foui UI.
func (ui *UI) Blur() {
	ui.pages.Blur()
}

// GetFocusable implements tview.Primitive foui UI.
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
// Only one event handler can be registered, and subsequent calls to Event will
// replace the handler.
// The function will be called synchronously on the UI goroutine, so don't do
// any intensive work (or launch a new goroutine if you must).
func (ui *UI) Handle(handler func(Event)) {
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
