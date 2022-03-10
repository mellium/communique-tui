// Copyright 2022 The Mellium Contributors.
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
)

// Conversation represents an open channel or chat.
type Conversation struct {
	JID         jid.JID
	Name        string
	idx         int
	firstUnread string
	presences   []presence
	Room        bool
}

// FirstUnread returns the ID of the first unread message.
func (c Conversation) FirstUnread() string {
	return c.firstUnread
}

// Conversations is a tview.Primitive that draws the recent/open conversations.
// This pane includes a mix of joined channels and recently updated 1:1 chats.
// It does not necessarily contain all bookmarked channels or all chats from the
// roster (in fact, it may include channels or chats that are not bookmarked or
// not in the roster).
type Conversations struct {
	items    map[string]Conversation
	itemLock *sync.Mutex
	list     *tview.List
	Width    int
	flex     *tview.Flex
	changed  func(int, string, string, rune)
}

// newConversations creates a new widget with the provided options.
func newConversations(onStatus func()) *Conversations {
	c := &Conversations{
		items:    make(map[string]Conversation),
		itemLock: &sync.Mutex{},
		list:     tview.NewList(),
		flex:     tview.NewFlex(),
	}
	c.flex.SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)
	c.flex.AddItem(c.list, 0, 1, true).
		SetDirection(tview.FlexRow)
	c.list.SetTitle("Conversations")

	// Add default status indicator.
	c.Upsert(Conversation{idx: 0}, func(Conversation) { onStatus() })
	c.Offline()

	return c
}

// Offline sets the state of the roster to show the user as offline.
func (c Conversations) Offline() {
	c.setStatus("silver::d", "Offline")
}

// Online sets the state of the roster to show the user as online.
func (c Conversations) Online() {
	c.setStatus("green", "Online")
}

// Away sets the state of the roster to show the user as away.
func (c Conversations) Away() {
	c.setStatus("orange", "Away")
}

// Busy sets the state of the roster to show the user as busy.
func (c Conversations) Busy() {
	c.setStatus("red", "Busy")
}

func (c Conversations) setStatus(color, name string) {
	var width int
	if c.Width > 4 {
		width = c.Width - 4
	}
	c.list.SetItemText(0, name, fmt.Sprintf("[%s]%s", color, strings.Repeat("─", width)))
}

// Delete removes an item from the list.
func (c Conversations) Delete(bareJID string) {
	c.itemLock.Lock()
	defer c.itemLock.Unlock()
	c.deleteItem(bareJID)
}

func (c Conversations) deleteItem(bareJID string) {
	var ok bool
	item, ok := c.items[bareJID]
	if !ok {
		return
	}
	oldIdx := item.idx
	c.list.RemoveItem(oldIdx)
	delete(c.items, bareJID)
	for bareJID, item = range c.items {
		found := c.list.FindItems(item.Name, bareJID, true, false)
		if len(found) == 0 {
			continue
		}
		item.idx = found[0]
		c.items[bareJID] = item
	}
}

// Upsert inserts or updates an item in the list.
func (c Conversations) Upsert(item Conversation, action func(Conversation)) int {
	c.itemLock.Lock()
	defer c.itemLock.Unlock()

	bare := item.JID.Bare().String()
	if item.Name == "" {
		item.Name = item.JID.Localpart()
	}

	existing, ok := c.items[bare]
	if ok {
		// Update the existing roster item.
		c.list.SetItemText(existing.idx, item.Name, bare)
		item.idx = existing.idx
		item.firstUnread = existing.firstUnread
		c.items[bare] = item
		return item.idx
	}
	c.list.AddItem(item.Name, bare, 0, func() { action(item) })
	item.idx = c.list.GetItemCount() - 1
	c.items[bare] = item
	return item.idx
}

// Draw implements tview.Primitive foc Conversations.
func (c Conversations) Draw(screen tcell.Screen) {
	c.flex.Draw(screen)
}

// GetRect implements tview.Primitive foc Conversations.
func (c Conversations) GetRect() (int, int, int, int) {
	return c.flex.GetRect()
}

// SetRect implements tview.Primitive foc Conversations.
func (c Conversations) SetRect(x, y, width, height int) {
	c.flex.SetRect(x, y, width, height)
}

