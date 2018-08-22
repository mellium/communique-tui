// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package ui ties together various widgets to create the main Communiqué UI.
package ui

import (
	"io"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"

	"mellium.im/communiqué/internal/roster"
)

const (
	setStatusPageName = "Set Status"
	statusPageName    = "Status"
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
	flex        *tview.Flex
	pages       *tview.Pages
	roster      roster.Roster
	hideJIDs    bool
	rosterWidth int
	defaultLog  string
	logWriter   io.Writer
	handler     func(Event)
	redraw      func() *tview.Application
	mainFocus   func()
}

// Option can be used to configure a new roster widget.
type Option func(*UI)

// ShowStatus returns an option that shows or hides the status line under
// contacts in the roster.
func ShowStatus(show bool) Option {
	s := roster.ShowStatus(show)
	return func(ui *UI) {
		s(&ui.roster)
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
	statusBar := tview.NewBox().SetBorder(true)
	pages := tview.NewPages()

	mainFocus := func() {
		app.SetFocus(pages)
	}

	rosterBox := roster.New(
		roster.Title("Roster"),
		roster.OnStatus(func() {
			mainFocus()
			pages.ShowPage(setStatusPageName)
		}),
	)

	rosterFocus := func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyESC {
			app.SetFocus(rosterBox)
			return nil
		}
		return event
	}

	logs := tview.NewTextView()
	logs.SetChangedFunc(func() {
		app.Draw()
	})
	logs.SetBorder(true).SetTitle("Logs")
	logs.SetInputCapture(rosterFocus)
	pages.AddPage(statusPageName, logs, true, true)
	ui := &UI{
		roster:      rosterBox,
		rosterWidth: 25,
		logWriter:   logs,
		handler:     func(Event) {},
		redraw:      app.Draw,
		pages:       pages,
		mainFocus:   mainFocus,
	}
	setStatusPage := tview.NewModal().
		SetText("Set Status").
		AddButtons([]string{"Online [green]●", "Away [orange]●", "Busy [red]●", "Offline [silver]●"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
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
			pages.SwitchToPage(statusPageName)
			app.SetFocus(rosterBox)
		})
	pages.AddPage(setStatusPageName, setStatusPage, true, false)

	for _, o := range opts {
		o(ui)
	}
	logs.SetText(ui.defaultLog)

	ltrFlex := tview.NewFlex().
		AddItem(rosterBox, ui.rosterWidth, 1, true).
		AddItem(pages, 0, 1, false)
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ltrFlex, 0, 1, true).
		AddItem(statusBar, 2, 1, false)
	ui.flex = flex

	return ui
}

// AddRoster adds an item to the roster.
func (ui *UI) AddRoster(name, addr string) {
	ui.roster.Upsert("[silver]●[white] "+name, "  "+addr, ui.mainFocus)
}

// Write writes to the logging text view.
func (ui *UI) Write(p []byte) (n int, err error) {
	return ui.logWriter.Write(p)
}

// Roster returns the underlying roster pane widget.
func (ui *UI) Roster() roster.Roster {
	return ui.roster
}

// Draw implements tview.Primitive foui UI.
func (ui *UI) Draw(screen tcell.Screen) {
	ui.flex.Draw(screen)
}

// GetRect implements tview.Primitive foui UI.
func (ui *UI) GetRect() (int, int, int, int) {
	return ui.flex.GetRect()
}

// SetRect implements tview.Primitive foui UI.
func (ui *UI) SetRect(x, y, width, height int) {
	ui.flex.SetRect(x, y, width, height)
}

// InputHandler implements tview.Primitive foui UI.
func (ui *UI) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return ui.flex.InputHandler()
}

// Focus implements tview.Primitive foui UI.
func (ui *UI) Focus(delegate func(p tview.Primitive)) {
	ui.flex.Focus(delegate)
}

// Blur implements tview.Primitive foui UI.
func (ui *UI) Blur() {
	ui.flex.Blur()
}

// GetFocusable implements tview.Primitive foui UI.
func (ui *UI) GetFocusable() tview.Focusable {
	return ui.flex.GetFocusable()
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
