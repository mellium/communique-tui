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

// SearchDir the direction of a search.
type SearchDir bool

// Valid search directions.
const (
	SearchUp   SearchDir = true
	SearchDown SearchDir = false
)

// Roster is a tview.Primitive that draws a roster pane.
type Roster struct {
	items      map[string]RosterItem
	itemLock   *sync.Mutex
	list       *tview.List
	Width      int
	search     *tview.InputField
	flex       *tview.Flex
	searching  bool
	lastSearch string
	searchDir  SearchDir
}

// newRoster creates a new roster widget with the provided options.
func newRoster(onStatus func()) *Roster {
	r := &Roster{
		items:    make(map[string]RosterItem),
		itemLock: &sync.Mutex{},
		list:     tview.NewList(),
		search:   tview.NewInputField(),
		flex:     tview.NewFlex(),
	}
	r.flex.SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)
	r.flex.AddItem(r.list, 0, 1, true).
		SetDirection(tview.FlexRow)
	r.list.SetTitle("Roster")

	events := &bytes.Buffer{}
	m := &sync.Mutex{}
	r.search.SetPlaceholder("Search").
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetDoneFunc(func(key tcell.Key) {
			switch key {
			case tcell.KeyTab, tcell.KeyBacktab:
				return
			case tcell.KeyESC:
			case tcell.KeyEnter:
				r.Search(r.search.GetText(), r.searchDir)
			}
			m.Lock()
			defer m.Unlock()
			events.Reset()
			r.searching = false
			r.search.SetText("")
			r.flex.RemoveItem(r.search)
		})
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
		case 'o':
			events.Reset()
			for i := 0; i < r.list.GetItemCount(); i++ {
				idx := (i + r.list.GetCurrentItem()) % r.list.GetItemCount()
				main, _ := r.list.GetItemText(idx)
				if strings.HasPrefix(main, highlightTag) {
					r.list.SetCurrentItem(idx)
					return tcell.NewEventKey(tcell.KeyCR, 0, tcell.ModNone)
				}
			}
			idx := (r.list.GetCurrentItem() + 1) % r.list.GetItemCount()
			r.list.SetCurrentItem(idx)
			return tcell.NewEventKey(tcell.KeyCR, 0, tcell.ModNone)
		case 'O':
			events.Reset()
			// -1 because we ignore the online indicator
			count := r.list.GetItemCount()
			currentItem := r.list.GetCurrentItem()
			for i := 0; i < count; i++ {
				// Least positive remainder
				idx := ((currentItem-i)%count + count) % count
				main, _ := r.list.GetItemText(idx)
				if strings.HasPrefix(main, highlightTag) {
					r.list.SetCurrentItem(idx)
					return tcell.NewEventKey(tcell.KeyCR, 0, tcell.ModNone)
				}
			}
			idx := ((currentItem-1)%count + count) % count
			r.list.SetCurrentItem(idx)
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
		case '/':
			if events.String() == "/" {
				r.searching = true
				r.searchDir = SearchDown
				r.flex.AddItem(r.search, 1, 0, true)
			}
			return event
		case '?':
			if events.String() == "?" {
				r.searching = true
				r.searchDir = SearchUp
				r.flex.AddItem(r.search, 1, 0, true)
			}
			return event
		case 'n':
			events.Reset()
			if r.lastSearch != "" {
				r.Search(r.lastSearch, r.searchDir)
			}
			return nil
		case 'N':
			events.Reset()
			if r.lastSearch != "" {
				r.Search(r.lastSearch, !r.searchDir)
			}
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

	bare := item.JID.Bare().String()
	if item.Name == "" {
		item.Name = item.JID.Localpart()
	}

	switch item.Subscription {
	case "remove":
		var ok bool
		item, ok = r.items[bare]
		if !ok {
			return
		}
		r.list.RemoveItem(item.idx)
		delete(r.items, bare)
	default:
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
	if r.searching {
		return r.search.InputHandler()
	}
	return r.flex.InputHandler()
}

// Focus implements tview.Primitive for Roster.
func (r Roster) Focus(delegate func(p tview.Primitive)) {
	r.flex.Focus(delegate)
}

// Blur implements tview.Primitive for Roster.
func (r Roster) Blur() {
	r.flex.Blur()
}

// GetFocusable implements tview.Primitive for Roster.
func (r Roster) GetFocusable() tview.Focusable {
	return r.flex.GetFocusable()
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

// Search looks forward in the roster trying to find items that match s.
// It is case insensitive and looks in the primary or secondary texts.
// If a match is found after the current selection, we jump to the match,
// wrapping at the end of the list.
func (r *Roster) Search(s string, dir SearchDir) bool {
	r.lastSearch = s
	items := r.list.FindItems(s, s, false, true)
	if len(items) == 0 {
		return false
	}
	if dir == SearchDown {
		for _, item := range items {
			if item > r.list.GetCurrentItem() {
				r.list.SetCurrentItem(item)
				return true
			}
		}
		r.list.SetCurrentItem(items[0])
	} else {
		for i := len(items) - 1; i >= 0; i-- {
			if items[i] < r.list.GetCurrentItem() {
				r.list.SetCurrentItem(items[i])
				return true
			}
		}
		r.list.SetCurrentItem(items[len(items)-1])
	}
	return true
}
