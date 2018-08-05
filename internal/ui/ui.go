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

// UI is a widget that combines other widgets to make the main UI.
type UI struct {
	flex        *tview.Flex
	roster      roster.Roster
	hideJIDs    bool
	rosterWidth int
	defaultLog  string
	logWriter   io.Writer
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

// RosterWidth returns an option that sets the width of the roster.
// It accepts a minimum of 2 and a max of 50 the default is 25.
func RosterWidth(width int) Option {
	return func(ui *UI) {
		if width == 0 {
			ui.rosterWidth = 25
			return
		}
		if width < 2 {
			ui.rosterWidth = 2
			return
		}
		if width > 50 {
			ui.rosterWidth = 50
			return
		}
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
func New(app *tview.Application, opts ...Option) UI {
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
	logs.SetBorder(true).SetTitle("Logs")
	logs.SetInputCapture(rosterFocus)
	pages.AddPage(statusPageName, logs, true, true)
	setStatusPage := tview.NewModal().
		SetText("Set Status").
		AddButtons([]string{"Online [green]●", "Away [orange]●", "Busy [red]●", "Offline [silver]●"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			switch buttonIndex {
			case 0:
				rosterBox.Online()
			case 1:
				rosterBox.Away()
			case 2:
				rosterBox.Busy()
			case 3:
				rosterBox.Offline()
			}
			pages.SwitchToPage(statusPageName)
			app.SetFocus(rosterBox)
		})
	//setStatusPage.SetInputCapture(rosterFocus)
	pages.AddPage(setStatusPageName, setStatusPage, true, false)

	rosterBox.Upsert("[orange]●[white] Thespian", "", mainFocus)
	rosterBox.Upsert("[red]●[white] Twinkletoes", "  🆒🍩 🍪 🍫👆👇👈👉👊👋", mainFocus)
	rosterBox.Upsert("[green]●[white] Papa Shrimp", "  Online, let's chat", mainFocus)
	rosterBox.Upsert("[silver]●[white] Pockets full of Sunshine", "  Listening to: Watermark by Enya", mainFocus)

	ui := UI{
		roster:      rosterBox,
		rosterWidth: 25,
		logWriter:   logs,
	}
	for _, o := range opts {
		o(&ui)
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

// Write writes to the logging text view.
func (ui UI) Write(p []byte) (n int, err error) {
	return ui.logWriter.Write(p)
}

// Roster returns the underlying roster pane widget.
func (ui UI) Roster() roster.Roster {
	return ui.roster
}

// Draw implements tview.Primitive foui UI.
func (ui UI) Draw(screen tcell.Screen) {
	ui.flex.Draw(screen)
}

// GetRect implements tview.Primitive foui UI.
func (ui UI) GetRect() (int, int, int, int) {
	return ui.flex.GetRect()
}

// SetRect implements tview.Primitive foui UI.
func (ui UI) SetRect(x, y, width, height int) {
	ui.flex.SetRect(x, y, width, height)
}

// InputHandler implements tview.Primitive foui UI.
func (ui UI) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return ui.flex.InputHandler()
}

// Focus implements tview.Primitive foui UI.
func (ui UI) Focus(delegate func(p tview.Primitive)) {
	ui.flex.Focus(delegate)
}

// Blur implements tview.Primitive foui UI.
func (ui UI) Blur() {
	ui.flex.Blur()
}

// GetFocusable implements tview.Primitive foui UI.
func (ui UI) GetFocusable() tview.Focusable {
	return ui.flex.GetFocusable()
}
