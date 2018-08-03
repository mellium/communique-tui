// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package ui ties together various widgets to create the main Communiqué UI.
package ui

import (
	"mellium.im/communiqué/internal/roster"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

// UI is a widget that combines other widgets to make the main UI.
type UI struct {
	flex     *tview.Flex
	roster   roster.Roster
	hideJIDs bool
}

// Option can be used to configure a new roster widget.
type Option func(*UI)

// ShowJIDs returns an option that shows or hides JIDs in the roster.
func ShowJIDs(show bool) Option {
	s := roster.ShowJIDs(show)
	return func(ui *UI) {
		s(&ui.roster)
	}
}

// New constructs a new UI.
func New(app *tview.Application, opts ...Option) UI {
	statusBar := tview.NewBox().SetBorder(true)
	rosterBox := roster.New(roster.Title("Roster"))

	pages := tview.NewPages()
	logs := tview.NewBox().SetBorder(true).SetTitle("Status")
	logs.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyESC {
			app.SetFocus(rosterBox)
			return nil
		}
		return event
	})
	pages.AddPage("Status", logs, true, true)

	mainFocus := func() {
		app.SetFocus(pages)
	}

	rosterBox.Upsert("[orange]●[white] Thespian", "  me@example.net", mainFocus)
	rosterBox.Upsert("[red]●[white] Twinkletoes", "  cathycathy@example.net", mainFocus)
	rosterBox.Upsert("[green]●[white] Papa Shrimp", "  joooley@example.org", mainFocus)
	rosterBox.Upsert("[silver]●[white] Pockets full of Sunshine", "  pockets@example.com", mainFocus)
	rosterBox.Upsert("Quit", "Exit the application", func() {
		app.Stop()
	})

	ltrFlex := tview.NewFlex().
		AddItem(rosterBox, 25, 1, true).
		AddItem(pages, 0, 1, false)
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ltrFlex, 0, 1, true).
		AddItem(statusBar, 2, 1, false)

	ui := UI{
		roster: rosterBox,
		flex:   flex,
	}
	for _, o := range opts {
		o(&ui)
	}
	return ui
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
