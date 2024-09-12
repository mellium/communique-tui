// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package ui ties together various widgets to create the main Communiqué UI.
package ui // import "mellium.im/communique/internal/ui"

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"text/tabwriter"
	"text/template"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/text/message"

	"mellium.im/communique/internal/client/event"
	"mellium.im/xmpp/bookmarks"
	"mellium.im/xmpp/commands"
	"mellium.im/xmpp/form"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/roster"
)

const (
	commandsLabel       = "Commands"
	getPasswordPageName = "get_password"
	logsPageName        = "logs"
	chatPageName        = "chat"
	helpPageName        = "help"
	delRosterPageName   = "del_roster"
	delBookmarkPageName = "del_bookmark"
	cmdPageName         = "list_cmd"
	infoPageName        = "info"
	setStatusPageName   = "set_status"
	uiPageName          = "ui"

	statusOnline  = "online"
	statusOffline = "offline"
	statusAway    = "away"
	statusBusy    = "busy"
)

type syncBool struct {
	b bool
	m sync.Mutex
}

func (b *syncBool) Set(v bool) {
	b.m.Lock()
	defer b.m.Unlock()
	b.b = v
}

func (b *syncBool) Get() bool {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b
}

// UI is a widget that combines other widgets to make the main UI.
type UI struct {
	app          *tview.Application
	flex         *tview.Flex
	pages        *tview.Pages
	buffers      *tview.Pages
	history      *ConversationView
	statusBar    *tview.TextView
	sidebar      *Sidebar
	sidebarWidth int
	logWriter    *tview.TextView
	handler      func(interface{})
	redraw       func() *tview.Application
	addr         string
	passPrompt   chan string
	chatsOpen    *syncBool
	cmdPane      *commandsPane
	debug        *log.Logger
	p            *message.Printer
}

// Printer returns the message printer that the UI is using for translations.
func (ui *UI) Printer() *message.Printer {
	return ui.p
}

// Run starts the application event loop.
func (ui *UI) Run() error {
	ui.logWriter.SetChangedFunc(func() {
		ui.app.Draw()
	})

	return ui.app.SetRoot(ui.pages, true).SetFocus(ui.pages).Run()
}

// Stop stops the application, causing Run() to return.
func (ui *UI) Stop() {
	ui.app.Stop()
}

// Option can be used to configure a new roster widget.
type Option func(*UI)

// ShowStatus returns an option that shows or hides the status line under
// contacts in the roster.
func ShowStatus(show bool) Option {
	return func(ui *UI) {
		ui.sidebar.ShowStatus(show)
	}
}

// Addr returns an option that sets the users address anywhere that it is
// displayed in the UI.
func Addr(addr string) Option {
	return func(ui *UI) {
		ui.addr = addr
	}
}

// Debug sets the verbose debug logger that will be used by the UI.
func Debug(l *log.Logger) Option {
	return func(ui *UI) {
		ui.debug = l
	}
}

// Handle returns an option that configures an event handler which will be
// called when the user performs certain actions in the UI.
// Only one event handler can be registered, and subsequent calls to Handle will
// replace the handler.
// The function will be called synchronously on the UI goroutine, so don't do
// any intensive work (or, if you must, launch a new goroutine).
func Handle(handler func(event interface{})) Option {
	return func(ui *UI) {
		ui.handler = handler
	}
}

// RosterWidth returns an option that sets the width of the roster.
// It accepts a minimum of 2 and a max of 50 the default is 25.
func RosterWidth(width int) Option {
	return func(ui *UI) {
		if width == 0 {
			ui.sidebar.SetWidth(25)
			ui.sidebarWidth = 25
			return
		}
		if width < 2 {
			ui.sidebar.SetWidth(2)
			ui.sidebarWidth = 2
			return
		}
		if width > 50 {
			ui.sidebar.SetWidth(50)
			ui.sidebarWidth = 50
			return
		}
		ui.sidebar.SetWidth(width)
		ui.sidebarWidth = width
	}
}