// InputHandler implements tview.Primitive foc Conversations.
func (c Conversations) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return c.flex.InputHandler()
}

// Focus implements tview.Primitive foc Conversations.
func (c Conversations) Focus(delegate func(p tview.Primitive)) {
	if c.changed != nil {
		idx := c.list.GetCurrentItem()
		main, secondary := c.list.GetItemText(idx)
		c.changed(idx, main, secondary, 0)
	}
	c.flex.Focus(delegate)
}

// Blur implements tview.Primitive foc Conversations.
func (c Conversations) Blur() {
	c.flex.Blur()
}

// HasFocus implements tview.Primitive foc Conversations.
func (c Conversations) HasFocus() bool {
	return c.flex.HasFocus()
}

// MouseHandler implements tview.Primitive foc Conversations.
func (c Conversations) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return c.flex.MouseHandler()
}

// ShowStatus shows or hides the status line under contacts in the roster.
func (c *Conversations) ShowStatus(show bool) {
	c.list.ShowSecondaryText(show)
}

// OnChanged sets a callback for when the user navigates to a roster item.
func (c *Conversations) OnChanged(f func(int, string, string, rune)) {
	c.changed = f
	c.list.SetChangedFunc(f)
}

// SetInputCapture passes calls through to the underlying view(s).
func (c Conversations) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) *tview.Box {
	return c.flex.SetInputCapture(capture)
}

// GetInputCapture returns the input capture function for the underlying list.
func (c Conversations) GetInputCapture() func(event *tcell.EventKey) *tcell.EventKey {
	return c.flex.GetInputCapture()
}

// GetSelected returns the currently selected roster item.
func (c Conversations) GetSelected() (Conversation, bool) {
	_, j := c.list.GetItemText(c.list.GetCurrentItem())
	return c.GetItem(j)
}

// UpsertPresence updates an existing roster item with a newly seen resource or
// presence change.
// If the item is not in the roster, false is returned.
func (c Conversations) UpsertPresence(j jid.JID, status string) bool {
	c.itemLock.Lock()
	defer c.itemLock.Unlock()

	key := j.Bare().String()
	item, ok := c.items[key]
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
	c.items[key] = item

	return ok
}

// GetItem returns the item for the given JID.
func (c Conversations) GetItem(j string) (Conversation, bool) {
	c.itemLock.Lock()
	defer c.itemLock.Unlock()

	item, ok := c.items[j]
	return item, ok
}

// MarkUnread sets the given jid to bold and sets the first message seen after
// the unread marker (unless the unread marker is already set).
func (c Conversations) MarkUnread(j, msgID string) bool {
	c.itemLock.Lock()
	defer c.itemLock.Unlock()

	item, ok := c.items[j]
	if !ok {
		return false
	}

	// The unread size is the moment at which the item first became unread, so if
	// it's already set don't chagne it.
	if item.firstUnread == "" {
		item.firstUnread = msgID
		c.items[j] = item
	}

	primary, secondary := c.list.GetItemText(item.idx)
	// If it already has the highlighted prefix, do nothing.
	if strings.HasPrefix(primary, highlightTag) {
		return true
	}
	c.list.SetItemText(item.idx, highlightTag+tview.Escape(primary), secondary)
	return true
}

// MarkRead sets the given jid back to the normal font.
func (c Conversations) MarkRead(j string) {
	c.itemLock.Lock()
	defer c.itemLock.Unlock()

	item, ok := c.items[j]
	if !ok {
		return
	}
	item.firstUnread = ""
	c.items[j] = item

	primary, secondary := c.list.GetItemText(item.idx)
	c.list.SetItemText(item.idx, strings.TrimPrefix(primary, highlightTag), secondary)
}

// Unread returns whether the roster item is currently marked as having unread
// messages.
// If no such roster item exists, it returns false.
func (c Conversations) Unread(j string) bool {
	c.itemLock.Lock()
	defer c.itemLock.Unlock()

	item, ok := c.items[j]
	if !ok {
		// If it doesn't exist, it's not unread.
		return false
	}
	primary, _ := c.list.GetItemText(item.idx)
	return strings.HasPrefix(primary, highlightTag)
}

// Len returns the length of the roster.
func (c *Conversations) Len() int {
	return len(c.items)
}
