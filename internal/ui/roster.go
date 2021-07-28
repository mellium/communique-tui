// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"mellium.im/xmpp/roster"
)

// RosterItem represents a contact in the roster.
type RosterItem struct {
	roster.Item
	idx         int
	firstUnread string
}

// FirstUnread returns the ID of the first unread message.
func (r RosterItem) FirstUnread() string {
	return r.firstUnread
}

// Roster is a tview.Primitive that draws a roster pane.
type Roster struct {
	items    map[string]RosterItem
	itemLock *sync.Mutex
	list     *tview.List
	Width    int
}

// newRoster creates a new roster widget with the provided options.
func newRoster(onStatus func()) Roster {
	r := Roster{
		items:    make(map[string]RosterItem),
		itemLock: &sync.Mutex{},
		list:     tview.NewList(),
	}
	r.list.SetTitle("Roster")
	r.list.SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	events := &bytes.Buffer{}
	m := &sync.Mutex{}
	r.list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event == nil || event.Key() != tcell.KeyRune {
			return event
		}

		m.Lock()
		defer m.Unlock()
		/* #nosec */
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

					events.Reset()
					return nil
				}
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

					events.Reset()
					return nil
				}
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

	// Add default status indicator.
	r.Upsert(RosterItem{idx: 0}, onStatus)
	r.Offline()

	return r
}

// Offline sets the state of the roster to show the user as offline.
func (r Roster) Offline() {
	r.setStatus("silver::d", "Offline")
}

// Online sets the state of the roster to show the user as online.
func (r Roster) Online() {
	r.setStatus("green", "Online")
}

// Away sets the state of the roster to show the user as away.
func (r Roster) Away() {
	r.setStatus("orange", "Away")
}

// Busy sets the state of the roster to show the user as busy.
func (r Roster) Busy() {
	r.setStatus("red", "Busy")
}

func (r Roster) setStatus(color, name string) {
	var width int
	if r.Width > 4 {
		width = r.Width - 4
	}
	r.list.SetItemText(0, name, fmt.Sprintf("[%s]%s", color, strings.Repeat("─", width)))
}

// Upsert inserts or updates an item in the roster.
func (r Roster) Upsert(item RosterItem, action func()) {
	r.itemLock.Lock()
	defer r.itemLock.Unlock()

	switch item.Subscription {
	case "remove":
		bare := item.JID.Bare().String()
		var ok bool
		item, ok = r.items[bare]
		if !ok {
			return
		}
		r.list.RemoveItem(item.idx)
		delete(r.items, bare)
	default:
		bare := item.JID.Bare().String()
		existing, ok := r.items[bare]
		if ok {
			// Update the existing roster item.
			r.list.SetItemText(existing.idx, item.Name, bare)
			item.idx = existing.idx
			item.firstUnread = existing.firstUnread
			r.items[bare] = item
			return
		}
		r.list.AddItem(item.Name, bare, 0, action)
		item.idx = r.list.GetItemCount() - 1
		r.items[bare] = item
	}
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

// MouseHandler implements tview.Primitive for Roster.
func (r Roster) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return r.list.MouseHandler()
}

// ShowStatus shows or hides the status line under contacts in the roster.
func (r *Roster) ShowStatus(show bool) {
	r.list.ShowSecondaryText(show)
}

// OnChanged sets a callback for when the user navigates to a roster item.
func (r *Roster) OnChanged(f func(int, string, string, rune)) {
	r.list.SetChangedFunc(f)
}

// SetInputCapture passes calls through to the underlying list view.
func (r Roster) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) *tview.Box {
	return r.list.SetInputCapture(capture)
}

// GetInputCapture returns the input capture function for the underlying list.
func (r Roster) GetInputCapture() func(event *tcell.EventKey) *tcell.EventKey {
	return r.list.GetInputCapture()
}

// GetSelected returns the currently selected roster item.
func (r Roster) GetSelected() (RosterItem, bool) {
	_, j := r.list.GetItemText(r.list.GetCurrentItem())
	return r.GetItem(j)
}

// GetItem returns the item for the given JID.
func (r Roster) GetItem(j string) (RosterItem, bool) {
	r.itemLock.Lock()
	defer r.itemLock.Unlock()

	item, ok := r.items[j]
	return item, ok
}

var highlightTag = fmt.Sprintf("[#%06x::b]", tview.Styles.ContrastSecondaryTextColor.Hex())

// MarkUnread sets the given jid to bold and sets the first message seen after
// the unread marker (unless the unread marker is already set).
func (r Roster) MarkUnread(j, msgID string) bool {
	r.itemLock.Lock()
	defer r.itemLock.Unlock()

	item, ok := r.items[j]
	if !ok {
		return false
	}

	// The unread size is the moment at which the item first became unread, so if
	// it's already set don't chagne it.
	if item.firstUnread == "" {
		item.firstUnread = msgID
		r.items[j] = item
	}

	primary, secondary := r.list.GetItemText(item.idx)
	// If it already has the highlighted prefix, do nothing.
	if strings.HasPrefix(primary, highlightTag) {
		return true
	}
	r.list.SetItemText(item.idx, highlightTag+tview.Escape(primary), secondary)
	return true
}

// MarkRead sets the given jid back to the normal font.
func (r Roster) MarkRead(j string) {
	r.itemLock.Lock()
	defer r.itemLock.Unlock()

	item, ok := r.items[j]
	if !ok {
		return
	}
	item.firstUnread = ""
	r.items[j] = item

	primary, secondary := r.list.GetItemText(item.idx)
	r.list.SetItemText(item.idx, strings.TrimPrefix(primary, highlightTag), secondary)
}

// Unread returns whether the roster item is currently marked as having unread
// messages.
// If no such roster item exists, it returns false.
func (r Roster) Unread(j string) bool {
	r.itemLock.Lock()
	defer r.itemLock.Unlock()

	item, ok := r.items[j]
	if !ok {
		// If it doesn't exist, it's not unread.
		return false
	}
	primary, _ := r.list.GetItemText(item.idx)
	return strings.HasPrefix(primary, highlightTag)
}