// New constructs a new UI.
func New(p *message.Printer, opts ...Option) *UI {
	app := tview.NewApplication()
	statusBar := tview.NewTextView()
	statusBar.
		SetTextColor(tview.Styles.PrimaryTextColor).
		SetBackgroundColor(tview.Styles.MoreContrastBackgroundColor).
		SetBorder(false).
		SetBorderPadding(0, 0, 2, 0)
	buffers := tview.NewPages()
	pages := tview.NewPages()

	ui := &UI{
		app:          app,
		sidebarWidth: 25,
		statusBar:    statusBar,
		handler:      func(interface{}) {},
		redraw:       app.Draw,
		buffers:      buffers,
		pages:        pages,
		passPrompt:   make(chan string),
		chatsOpen:    &syncBool{},
		debug:        log.New(io.Discard, "", 0),
		p:            p,
	}
	sidebarBox := newSidebar(ui)
	ui.sidebar = sidebarBox
	ui.cmdPane = cmdPane()
	for _, o := range opts {
		o(ui)
	}

	app.SetInputCapture(ui.handleInput)

	chats := NewConversationView(ui)
	ui.history = chats
	buffers.AddPage(chatPageName, chats, true, false)

	logs := newLogs(p, app)
	buffers.AddPage(logsPageName, logs, true, true)
	ui.logWriter = logs

	setStatusPage := statusModal(func(buttonIndex int, buttonLabel string) {
		switch buttonIndex {
		case 0:
			ui.handler(event.StatusOnline{})
		case 1:
			ui.handler(event.StatusAway{})
		case 2:
			ui.handler(event.StatusBusy{})
		case 3:
			ui.handler(event.StatusOffline{})
		}
		ui.pages.HidePage(setStatusPageName)
	})

	getPasswordPage := passwordModal(ui.addr, func(getPasswordPage *tview.Form) {
		ui.passPrompt <- getPasswordPage.GetFormItem(0).(*tview.InputField).GetText()
		ui.pages.HidePage(getPasswordPageName)
	})

	ltrFlex := tview.NewFlex().
		AddItem(sidebarBox, ui.sidebarWidth, 1, true).
		AddItem(buffers, 0, 1, false)
	ui.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ltrFlex, 0, 1, true).
		AddItem(statusBar, 1, 1, false)

	ui.pages.AddPage(setStatusPageName, setStatusPage, true, false)
	ui.pages.AddPage(uiPageName, ui.flex, true, true)
	buffers.AddPage(cmdPageName, ui.cmdPane, true, false)
	ui.pages.AddPage(delRosterPageName, delRosterModal(p, func() {
		ui.pages.HidePage(delRosterPageName)
	}, func() {
		cur := ui.sidebar.roster.list.GetCurrentItem()
		for _, item := range ui.sidebar.roster.items {
			if item.idx == cur {
				ui.handler(event.DeleteRosterItem(item.Item))
				break
			}
		}
	}), true, false)
	ui.pages.AddPage(delBookmarkPageName, delBookmarkModal(p, func() {
		ui.pages.HidePage(delBookmarkPageName)
	}, func() {
		cur := ui.sidebar.bookmarks.list.GetCurrentItem()
		for _, item := range ui.sidebar.bookmarks.items {
			if item.idx == cur {
				ui.handler(event.DeleteBookmark(item.Channel))
				ui.sidebar.bookmarks.Delete(item.Channel.JID.Bare().String())
				break
			}
		}
	}), true, false)

	ui.pages.AddPage(getPasswordPageName, getPasswordPage, true, false)

	return ui
}

// RosterLen returns the length of the currently visible roster.
func (ui *UI) RosterLen() int {
	roster := ui.sidebar.getFrontList()
	if roster == nil {
		return 0
	}
	return roster.GetItemCount()
}

// UpdateRoster adds an item to the roster.
func (ui *UI) UpdateRoster(item RosterItem) {
	ui.sidebar.roster.Upsert(item, func() {
		selected := func(c Conversation) {
			ui.buffers.SwitchToPage(chatPageName)
			ui.chatsOpen.Set(true)
			ui.handler(event.OpenChat(item.Item))
			ui.app.SetFocus(ui.buffers)
		}
		c := Conversation{
			JID:         item.JID,
			Name:        item.Name,
			firstUnread: item.firstUnread,
			presences:   item.presences,
		}
		idx := ui.sidebar.conversations.Upsert(c, selected)
		ui.sidebar.conversations.list.SetCurrentItem(idx)
		ui.sidebar.dropDown.SetCurrentOption(0)
		selected(c)
	})
	ui.redraw()
}

