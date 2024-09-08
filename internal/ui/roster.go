// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/roster"
)

type presence struct {
	From   jid.JID
	Status string
}

// RosterItem represents a contact in the roster.
type RosterItem struct {
	roster.Item
	idx         int
	firstUnread string
	presences   []presence
}

// FirstUnread returns the ID of the first unread message.
func (r RosterItem) FirstUnread() string {
	return r.firstUnread
}

// SearchDir the direction of a search.
type SearchDir bool

// Valid search directions.
const (
	SearchUp   SearchDir = true
	SearchDown SearchDir = false
)

// Roster is a tview.Primitive that draws a roster pane.
type Roster struct {
	items    map[string]RosterItem
	itemLock *sync.Mutex
	list     *tview.List
	Width    int
	flex     *tview.Flex
	onDelete func()
	changed  func(int, string, string, rune)
}

// newRoster creates a new roster widget with the provided options.
func newRoster(onDelete func()) *Roster {
	r := &Roster{
		items:    make(map[string]RosterItem),
		itemLock: &sync.Mutex{},
		list:     tview.NewList(),
		flex:     tview.NewFlex(),
		onDelete: onDelete,
	}
	r.flex.SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)
	r.flex.AddItem(r.list, 0, 1, true).
		SetDirection(tview.FlexRow)
	r.list.SetTitle("Roster")

	return r
}

// Delete removes an item from the roster.
func (r Roster) Delete(bareJID string) {
	r.itemLock.Lock()
	defer r.itemLock.Unlock()
	r.deleteItem(bareJID)
}

func (r Roster) deleteItem(bareJID string) {
	var ok bool
	item, ok := r.items[bareJID]
	if !ok {
		return
	}
	oldIdx := item.idx
	r.list.RemoveItem(oldIdx)
	delete(r.items, bareJID)
	for bareJID, item = range r.items {
		found := r.list.FindItems(item.Name, bareJID, true, false)
		if len(found) == 0 {
			continue
		}
		item.idx = found[0]
		r.items[bareJID] = item
	}
}

// Upsert inserts or updates an item in the roster.
func (r Roster) Upsert(item RosterItem, action func()) {
	r.itemLock.Lock()
	defer r.itemLock.Unlock()

	bare := item.JID.Bare().String()
	if item.Name == "" {
		item.Name = item.JID.Localpart()
	}

	if item.Subscription == "remove" {
		r.deleteItem(bare)
		return
	}

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

// Draw implements tview.Primitive for Roster.
func (r Roster) Draw(screen tcell.Screen) {
	r.flex.Draw(screen)
}

// GetRect implements tview.Primitive for Roster.
func (r Roster) GetRect() (int, int, int, int) {
	return r.flex.GetRect()
}

// SetRect implements tview.Primitive for Roster.
func (r Roster) SetRect(x, y, width, height int) {
	r.flex.SetRect(x, y, width, height)
}

// InputHandler implements tview.Primitive for Roster.
func (r Roster) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return r.flex.InputHandler()
}

// Focus implements tview.Primitive for Roster.
func (r Roster) Focus(delegate func(p tview.Primitive)) {
	if r.changed != nil && r.list.GetItemCount() > 0 {
		idx := r.list.GetCurrentItem()
		main, secondary := r.list.GetItemText(idx)
		r.changed(idx, main, secondary, 0)
	}
	r.flex.Focus(delegate)
}

// Blur implements tview.Primitive for Roster.
func (r Roster) Blur() {
	r.flex.Blur()
}

// HasFocus implements tview.Primitive for Roster.
func (r Roster) HasFocus() bool {
	return r.flex.HasFocus()
}

// MouseHandler implements tview.Primitive for Roster.
func (r Roster) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return r.flex.MouseHandler()
}

// ShowStatus shows or hides the status line under contacts in the roster.
func (r *Roster) ShowStatus(show bool) {
	r.list.ShowSecondaryText(show)
}

// OnChanged sets a callback for when the user navigates to a roster item.
func (r *Roster) OnChanged(f func(int, string, string, rune)) {
	r.changed = f
	r.list.SetChangedFunc(f)
}

// SetInputCapture passes calls through to the underlying view(s).
func (r Roster) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) *tview.Box {
	return r.flex.SetInputCapture(capture)
}

// GetInputCapture returns the input capture function for the underlying list.
func (r Roster) GetInputCapture() func(event *tcell.EventKey) *tcell.EventKey {
	return r.flex.GetInputCapture()
}

// GetSelected returns the currently selected roster item.
func (r Roster) GetSelected() (RosterItem, bool) {
	_, j := r.list.GetItemText(r.list.GetCurrentItem())
	return r.GetItem(j)
}

// UpsertPresence updates an existing roster item with a newly seen resource or
// presence change.
// If the item is not in the roster, false is returned.
func (r Roster) UpsertPresence(j jid.JID, status string) bool {
	r.itemLock.Lock()
	defer r.itemLock.Unlock()

	key := j.Bare().String()
	item, ok := r.items[key]
	if !ok {
		return ok
	}
	var found bool
	filtered := item.presences[:0]
	for _, p := range item.presences {
		if !p.From.Equal(j) {
			filtered = append(filtered, p)
			continue
		}
		found = true
		if status == statusOffline {
			continue
		}
		p.Status = status
		filtered = append(filtered, p)
	}
	item.presences = filtered
	if !found {
		item.presences = append(item.presences, presence{
			From:   j,
			Status: status,
		})
	}
	r.items[key] = item

	return ok
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

// Len returns the length of the roster.
func (r *Roster) Len() int {
	return len(r.items)
}

// PasteHandler implements tview.Primitive.
func (Roster) PasteHandler() func(string, func(tview.Primitive)) {
	return nil
}
