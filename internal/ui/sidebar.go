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
	ui            *UI
	events        *bytes.Buffer
	eventsM       *sync.Mutex
}

// newSidebar creates a new widget with the provided options.
func newSidebar(roster *Roster, b *Bookmarks, c *Conversations, ui *UI) *Sidebar {
	r := &Sidebar{
		pages:         tview.NewPages(),
		dropDown:      tview.NewDropDown().SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor),
		search:        tview.NewInputField(),
		roster:        roster,
		bookmarks:     b,
		conversations: c,
		ui:            ui,
		events:        &bytes.Buffer{},
		eventsM:       &sync.Mutex{},
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
			r.eventsM.Lock()
			defer r.eventsM.Unlock()
			r.events.Reset()
			r.searching = false
			r.search.SetText("")
			r.Flex.RemoveItem(r.search)
		})
	return r
}

// InputHandler implements tview.Primitive for Roster.
func (s *Sidebar) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	if s.searching {
		return s.search.InputHandler()
	}
	return s.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if event == nil {
			return
		}
		switch event.Key() {
		case tcell.KeyESC:
			s.events.Reset()
			return
		case tcell.KeyRune:
		default:
			return
		}

		s.eventsM.Lock()
		defer s.eventsM.Unlock()
		/* #nosec */
		s.events.WriteRune(event.Rune())

		switch event.Rune() {
		case '!':
			s.events.Reset()
			s.ui.PickResource(func(j jid.JID, ok bool) {
				if ok {
					s.ui.ShowLoadCmd(j)
				}
			})
			return
		case 'i':
			s.events.Reset()
			_, item := s.pages.GetFrontPage()
			if item != nil {
				item.InputHandler()(tcell.NewEventKey(tcell.KeyCR, 0, tcell.ModNone), nil)
			}
			return
		case 'I':
			s.ui.ShowRosterInfo()
			return
		case 'o':
			s.events.Reset()
			roster := s.getFrontList()
			if roster == nil {
				return
			}
			for i := 0; i < roster.GetItemCount(); i++ {
				idx := (i + roster.GetCurrentItem()) % roster.GetItemCount()
				main, _ := roster.GetItemText(idx)
				if strings.HasPrefix(main, highlightTag) {
					roster.SetCurrentItem(idx)
					_, item := s.pages.GetFrontPage()
					if item != nil {
						item.InputHandler()(tcell.NewEventKey(tcell.KeyCR, 0, tcell.ModNone), nil)
					}
				}
			}
			idx := (roster.GetCurrentItem() + 1) % roster.GetItemCount()
			roster.SetCurrentItem(idx)
			_, item := s.pages.GetFrontPage()
			if item != nil {
				item.InputHandler()(tcell.NewEventKey(tcell.KeyCR, 0, tcell.ModNone), nil)
			}
		case 'O':
			s.events.Reset()
			roster := s.getFrontList()
			if roster == nil {
				return
			}
			count := roster.GetItemCount()
			currentItem := roster.GetCurrentItem()
			for i := 0; i < count; i++ {
				// Least positive remainder
				idx := ((currentItem-i)%count + count) % count
				main, _ := roster.GetItemText(idx)
				if strings.HasPrefix(main, highlightTag) {
					roster.SetCurrentItem(idx)
					_, item := s.pages.GetFrontPage()
					if item != nil {
						item.InputHandler()(tcell.NewEventKey(tcell.KeyCR, 0, tcell.ModNone), nil)
					}
				}
			}
			idx := ((currentItem-1)%count + count) % count
			roster.SetCurrentItem(idx)
			_, item := s.pages.GetFrontPage()
			if item != nil {
				item.InputHandler()(tcell.NewEventKey(tcell.KeyCR, 0, tcell.ModNone), nil)
			}
		case 'k':
			roster := s.getFrontList()
			if roster == nil {
				return
			}
			if s.events.Len() > 1 {
				n, err := strconv.Atoi(s.events.String()[0 : s.events.Len()-1])
				if err == nil {
					n = roster.GetCurrentItem() - n
					if m := roster.GetItemCount() - 1; n > m {
						n = m
					}
					if n < 0 {
						n = 0
					}
					roster.SetCurrentItem(n)

					s.events.Reset()
					return
				}
			}

			s.events.Reset()
			cur := roster.GetCurrentItem()
			if cur <= 0 {
				return
			}
			roster.SetCurrentItem(cur - 1)
			return
		case 'j':
			roster := s.getFrontList()
			if roster == nil {
				return
			}
			if s.events.Len() > 1 {
				n, err := strconv.Atoi(s.events.String()[0 : s.events.Len()-1])
				if err == nil {
					n = roster.GetCurrentItem() + n
					if m := roster.GetItemCount() - 1; n > m {
						n = m
					}
					if n < 0 {
						n = 0
					}
					roster.SetCurrentItem(n)

					s.events.Reset()
					return
				}
			}

			s.events.Reset()
			cur := roster.GetCurrentItem()
			if cur >= roster.GetItemCount()-1 {
				return
			}
			roster.SetCurrentItem(cur + 1)
			return
		case 'G':
			roster := s.getFrontList()
			if roster == nil {
				return
			}
			roster.SetCurrentItem(roster.GetItemCount() - 1)
		case 'g':
			roster := s.getFrontList()
			if roster == nil {
				return
			}
			if s.events.String() != "gg" {
				return
			}
			s.events.Reset()
			roster.SetCurrentItem(0)
		case 't':
			if s.events.String() != "gt" {
				return
			}
			i, _ := s.dropDown.GetCurrentOption()
			s.dropDown.SetCurrentOption((i + 1) % s.dropDown.GetOptionCount())
		case 'T':
			if s.events.String() != "gT" {
				return
			}
			i, _ := s.dropDown.GetCurrentOption()
			l := s.dropDown.GetOptionCount()
			s.dropDown.SetCurrentOption(((i - 1) + l) % l)
		case 'd':
			if s.events.String() != "dd" {
				return
			}
			_, item := s.pages.GetFrontPage()
			if item == nil {
				return
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
		case '1', '2', '3', '4', '5', '6', '7', '8', '9', '0':
			// Don't reset events, after a number we may provide an action such as
			// '10j'.
			return
		case '/':
			s.searching = true
			s.searchDir = SearchDown
			s.Flex.AddItem(s.search, 1, 0, true)
		case '?':
			s.searching = true
			s.searchDir = SearchUp
			s.Flex.AddItem(s.search, 1, 0, true)
		case 'n':
			if s.lastSearch != "" {
				s.Search(s.lastSearch, s.searchDir)
			}
		case 'N':
			if s.lastSearch != "" {
				s.Search(s.lastSearch, !s.searchDir)
			}
		case 'q':
			s.ui.ShowQuitPrompt()
		case 'K':
			s.ui.ShowHelpPrompt()
		case 'c':
			name, _ := s.pages.GetFrontPage()
			switch name {
			case s.roster.list.GetTitle():
				s.ui.ShowAddRoster()
			case s.bookmarks.list.GetTitle():
				s.ui.ShowAddBookmark()
			}
		default:
			_, item := s.pages.GetFrontPage()
			if item != nil {
				item.InputHandler()(event, setFocus)
			}
		}

		s.events.Reset()
	})
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