// UpdateConversations adds a roster item to the recent conversations list.
func (ui *UI) UpdateConversations(c Conversation) {
	ui.sidebar.conversations.Upsert(c, func(c Conversation) {
		ui.buffers.SwitchToPage(chatPageName)
		ui.chatsOpen.Set(true)
		ui.handler(event.OpenChat(roster.Item{
			JID:  c.JID,
			Name: c.Name,
		}))
		ui.app.SetFocus(ui.buffers)
	})
	ui.redraw()
}

// UpdateBookmarks adds an item to the bookmarks sidebar.
func (ui *UI) UpdateBookmarks(item bookmarks.Channel) {
	ui.handler(event.UpdateBookmark(item))
	ui.sidebar.bookmarks.Upsert(item, func() {
		selected := func(c Conversation) {
			ui.buffers.SwitchToPage(chatPageName)
			ui.chatsOpen.Set(true)
			ui.handler(event.OpenChat(roster.Item{
				JID:  item.JID,
				Name: item.Name,
			}))
			ui.app.SetFocus(ui.buffers)
		}
		c := Conversation{
			JID:  item.JID,
			Name: item.Name,
			Room: true,
		}
		idx := ui.sidebar.conversations.Upsert(c, selected)
		ui.sidebar.conversations.list.SetCurrentItem(idx)
		ui.sidebar.dropDown.SetCurrentOption(0)
		selected(c)
		ui.app.SetFocus(ui.buffers)
		ui.handler(event.OpenChannel(item))
		ui.handler(event.OpenChat(roster.Item{
			JID:  item.JID,
			Name: item.Name,
		}))
	})
	ui.redraw()
}

// Write writes to the logging text view.
func (ui *UI) Write(p []byte) (n int, err error) {
	return ui.logWriter.Write(p)
}

// Roster returns the underlying roster pane widget.
func (ui *UI) Roster() *Roster {
	return ui.sidebar.roster
}

// Bookmarks returns the underlying bookmark pane widget.
func (ui *UI) Bookmarks() *Bookmarks {
	return ui.sidebar.bookmarks
}

// Conversations returns the recent conversations widget.
func (ui *UI) Conversations() *Conversations {
	return ui.sidebar.conversations
}

// ChatsOpen returns true if the chat pane is open.
func (ui *UI) ChatsOpen() bool {
	return ui.chatsOpen.Get()
}

// Offline sets the state of the roster to show the user as offline.
func (ui *UI) Offline(j jid.JID, self bool) {
	if self {
		ui.sidebar.Offline()
		ui.redraw()
	}
	ui.sidebar.UpsertPresence(j, statusOffline)
}

// Online sets the state of the roster to show the user as online.
func (ui *UI) Online(j jid.JID, self bool) {
	if self {
		ui.sidebar.Online()
		ui.redraw()
	}
	ui.sidebar.UpsertPresence(j, statusOnline)
}

// Away sets the state of the roster to show the user as away.
func (ui *UI) Away(j jid.JID, self bool) {
	if self {
		ui.sidebar.Away()
		ui.redraw()
	}
	ui.sidebar.UpsertPresence(j, statusAway)
}

// Busy sets the state of the roster to show the user as busy.
func (ui *UI) Busy(j jid.JID, self bool) {
	if self {
		ui.sidebar.Busy()
		ui.redraw()
	}
	ui.sidebar.UpsertPresence(j, statusBusy)
}

// Handle configures an event handler which will be called when the user
// performs certain actions in the UI.
// Only one event handler can be registered, and subsequent calls to Handle will
// replace the handler.
// The function will be called synchronously on the UI goroutine, so don't do
// any intensive work (or launch a new goroutine if you must).
func (ui *UI) Handle(handler func(interface{})) {
	ui.handler = handler
}

// ShowPasswordPrompt displays a modal and blocks until the user enters a
// password and submits it.
func (ui *UI) ShowPasswordPrompt() string {
	ui.pages.ShowPage(getPasswordPageName)
	ui.pages.SendToFront(getPasswordPageName)
	ui.app.SetFocus(ui.pages)
	return <-ui.passPrompt
}

