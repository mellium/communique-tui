// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package roster contains a tview widget that displays a list of contacts.
package roster

import (
	"bytes"
	"strconv"
	"sync"

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

	events := &bytes.Buffer{}
	m := &sync.Mutex{}
	r.list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() != tcell.KeyRune {
			return event
		}

		m.Lock()
		defer m.Unlock()
		events.WriteRune(event.Rune())

		// TODO: this is not going to be very maintainable. Figure out a better way
		// to handle keyboard shortcuts.
		switch event.Rune() {
		case 'i':
			events.Reset()
			return tcell.NewEventKey(tcell.KeyCR, 0, tcell.ModNone)
		case 'k':
			if events.Len() > 1 {
				n, err := strconv.Atoi(events.String()[0 : events.Len()-1])
				if err == nil {
					n = r.list.GetCurrentItem() - n
					if m := r.list.GetItemCount() - 1; n > m {
						n = m
					}
					if n < 0 {
						n = 0
					}
					r.list.SetCurrentItem(n)
				}

				events.Reset()
				return nil
			}

			events.Reset()
			cur := r.list.GetCurrentItem()
			if cur <= 0 {
				return event
			}
			r.list.SetCurrentItem(cur - 1)
			return nil
		case 'j':
			if events.Len() > 1 {
				n, err := strconv.Atoi(events.String()[0 : events.Len()-1])
				if err == nil {
					n = r.list.GetCurrentItem() + n
					if m := r.list.GetItemCount() - 1; n > m {
						n = m
					}
					if n < 0 {
						n = 0
					}
					r.list.SetCurrentItem(n)
				}

				events.Reset()
				return nil
			}

			events.Reset()
			cur := r.list.GetCurrentItem()
			if cur >= r.list.GetItemCount()-1 {
				return event
			}
			r.list.SetCurrentItem(cur + 1)
			return nil
		case 'G':
			events.Reset()
			r.list.SetCurrentItem(r.list.GetItemCount() - 1)
			return nil

		case 'g':
			if events.String() == "gg" {
				events.Reset()
				r.list.SetCurrentItem(0)
			}
			return nil
		case '1', '2', '3', '4', '5', '6', '7', '8', '9', '0':
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
