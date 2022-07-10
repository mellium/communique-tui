// Copyright 2022 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"bytes"
	"strconv"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mellium.im/xmpp/jid"
)

// Sidebar is a tview.Primitive that draws a sidebar containing multiple lists
// that can be toggled between using a drop down.
type Sidebar struct {
	*tview.Flex
	Width         int
	dropDown      *tview.DropDown
	pages         *tview.Pages
	search        *tview.InputField
	searching     bool
	lastSearch    string
	searchDir     SearchDir
	roster        *Roster
	bookmarks     *Bookmarks
	conversations *Conversations
}

// newSidebar creates a new widget with the provided options.
func newSidebar(roster *Roster, b *Bookmarks, c *Conversations) *Sidebar {
	r := &Sidebar{
		pages:         tview.NewPages(),
		dropDown:      tview.NewDropDown().SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor),
		search:        tview.NewInputField(),
		roster:        roster,
		bookmarks:     b,
		conversations: c,
	}
	r.pages.AddAndSwitchToPage(r.conversations.list.GetTitle(), r.conversations, true)
	r.pages.AddPage(r.bookmarks.list.GetTitle(), r.bookmarks, true, false)
	r.pages.AddPage(r.roster.list.GetTitle(), r.roster, true, false)
	options := []string{
		r.conversations.list.GetTitle(),
		r.roster.list.GetTitle(),
		r.bookmarks.list.GetTitle(),
	}
	r.dropDown.SetOptions(options, func(name string, _ int) {
		r.pages.SwitchToPage(name)
		r.SetWidth(r.Width)
	})
	r.dropDown.SetCurrentOption(0)

	//r.dropDown.SetTitleAlign(tview.AlignCenter)
	r.Flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(r.dropDown, 1, 1, false).
		AddItem(r.pages, 0, 1, true)

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
			r.Flex.RemoveItem(r.search)
		})

	innerCapture := r.Flex.GetInputCapture()
	r.Flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event == nil {
			return event
		}
		switch event.Key() {
		case tcell.KeyESC:
			events.Reset()
			return event
		case tcell.KeyRune:
		default:
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
			roster := r.getFrontList()
			if roster == nil {
				return event
			}
			for i := 0; i < roster.GetItemCount(); i++ {
				idx := (i + roster.GetCurrentItem()) % roster.GetItemCount()
				main, _ := roster.GetItemText(idx)
				if strings.HasPrefix(main, highlightTag) {
					roster.SetCurrentItem(idx)
					return tcell.NewEventKey(tcell.KeyCR, 0, tcell.ModNone)
				}
			}
			idx := (roster.GetCurrentItem() + 1) % roster.GetItemCount()
			roster.SetCurrentItem(idx)
			return tcell.NewEventKey(tcell.KeyCR, 0, tcell.ModNone)
		case 'O':
			events.Reset()
			roster := r.getFrontList()
			if roster == nil {
				return event
			}
			count := roster.GetItemCount()
			currentItem := roster.GetCurrentItem()
			for i := 0; i < count; i++ {
				// Least positive remainder
				idx := ((currentItem-i)%count + count) % count
				main, _ := roster.GetItemText(idx)
				if strings.HasPrefix(main, highlightTag) {
					roster.SetCurrentItem(idx)
					return tcell.NewEventKey(tcell.KeyCR, 0, tcell.ModNone)
				}
			}
			idx := ((currentItem-1)%count + count) % count
			roster.SetCurrentItem(idx)
			return tcell.NewEventKey(tcell.KeyCR, 0, tcell.ModNone)
		case 'k':
			roster := r.getFrontList()
			if roster == nil {
				return event
			}
			if events.Len() > 1 {
				n, err := strconv.Atoi(events.String()[0 : events.Len()-1])
				if err == nil {
					n = roster.GetCurrentItem() - n
					if m := roster.GetItemCount() - 1; n > m {
						n = m
					}
					if n < 0 {
						n = 0
					}
					roster.SetCurrentItem(n)

					events.Reset()
					return nil
				}
			}

			events.Reset()
			cur := roster.GetCurrentItem()
			if cur <= 0 {
				return event
			}
			roster.SetCurrentItem(cur - 1)
			return nil
		case 'j':
			roster := r.getFrontList()
			if roster == nil {
				return event
			}
			if events.Len() > 1 {
				n, err := strconv.Atoi(events.String()[0 : events.Len()-1])
				if err == nil {
					n = roster.GetCurrentItem() + n
					if m := roster.GetItemCount() - 1; n > m {
						n = m
					}
					if n < 0 {
						n = 0
					}
					roster.SetCurrentItem(n)

					events.Reset()
					return nil
				}
			}

			events.Reset()
			cur := roster.GetCurrentItem()
			if cur >= roster.GetItemCount()-1 {
				return event
			}
			roster.SetCurrentItem(cur + 1)
			return nil
		case 'G':
			events.Reset()
			roster := r.getFrontList()
			if roster == nil {
				return event
			}
			roster.SetCurrentItem(roster.GetItemCount() - 1)
			return nil
		case 'g':
			roster := r.getFrontList()
			if roster == nil {
				return event
			}
			if events.String() == "gg" {
				events.Reset()
				roster.SetCurrentItem(0)
			}
			return nil
		case 't':
			if events.String() == "gt" {
				events.Reset()
				i, _ := r.dropDown.GetCurrentOption()
				r.dropDown.SetCurrentOption((i + 1) % len(options))
			}
			return nil
		case 'T':
			if events.String() == "gT" {
				events.Reset()
				i, _ := r.dropDown.GetCurrentOption()
				r.dropDown.SetCurrentOption(((i - 1) + len(options)) % len(options))
			}
			return nil
		case 'd':
			if events.String() != "dd" {
				return event
			}
			events.Reset()
			_, item := r.pages.GetFrontPage()
			if item == nil {
				return event
			}
			switch i := item.(type) {
			case *Roster:
				i.onDelete()
			case *Bookmarks:
				i.onDelete()
			case *Conversations:
				c, ok := i.GetSelected()
				if !ok {
					break
				}
				i.Delete(c.JID.String())
			}
			return nil
		case '1', '2', '3', '4', '5', '6', '7', '8', '9', '0':
			return nil
		case '/':
			if events.String() == "/" {
				r.searching = true
				r.searchDir = SearchDown
				r.Flex.AddItem(r.search, 1, 0, true)
			}
			return event
		case '?':
			if events.String() == "?" {
				r.searching = true
				r.searchDir = SearchUp
				r.Flex.AddItem(r.search, 1, 0, true)
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

		events.Reset()
		if innerCapture != nil {
			return innerCapture(event)
		}
		return event
	})

	return r
}