// ShowQuitPrompt asks if the user wants to quit the application.
func (ui *UI) ShowQuitPrompt() {
	p := ui.Printer()
	const quitPageName = "quit"
	done := func(buttonIndex int, _ string) {
		if buttonIndex == 0 {
			ui.Stop()
		}
		ui.pages.HidePage(quitPageName)
		ui.pages.RemovePage(quitPageName)
	}
	quitModal := NewModal().
		SetText(p.Sprintf("Are you sure you want to quit?")).
		AddButtons([]string{p.Sprintf("Quit"), p.Sprintf("Cancel")}).
		SetDoneFunc(done)
	quitModal.SetInputCapture(modalClose(func() {
		done(-1, "")
	}))
	ui.pages.AddPage(quitPageName, quitModal, true, false)
	ui.pages.ShowPage(quitPageName)
	ui.pages.SendToFront(quitPageName)
	ui.app.SetFocus(ui.pages)
}

// ShowAddBookmark asks the user for a new JID.
func (ui *UI) ShowAddBookmark() {
	const (
		pageName = "add_bookmark"
	)
	p := ui.Printer()
	var addButton = p.Sprintf("Join")
	// Autocomplete rooms that are joined in the chats list but that we don't have
	// bookmarks for and autocomplete domains of existing bookmarks.
	l := len(ui.sidebar.bookmarks.items) + len(ui.sidebar.conversations.items)
	autocomplete := make([]jid.JID, 0, l)
	for _, item := range ui.sidebar.bookmarks.items {
		autocomplete = append(autocomplete, item.JID.Domain())
	}
	for _, item := range ui.sidebar.conversations.items {
		if !item.Room {
			continue
		}
		bare := item.JID.Bare()
		if _, ok := ui.sidebar.bookmarks.items[bare.String()]; ok {
			continue
		}
		autocomplete = append(autocomplete, bare)
	}
	mod := getJID(p, p.Sprintf("Join Channel"), addButton, false, func(j jid.JID, buttonLabel string) {
		if buttonLabel == addButton {
			go func() {
				ui.UpdateBookmarks(bookmarks.Channel{
					JID: j.Bare(),
				})
			}()
		}
		ui.pages.HidePage(pageName)
		ui.pages.RemovePage(pageName)
	}, autocomplete)

	ui.pages.AddPage(pageName, mod, true, true)
	ui.pages.ShowPage(pageName)
	ui.pages.SendToFront(pageName)
	ui.app.SetFocus(ui.pages)
}

// ShowAddRoster asks the user for a new JID.
func (ui *UI) ShowAddRoster() {
	const (
		pageName = "add_roster"
	)
	p := ui.Printer()
	addButton := p.Sprintf("Add")

	// Autocomplete using bare JIDs in the conversations list that aren't already
	// in the roster, and domains from the roster list in case we're adding
	// another contact at the same domain.
	l := len(ui.sidebar.roster.items) + len(ui.sidebar.conversations.items)
	autocomplete := make([]jid.JID, 0, l)
	for _, item := range ui.sidebar.conversations.items {
		bare := item.JID.Bare()
		if _, ok := ui.sidebar.roster.items[bare.String()]; ok {
			continue
		}
		autocomplete = append(autocomplete, bare)
	}
	mod := addRoster(p, addButton, autocomplete, func(v addRosterForm, buttonLabel string) {
		if buttonLabel == addButton {
			ui.handler(event.Subscribe(v.addr.Bare()))
			ev := event.UpdateRoster{
				Item: roster.Item{
					JID:  v.addr.Bare(),
					Name: v.nick,
				},
			}
			ui.handler(ev)
		}
		ui.pages.HidePage(pageName)
		ui.pages.RemovePage(pageName)
	})

	ui.pages.AddPage(pageName, mod, true, true)
	ui.pages.ShowPage(pageName)
	ui.pages.SendToFront(pageName)
	ui.app.SetFocus(ui.pages)
}

// ShowLoadCmd shows available ad-hoc commands for the selected JID.
func (ui *UI) ShowLoadCmd(j jid.JID) {
	p := ui.Printer()
	cancelButton := p.Sprintf("Cancel")
	ui.cmdPane.Form().SetButtonsAlign(tview.AlignLeft)
	ui.cmdPane.SetText(p.Sprintf("Commands"), p.Sprintf("Loading commands…"))
	ui.cmdPane.Form().Clear(true).
		AddButton(cancelButton, func() {
			ui.SelectRoster()
		})
	ui.buffers.SwitchToPage(cmdPageName)
	ui.app.SetFocus(ui.buffers)
	ui.handler(event.LoadingCommands(j))
}

