// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package roster contains a tview widget that displays a list of contacts.
package roster

import (
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

// Option can be used to configure a new roster widget.
type Option func(*Roster)

// Title returns an option that sets the rosters title.
func Title(title string) Option {
	return func(r *Roster) {
		r.list.SetTitle(title)
	}
}

// ShowJIDs returns an option that shows or hides JIDs in the roster.
func ShowJIDs(show bool) Option {
	return func(r *Roster) {
		r.list.ShowSecondaryText(show)
	}
}

// Roster is a tview.Primitive that draws a roster pane.
type Roster struct {
	list *tview.List
}

// New creates a new roster widget with the provided options.
func New(opts ...Option) Roster {
	r := Roster{
		list: tview.NewList(),
	}
	r.list.SetBorder(true)
	r.list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() != tcell.KeyRune {
			return event
		}

		switch event.Rune() {
		case 'i':
			return tcell.NewEventKey(tcell.KeyCR, 0, tcell.ModNone)
		case 'k':
			cur := r.list.GetCurrentItem()
			if cur <= 0 {
				return event
			}
			r.list.SetCurrentItem(cur - 1)
			return nil
		case 'j':
			cur := r.list.GetCurrentItem()
			if cur >= r.list.GetItemCount()-1 {
				return event
			}
			r.list.SetCurrentItem(cur + 1)
			return nil
		}

		return event
	})
	for _, o := range opts {
		o(&r)
	}

	// Add default status indicator.
	r.Upsert("Status", "[silver::d]──────", nil)

	return r
}

// Upsert inserts or updates an item in the roster.
func (r Roster) Upsert(name, uid string, action func()) {
	r.list.AddItem(name, uid, 0, action)
}

// Draw implements tview.Primitive for Roster.
func (r Roster) Draw(screen tcell.Screen) {
	r.list.Draw(screen)
}

// GetRect implements tview.Primitive for Roster.
func (r Roster) GetRect() (int, int, int, int) {
	return r.list.GetRect()
}

// SetRect implements tview.Primitive for Roster.
func (r Roster) SetRect(x, y, width, height int) {
	r.list.SetRect(x, y, width, height)
}

// InputHandler implements tview.Primitive for Roster.
func (r Roster) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return r.list.InputHandler()
}

// Focus implements tview.Primitive for Roster.
func (r Roster) Focus(delegate func(p tview.Primitive)) {
	r.list.Focus(delegate)
}

// Blur implements tview.Primitive for Roster.
func (r Roster) Blur() {
	r.list.Blur()
}

// GetFocusable implements tview.Primitive for Roster.
func (r Roster) GetFocusable() tview.Focusable {
	return r.list.GetFocusable()
}