// InputHandler implements tview.Primitive for Roster.
func (s *Sidebar) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	if s.searching {
		return s.search.InputHandler()
	}
	return s.Flex.InputHandler()
}

// Search looks forward in the roster trying to find items that match s.
// It is case insensitive and looks in the primary or secondary texts.
// If a match is found after the current selection, we jump to the match,
// wrapping at the end of the list.
func (s *Sidebar) Search(q string, dir SearchDir) bool {
	s.lastSearch = q
	roster := s.getFrontList()
	if roster == nil {
		return false
	}
	items := roster.FindItems(q, q, false, true)
	if len(items) == 0 {
		return false
	}
	if dir == SearchDown {
		for _, item := range items {
			if item > roster.GetCurrentItem() {
				roster.SetCurrentItem(item)
				return true
			}
		}
		roster.SetCurrentItem(items[0])
	} else {
		for i := len(items) - 1; i >= 0; i-- {
			if items[i] < roster.GetCurrentItem() {
				roster.SetCurrentItem(items[i])
				return true
			}
		}
		roster.SetCurrentItem(items[len(items)-1])
	}
	return true
}

func (s *Sidebar) getFrontList() *tview.List {
	_, item := s.pages.GetFrontPage()
	if item == nil {
		return nil
	}
	switch i := item.(type) {
	case *Roster:
		return i.list
	case *Bookmarks:
		return i.list
	case *Conversations:
		return i.list
	}
	return nil
}

// SetWidth sets the width of the sidebar and re-adjusts the dropdown text
// width to match.
func (s *Sidebar) SetWidth(width int) {
	s.Width = width
	s.roster.Width = width
	s.bookmarks.Width = width
	s.conversations.Width = width
	if s.dropDown != nil {
		_, txt := s.dropDown.GetCurrentOption()
		s.dropDown.SetLabelWidth((width / 2) - (len(txt) / 2))
	}
}

// GetSelected returns the currently selected roster item, bookmark, or
// conversation.
func (s *Sidebar) GetSelected() (interface{}, bool) {
	switch name, _ := s.pages.GetFrontPage(); name {
	case s.conversations.list.GetTitle():
		return s.conversations.GetSelected()
	case s.roster.list.GetTitle():
		return s.roster.GetSelected()
	case s.bookmarks.list.GetTitle():
		return s.bookmarks.GetSelected()
	}
	return nil, false
}

// ShowStatus shows or hides the status line under the currently selected list.
func (s *Sidebar) ShowStatus(show bool) {
	s.roster.list.ShowSecondaryText(show)
	s.bookmarks.list.ShowSecondaryText(show)
}

// Offline sets the state of the roster to show the user as offline.
func (s Sidebar) Offline() {
	s.conversations.Offline()
}

// Online sets the state of the roster to show the user as online.
func (s Sidebar) Online() {
	s.conversations.Online()
}

// Away sets the state of the roster to show the user as away.
func (s Sidebar) Away() {
	s.conversations.Away()
}

// Busy sets the state of the roster to show the user as busy.
func (s Sidebar) Busy() {
	s.conversations.Busy()
}

// UpsertPresence updates an existing roster item or bookmark with a newly seen
// resource or presence change.
// If the item is not in any roster, false is returned.
func (s Sidebar) UpsertPresence(j jid.JID, status string) bool {
	rosterOk := s.roster.UpsertPresence(j, status)
	conversationOk := s.conversations.UpsertPresence(j, status)
	return rosterOk || conversationOk
}