// ShowForm displays an ad-hoc commands form.
func (ui *UI) ShowForm(formData *form.Data, buttons []string, onDone func(string)) {
	defer func() {
		ui.buffers.SwitchToPage(cmdPageName)
		ui.app.SetFocus(ui.buffers)
		ui.Redraw()
	}()
	ui.cmdPane.Form().SetButtonsAlign(tview.AlignLeft)
	title := "Data Form"
	if t := formData.Title(); t != "" {
		title = t
	}
	ui.cmdPane.SetText(title, formData.Instructions())
	box := ui.cmdPane.Form().Clear(true)
	formData.ForFields(func(field form.FieldData) {
		switch field.Type {
		case form.TypeBoolean:
			// TODO: changed func/required
			def, _ := formData.GetBool(field.Var)
			box.AddCheckbox(field.Label, def, func(checked bool) {
				_, err := formData.Set(field.Var, checked)
				if err != nil {
					ui.debug.Printf("error setting bool form field %s: %v", field.Var, err)
				}
			})
		case form.TypeFixed:
			// TODO: rewrap text to some reasonable length first.
			if field.Label != "" {
				for _, line := range strings.Split(field.Label, "\n") {
					box.AddFormItem(tview.NewTextView().SetText(line))
				}
			}
			for _, val := range field.Raw {
				for _, line := range strings.Split(val, "\n") {
					box.AddFormItem(tview.NewTextView().SetText(line))
				}
			}
			// TODO: will this just work? it's on the form already right?
		//case form.TypeHidden:
		//box.AddButton("Hidden: "+field.Label, nil)
		case form.TypeJIDMulti:
			jids, _ := formData.GetJIDs(field.Var)
			opts := make([]string, len(jids), 0)
			for _, j := range jids {
				opts = append(opts, j.String())
			}
			box.AddDropDown(field.Label, opts, 0, func(option string, optionIndex int) {
				j, err := jid.Parse(option)
				if err != nil {
					ui.debug.Printf("error parsing jid-multi value for field %s: %v", field.Var, err)
					return
				}
				_, err = formData.Set(field.Var, j)
				if err != nil {
					ui.debug.Printf("error setting jid-multi form field %s: %v", field.Var, err)
				}
			})
		case form.TypeJID:
			j, _ := formData.GetJID(field.Var)
			box.AddInputField(field.Label, j.String(), 20, func(textToCheck string, _ rune) bool {
				_, err := jid.Parse(textToCheck)
				return err != nil
			}, func(text string) {
				j := jid.MustParse(text)
				_, err := formData.Set(field.Var, j)
				if err != nil {
					ui.debug.Printf("error setting jid form field %s: %v", field.Var, err)
				}
			})
		case form.TypeListMulti, form.TypeList:
			// TODO: multi select list?
			opts, _ := formData.GetStrings(field.Var)
			box.AddDropDown(field.Label, opts, 0, func(option string, optionIndex int) {
				_, err := formData.Set(field.Var, option)
				if err != nil {
					ui.debug.Printf("error setting list or list-multi form field %s: %v", field.Var, err)
				}
			})
		case form.TypeTextMulti, form.TypeText:
			// TODO: multi line text, max lengths, etc.
			t, _ := formData.GetString(field.Var)
			box.AddInputField(field.Label, t, 20, nil, func(text string) {
				_, err := formData.Set(field.Var, text)
				if err != nil {
					ui.debug.Printf("error setting text or text-multi form field %s: %v", field.Var, err)
				}
			})
		case form.TypeTextPrivate:
			// TODO: multi line text, max lengths, etc.
			t, _ := formData.GetString(field.Var)
			box.AddPasswordField(field.Label, t, 20, '*', func(text string) {
				_, err := formData.Set(field.Var, text)
				if err != nil {
					ui.debug.Printf("error setting password form field %s: %v", field.Var, err)
				}
			})
		}
	})
	for _, button := range buttons {
		ui.cmdPane.Form().AddButton(button, func() {
			onDone(button)
		})
	}
}

