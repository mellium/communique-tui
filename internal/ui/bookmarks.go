// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"sync"

	"mellium.im/xmpp/bookmarks"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// BookmarkItem represents a bookmark in the list.
type BookmarkItem struct {
	bookmarks.Channel
	idx int
}

// Bookmarks is a tview.Primitive that draws a list of bookmarks.
type Bookmarks struct {
	items    map[string]BookmarkItem
	itemLock *sync.Mutex
	list     *tview.List
	Width    int
	flex     *tview.Flex
	onDelete func()
	changed  func(int, string, string, rune)
}

// newBookmarks creates a new bookmarks widget with the provided options.
func newBookmarks(onDelete func()) *Bookmarks {
	r := &Bookmarks{
		items:    make(map[string]BookmarkItem),
		itemLock: &sync.Mutex{},
		list:     tview.NewList(),
		flex:     tview.NewFlex(),
		onDelete: onDelete,
	}
	r.flex.SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)
	r.flex.AddItem(r.list, 0, 1, true).
		SetDirection(tview.FlexRow)
	r.list.SetTitle("Channels")

	return r
}

// Delete removes a bookmark from the list.
func (b Bookmarks) Delete(bareJID string) {
	b.itemLock.Lock()
	defer b.itemLock.Unlock()
	b.deleteItem(bareJID)
}

func (b Bookmarks) deleteItem(bareJID string) {
	var ok bool
	item, ok := b.items[bareJID]
	if !ok {
		return
	}
	oldIdx := item.idx
	b.list.RemoveItem(oldIdx)
	delete(b.items, bareJID)
	for bareJID, item = range b.items {
		found := b.list.FindItems(item.Name, bareJID, true, false)
		if len(found) == 0 {
			continue
		}
		item.idx = found[0]
		b.items[bareJID] = item
	}
}

// Upsert inserts or updates a bookmark.
func (b Bookmarks) Upsert(bookmark bookmarks.Channel, action func()) {
	b.itemLock.Lock()
	defer b.itemLock.Unlock()

	item := BookmarkItem{
		Channel: bookmark,
	}

	bare := item.JID.Bare().String()
	if item.Name == "" {
		item.Name = item.JID.Localpart()
	}

	existing, ok := b.items[bare]
	if ok {
		// Update the existing bookmark.
		b.list.SetItemText(existing.idx, item.Name, bare)
		item.idx = existing.idx
		b.items[bare] = item
		return
	}
	b.list.AddItem(item.Name, bare, 0, action)
	item.idx = b.list.GetItemCount() - 1
	b.items[bare] = item
}

// Draw implements tview.Primitive.
func (b Bookmarks) Draw(screen tcell.Screen) {
	b.flex.Draw(screen)
}

// GetRect implements tview.Primitive.
func (b Bookmarks) GetRect() (int, int, int, int) {
	return b.flex.GetRect()
}

// SetRect implements tview.Primitive.
func (b Bookmarks) SetRect(x, y, width, height int) {
	b.flex.SetRect(x, y, width, height)
}

// InputHandler implements tview.Primitive.
func (b Bookmarks) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return b.flex.InputHandler()
}

// Focus implements tview.Primitive.
func (b Bookmarks) Focus(delegate func(p tview.Primitive)) {
	if b.changed != nil && b.list.GetItemCount() > 0 {
		idx := b.list.GetCurrentItem()
		main, secondary := b.list.GetItemText(idx)
		b.changed(idx, main, secondary, 0)
	}
	b.flex.Focus(delegate)
}

// Blur implements tview.Primitive.
func (b Bookmarks) Blur() {
	b.flex.Blur()
}

// HasFocus implements tview.Primitive.
func (b Bookmarks) HasFocus() bool {
	return b.flex.HasFocus()
}

// MouseHandler implements tview.Primitive.
func (b Bookmarks) MouseHandler() func(tview.MouseAction, *tcell.EventMouse, func(tview.Primitive)) (bool, tview.Primitive) {
	return b.flex.MouseHandler()
}

// ShowStatus shows or hides the status line under bookmarks in the list.
func (b Bookmarks) ShowStatus(show bool) {
	b.list.ShowSecondaryText(show)
}

// SetInputCapture passes calls through to the underlying view(s).
func (b Bookmarks) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) *tview.Box {
	return b.flex.SetInputCapture(capture)
}

// GetInputCapture returns the input capture function for the underlying list.
func (b Bookmarks) GetInputCapture() func(event *tcell.EventKey) *tcell.EventKey {
	return b.flex.GetInputCapture()
}

// GetSelected returns the currently selected bookmark.
func (b Bookmarks) GetSelected() (BookmarkItem, bool) {
	_, j := b.list.GetItemText(b.list.GetCurrentItem())
	return b.GetItem(j)
}

// GetItem returns the item for the given JID.
func (b Bookmarks) GetItem(j string) (BookmarkItem, bool) {
	b.itemLock.Lock()
	defer b.itemLock.Unlock()

	item, ok := b.items[j]
	return item, ok
}

// Len returns the length of the list.
func (b Bookmarks) Len() int {
	return len(b.items)
}

// OnChanged sets a callback for when the user navigates to a bookmark.
func (b *Bookmarks) OnChanged(f func(int, string, string, rune)) {
	b.changed = f
	b.list.SetChangedFunc(f)
}