// ShowNote shows a text note from an ad-hoc command.
func (ui *UI) ShowNote(note commands.Note, buttons []string, onDone func(string)) {
	defer func() {
		ui.buffers.SwitchToPage(cmdPageName)
		ui.app.SetFocus(ui.buffers)
		ui.Redraw()
	}()
	var symbol string
	switch note.Type {
	case commands.NoteInfo:
		symbol = "ℹ️\n"
	case commands.NoteWarn:
		symbol = "⚠️\n"
	case commands.NoteError:
		symbol = "❌\n"
	default:
		symbol = "⁉️\n"
	}
	ui.cmdPane.SetText(symbol, note.Value)
	ui.cmdPane.Form().Clear(true)
	for _, button := range buttons {
		ui.cmdPane.Form().AddButton(button, func() {
			onDone(button)
		})
	}
	ui.cmdPane.Form().SetButtonsAlign(tview.AlignCenter)
}

// SetCommands populates the list of ad-hoc commands in the list commands
// window. It should generally be called after the commands have been loaded and
// after the "ShowListCMD" function has been called (since that sets the text to
// a loading indicator).
func (ui *UI) SetCommands(j jid.JID, c []commands.Command) {
	p := ui.Printer()
	cancelButton := p.Sprintf("Cancel")
	execButton := p.Sprintf("Exec")
	defer func() {
		ui.buffers.SwitchToPage(cmdPageName)
		ui.app.SetFocus(ui.buffers)
		ui.Redraw()
	}()

	if len(c) == 0 {
		ui.cmdPane.Form().SetButtonsAlign(tview.AlignCenter)
		ui.cmdPane.SetText("Commands", fmt.Sprintf("No commands found for %v!", j))
		return
	}

	ui.cmdPane.Form().SetButtonsAlign(tview.AlignLeft)
	var cmds []string
	for _, name := range c {
		cmds = append(cmds, name.Name)
	}
	ui.cmdPane.SetText("Commands", j.String())
	var idx int
	ui.cmdPane.Form().
		Clear(true).
		AddDropDown(commandsLabel, cmds, 0, func(option string, optionIndex int) {
			idx = optionIndex
		})
	ui.cmdPane.Form().AddButton(cancelButton, func() {
		ui.SelectRoster()
	})
	ui.cmdPane.Form().AddButton(execButton, func() {
		ui.SelectRoster()
		ui.handler(event.ExecCommand(c[idx]))
	})
	ui.app.SetFocus(ui.buffers)
}

// ShowHelpPrompt shows a list of keyboard shortcuts..
func (ui *UI) ShowHelpPrompt() {
	onEsc := func() {
		ui.pages.HidePage(helpPageName)
		ui.pages.RemovePage(helpPageName)
	}
	// U+20E3 COMBINING ENCLOSING KEYCAP
	mod := NewModal().
		SetText(`[::b]Global[::-]

q, Esc: quit or close
F1, K: help or quick help

[::b]Navigation[::-]

Tab, Shift+Tab: focus to next/prev
gg, Home: scroll to top
G, End: scroll to bottom
h, ←: move left
j, ↓: move down
k, ↑: move up
l, →: move right
PageUp, PageDown: move up/down one page
<n>k: move <n> lines up
<n>j: move <n> lines down
/: search forward
?: search backward
n: next search result
N: previous search result
gt: next sidebar tab
gT: previous sidebar tab

[::b]Roster[::-]

c: start chat
i, Enter: open chat
I: more info
o, O: open next/prev unread
dd: remove contact
!: execute command`).
		SetDoneFunc(func(int, string) {
			onEsc()
		})
	mod.SetInputCapture(modalClose(onEsc))

	ui.pages.AddPage(helpPageName, mod, true, false)
	ui.pages.ShowPage(helpPageName)
	ui.pages.SendToFront(helpPageName)
	ui.app.SetFocus(ui.pages)
}

// ShowManualPage suspends the application and invokes man(1) to view the
// manual page.
func (ui *UI) ShowManualPage() {
	ui.app.Suspend(func() {
		cmd := exec.Command("man", "1", "communiqué")
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			ui.debug.Print(err)
		}
	})
}

// GetRosterJID gets the currently selected roster or bookmark JID.
func (ui *UI) GetRosterJID() jid.JID {
	selected, ok := ui.sidebar.GetSelected()
	if !ok {
		return jid.JID{}
	}
	switch s := selected.(type) {
	case RosterItem:
		return s.JID
	case BookmarkItem:
		return s.JID
	case Conversation:
		return s.JID
	}
	return jid.JID{}
}

func formatPresence(p []presence) string {
	var buf strings.Builder
	tabWriter := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', 0)
	for _, pres := range p {
		icon := ""
		switch pres.Status {
		case statusOnline:
			icon = "●"
		case statusBusy:
			icon = "◐"
		case statusAway:
			icon = "◓"
		case statusOffline:
			icon = "◯"
		}
		resPart := pres.From.Resourcepart()
		if resPart != "" {
			/* #nosec */
			fmt.Fprintf(tabWriter, "%s\t%s\t\n", icon, resPart)
		}
	}
	/* #nosec */
	tabWriter.Flush()
	return buf.String()
}

// PickResource shows a modal with the currently selected roster items resources
// and lets the user pick one.
// It then calls f with the full JID and whether or not picking a resource was
// successful.
func (ui *UI) PickResource(f func(jid.JID, bool)) {
	const pageName = "resource_picker"
	item, ok := ui.sidebar.roster.GetSelected()
	if !ok {
		ui.pages.HidePage(pageName)
		f(jid.JID{}, false)
		return
	}
	if len(item.presences) == 0 {
		ui.pages.HidePage(pageName)
		f(item.JID, true)
		return
	}
	opts := make([]string, 0, len(item.presences))
	bare := item.JID.String()
	var foundBare bool
	for _, p := range item.presences {
		addr := p.From.String()
		if addr == bare {
			foundBare = true
		}
		opts = append(opts, addr)
	}
	if !foundBare {
		opts = append(opts, bare)
	}
	if len(opts) == 1 {
		ui.pages.HidePage(pageName)
		f(item.JID, true)
		return
	}
	var idx int
	const selectButton = "Select"
	mod := NewModal().SetText("Pick Address").
		AddButtons([]string{selectButton})
	mod.Form().AddDropDown("Address", opts, 0, func(_ string, optionIndex int) {
		idx = optionIndex
	})
	mod.SetDoneFunc(func(_ int, label string) {
		ui.pages.HidePage(pageName)
		if label != selectButton {
			return
		}
		ui.pages.RemovePage(pageName)
		if idx >= len(item.presences) {
			f(item.JID, true)
			return
		}
		f(item.presences[idx].From, true)
	})

	ui.pages.AddPage(pageName, mod, false, true)
	ui.pages.ShowPage(pageName)
	ui.pages.SendToFront(pageName)
	ui.app.SetFocus(ui.pages)
}

var infoTmpl = template.Must(template.New("info").Funcs(template.FuncMap{
	"formatPresence": formatPresence,
}).Parse(`
🛈

{{ .Name }}
{{ if ne .JID.String .Name }}{{ .JID }}{{ end }}

{{ if .Room }}Bookmarked: {{ if .Bookmarked}}🔖{{ else }}✘{{ end }}{{ end }}
{{ if not .Room }}Subscription:
{{- if eq .Subscription "both" -}}
⇆
{{- else if eq .Subscription "to" -}}
→
{{- else if eq .Subscription "from" -}}
←
{{- else -}}
✘
{{- end -}}
{{- end }}
{{ if .Group }}Groups: {{ print "%v" .Group }}{{ end }}
{{ if .Presences }}
Resources:

{{ formatPresence .Presences }}
{{ end }}
`))

// ShowRosterInfo displays more info about the currently selected roster item.
func (ui *UI) ShowRosterInfo() {
	onEsc := func() {
		ui.pages.HidePage(infoPageName)
		ui.pages.RemovePage(infoPageName)
	}
	mod := NewModal().
		SetText(`Roster info:`).
		SetDoneFunc(func(int, string) {
			onEsc()
		})
	mod.SetInputCapture(modalClose(onEsc))

	v, ok := ui.sidebar.GetSelected()
	if !ok {
		ui.debug.Printf("no sidebar open, not showing info pane…")
		return
	}

	infoData := struct {
		Room         bool
		Bookmarked   bool
		Subscription string
		Name         string
		JID          jid.JID
		Group        []string
		Presences    []presence
	}{}
	// If the selected item is a conversation that also exists in the bookmarks or
	// roster bar, use the data from the bookmarks or roster instead.
	if c, ok := v.(Conversation); ok {
		if c.Room {
			bookmark, ok := ui.sidebar.bookmarks.GetItem(c.JID.Bare().String())
			if ok {
				v = bookmark
			}
		} else {
			item, ok := ui.sidebar.roster.GetItem(c.JID.Bare().String())
			if ok {
				v = item
			}
		}
	}
	switch item := v.(type) {
	case Conversation:
		infoData.Room = item.Room
		if item.Room {
			_, infoData.Bookmarked = ui.sidebar.bookmarks.GetItem(item.JID.Bare().String())
		}
		infoData.Name = item.Name
		infoData.JID = item.JID
	case BookmarkItem:
		infoData.Room = true
		infoData.Bookmarked = true
		infoData.Name = item.Name
		infoData.JID = item.JID
	case RosterItem:
		infoData.Name = item.Name
		if item.Name == "" {
			infoData.Name = item.JID.Localpart()
		}
		infoData.JID = item.JID
		infoData.Presences = item.presences
		infoData.Subscription = item.Subscription
	default:
		ui.debug.Printf("unrecognized sidebar item type %T, not showing info…", item)
		return
	}

	var buf strings.Builder
	err := infoTmpl.Execute(&buf, infoData)
	if err != nil {
		ui.debug.Printf("error executing info template: %v", err)
		return
	}

	mod.SetText(buf.String()).
		ClearButtons()
	// If we're not subscribed, add a subscribe button.
	if infoData.Subscription != "to" && infoData.Subscription != "both" {
		const subscribeBtn = "Subscribe"
		mod.AddButtons([]string{subscribeBtn}).
			SetDoneFunc(func(_ int, buttonLabel string) {
				switch buttonLabel {
				case subscribeBtn:
					ui.handler(event.Subscribe(infoData.JID.Bare()))
				}
				ui.pages.HidePage(infoPageName)
			})
	}
	ui.pages.AddPage(infoPageName, mod, true, false)
	ui.pages.ShowPage(infoPageName)
	ui.pages.SendToFront(infoPageName)
	ui.app.SetFocus(ui.pages)
}

// SelectRoster moves the input selection back to the roster and shows the logs
// view.
func (ui *UI) SelectRoster() {
	if ui.ChatsOpen() {
		item, ok := ui.sidebar.roster.GetSelected()
		if ok {
			ui.handler(event.CloseChat(item.Item))
		}
	}
	ui.buffers.SwitchToPage(logsPageName)
	ui.app.SetFocus(ui.sidebar)
}

// History returns the chat history view.
// To flush any remaining data to the buffer, the writer must be closed after
// use.
func (ui *UI) History() *tview.TextView {
	return ui.history.TextView
}

// GetRect returns the size of the UI on the screen (including borders and
// bounding boxes).
func (ui *UI) GetRect() (x, y, width, height int) {
	return ui.flex.GetRect()
}

// Redraw redraws the UI.
func (ui *UI) Redraw() {
	ui.redraw()
}

func (ui *UI) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlC:
		// The application intercepts Ctrl-C by default and terminates itself. We
		// don't want Ctrl-C to stop the application, so disable this behavior by
		// default. Manually sending a SIGINT will still work (see the signal
		// handling goroutine in main.go).
		return nil
	case tcell.KeyTAB, tcell.KeyBacktab:
		switch {
		case ui.buffers.HasFocus():
			name, _ := ui.buffers.GetFrontPage()
			if !ui.chatsOpen.Get() && name == logsPageName {
				ui.SelectRoster()
				return nil
			}
			return event
		case ui.sidebar.HasFocus():
			ui.buffers.SwitchToPage(logsPageName)
			ui.app.SetFocus(ui.buffers)
			return nil
		}
		return event
	case tcell.KeyESC:
		if ui.buffers.HasFocus() {
			ui.chatsOpen.Set(false)
			ui.SelectRoster()
			return nil
		}
		return event
	case tcell.KeyF1, tcell.KeyHelp:
		if name, _ := ui.pages.GetFrontPage(); name == uiPageName {
			ui.ShowManualPage()
			return nil
		}
	}

	return event
}
