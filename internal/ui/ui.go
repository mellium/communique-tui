// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package ui ties together various widgets to create the main Communiqué UI.
package ui // import "mellium.im/communique/internal/ui"

import (
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mellium.im/communique/internal/client/event"
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
	quitPageName        = "quit"
	helpPageName        = "help"
	delRosterPageName   = "del_roster"
	addRosterPageName   = "add_roster"
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
	app            *tview.Application
	flex           *tview.Flex
	pages          *tview.Pages
	buffers        *tview.Pages
	history        *ConversationView
	statusBar      *tview.TextView
	roster         *Roster
	rosterWidth    int
	logWriter      *tview.TextView
	handler        func(interface{})
	redraw         func() *tview.Application
	addr           string
	passPrompt     chan string
	chatsOpen      *syncBool
	infoModal      *tview.Modal
	addRosterModal *Modal
	cmdPane        *commandsPane
	debug          *log.Logger
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
		ui.roster.ShowStatus(show)
	}
}

// InputCapture returns an option that overrides the default input handler for
// the application.
func InputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) Option {
	return func(ui *UI) {
		ui.app.SetInputCapture(capture)
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
			ui.roster.Width = 25
			ui.rosterWidth = 25
			return
		}
		if width < 2 {
			ui.roster.Width = 2
			ui.rosterWidth = 2
			return
		}
		if width > 50 {
			ui.roster.Width = 50
			ui.rosterWidth = 50
			return
		}
		ui.roster.Width = width
		ui.rosterWidth = width
	}
}

// New constructs a new UI.
func New(opts ...Option) *UI {
	app := tview.NewApplication()
	statusBar := tview.NewTextView()
	statusBar.
		SetTextColor(tview.Styles.PrimaryTextColor).
		SetBackgroundColor(tview.Styles.MoreContrastBackgroundColor).
		SetBorder(false).
		SetBorderPadding(0, 0, 2, 0)
	buffers := tview.NewPages()
	pages := tview.NewPages()

	rosterBox := newRoster(func() {
		pages.ShowPage(setStatusPageName)
		pages.SendToFront(setStatusPageName)
		app.SetFocus(pages)
	}, func() {
		pages.ShowPage(delRosterPageName)
		pages.SendToFront(delRosterPageName)
		app.SetFocus(pages)
	})
	rosterBox.OnChanged(func(idx int, main string, secondary string, shortcut rune) {
		if idx == 0 {
			statusBar.SetText("Status: " + main)
			return
		}
		main = strings.TrimPrefix(main, highlightTag)
		statusBar.SetText(fmt.Sprintf("Chat: %q (%s)", main, secondary))
	})

	ui := &UI{
		app:         app,
		roster:      rosterBox,
		rosterWidth: 25,
		statusBar:   statusBar,
		handler:     func(interface{}) {},
		redraw:      app.Draw,
		buffers:     buffers,
		pages:       pages,
		passPrompt:  make(chan string),
		chatsOpen:   &syncBool{},
		debug:       log.New(io.Discard, "", 0),
	}
	ui.infoModal = infoModal(func() {
		ui.pages.HidePage(infoPageName)
	})
	ui.cmdPane = cmdPane()
	ui.addRosterModal = addRosterModal(func(s string) []string {
		idx := strings.IndexByte(s, '@')
		if idx < 0 {
			return nil
		}
		search := s[idx+1:]
		entriesSet := make(map[string]struct{})
		for _, item := range ui.roster.items {
			domainpart := item.JID.Domainpart()
			entry := strings.TrimPrefix(domainpart, search)
			if entry == domainpart {
				continue
			}
			entriesSet[entry] = struct{}{}
		}
		var entries []string
		for entry := range entriesSet {
			entries = append(entries, s+entry)
		}
		return entries
	}, func() {
		ui.pages.HidePage(addRosterPageName)
	}, func(j jid.JID) {
		// add to roster
		go func() {
			ui.UpdateRoster(RosterItem{
				Item: roster.Item{
					JID: j,
				},
			})
		}()
	})
	for _, o := range opts {
		o(ui)
	}

	chats := NewConversationView(ui)
	ui.history = chats
	buffers.AddPage(chatPageName, chats, true, false)

	logs := newLogs(app, func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTAB, tcell.KeyBacktab:
			name, _ := ui.buffers.GetFrontPage()
			if name == logsPageName {
				ui.chatsOpen.Set(false)
				ui.SelectRoster()
				return nil
			}
			return event
		case tcell.KeyESC:
			ui.chatsOpen.Set(false)
			ui.SelectRoster()
			return nil
		}
		return event
	})
	buffers.AddPage(logsPageName, logs, true, true)
	ui.logWriter = logs

	innerCapture := rosterBox.GetInputCapture()
	rosterBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		key := event.Key()
		eventRune := event.Rune()
		switch {
		case key == tcell.KeyTAB || key == tcell.KeyBacktab:
			buffers.SwitchToPage(logsPageName)
			app.SetFocus(buffers)
			return nil
		case eventRune == '!':
			ui.ShowLoadCmd()
			return nil
		case eventRune == 'c':
			ui.ShowAddRoster()
			return nil
		case eventRune == 'q':
			ui.ShowQuitPrompt()
			return nil
		case eventRune == 'K' || key == tcell.KeyF1 || key == tcell.KeyHelp:
			ui.ShowHelpPrompt()
			return nil
		case eventRune == 'I':
			ui.ShowRosterInfo()
			return nil
		}

		if innerCapture != nil {
			return innerCapture(event)
		}

		return event
	})

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
		AddItem(rosterBox, ui.rosterWidth, 1, true).
		AddItem(buffers, 0, 1, false)
	ui.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(ltrFlex, 0, 1, true).
		AddItem(statusBar, 1, 1, false)

	ui.pages.AddPage(setStatusPageName, setStatusPage, true, false)
	ui.pages.AddPage(uiPageName, ui.flex, true, true)
	ui.pages.AddPage(quitPageName, quitModal(func(buttonIndex int, _ string) {
		if buttonIndex == 0 {
			app.Stop()
		}
		ui.pages.HidePage(quitPageName)
	}), true, false)
	ui.pages.AddPage(helpPageName, helpModal(func() {
		ui.pages.HidePage(helpPageName)
	}), true, false)
	ui.pages.AddPage(addRosterPageName, ui.addRosterModal, true, false)
	buffers.AddPage(cmdPageName, ui.cmdPane, true, false)
	ui.pages.AddPage(delRosterPageName, delRosterModal(func() {
		ui.pages.HidePage(delRosterPageName)
	}, func() {
		cur := ui.roster.list.GetCurrentItem()
		if cur == 0 {
			// we can't delete the status selector.
			return
		}
		for _, item := range ui.roster.items {
			if item.idx == cur {
				ui.handler(event.DeleteRosterItem(item.Item))
				break
			}
		}
	}), true, false)

	ui.pages.AddPage(infoPageName, ui.infoModal, true, false)
	ui.pages.AddPage(getPasswordPageName, getPasswordPage, true, false)

	return ui
}

// RosterLen returns the length of the roster.
func (ui *UI) RosterLen() int {
	return ui.roster.Len()
}

// UpdateRoster adds an item to the roster.
func (ui *UI) UpdateRoster(item RosterItem) {
	ui.roster.Upsert(item, func() {
		ui.buffers.SwitchToPage(chatPageName)
		ui.chatsOpen.Set(true)
		item, ok := ui.roster.GetSelected()
		if ok {
			ui.handler(event.OpenChat(item.Item))
		}
		ui.app.SetFocus(ui.buffers)
	})
	ui.redraw()
	ui.handler(event.UpdateRoster{Item: item.Item})
}

// Write writes to the logging text view.
func (ui *UI) Write(p []byte) (n int, err error) {
	return ui.logWriter.Write(p)
}

// Roster returns the underlying roster pane widget.
func (ui *UI) Roster() *Roster {
	return ui.roster
}

// ChatsOpen returns true if the chat pane is open.
func (ui *UI) ChatsOpen() bool {
	return ui.chatsOpen.Get()
}

// Offline sets the state of the roster to show the user as offline.
func (ui *UI) Offline(j jid.JID, self bool) {
	if self {
		ui.roster.Offline()
		ui.redraw()
	}
	ui.roster.UpsertPresence(j, statusOffline)
}

// Online sets the state of the roster to show the user as online.
func (ui *UI) Online(j jid.JID, self bool) {
	if self {
		ui.roster.Online()
		ui.redraw()
	}
	ui.roster.UpsertPresence(j, statusOnline)
}

// Away sets the state of the roster to show the user as away.
func (ui *UI) Away(j jid.JID, self bool) {
	if self {
		ui.roster.Away()
		ui.redraw()
	}
	ui.roster.UpsertPresence(j, statusAway)
}

// Busy sets the state of the roster to show the user as busy.
func (ui *UI) Busy(j jid.JID, self bool) {
	if self {
		ui.roster.Busy()
		ui.redraw()
	}
	ui.roster.UpsertPresence(j, statusBusy)
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
	ui.pages.ShowPage(quitPageName)
	ui.pages.SendToFront(quitPageName)
	ui.app.SetFocus(ui.pages)
}

// ShowAddRoster asks the user for a new JID.
func (ui *UI) ShowAddRoster() {
	ui.pages.ShowPage(addRosterPageName)
	ui.pages.SendToFront(addRosterPageName)
	ui.app.SetFocus(ui.pages)
}

// ShowLoadCmd shows available ad-hoc commands for the selected JID.
func (ui *UI) ShowLoadCmd() {
	ui.cmdPane.Form().SetButtonsAlign(tview.AlignLeft)
	ui.cmdPane.SetText("Commands", "Loading commands…")
	ui.cmdPane.Form().Clear(true).
		AddButton(cancelButton, func() {
			ui.SelectRoster()
		})
	ui.buffers.SwitchToPage(cmdPageName)
	ui.app.SetFocus(ui.buffers)
	ui.handler(event.LoadingCommands{})
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
					box.AddFormItem(newLabel(line))
				}
			}
			for _, val := range field.Raw {
				for _, line := range strings.Split(val, "\n") {
					box.AddFormItem(newLabel(line))
				}
			}
			// TODO: will this just work? it's on the form already right?
		//case form.TypeHidden:
		//box.AddButton("Hidden: "+field.Label, nil)
		case form.TypeJIDMulti:
			jids, _ := formData.GetJIDs(field.Var)
			opts := make([]string, 0, len(jids))
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
	defer func() {
		ui.buffers.SwitchToPage(cmdPageName)
		ui.app.SetFocus(ui.buffers)
		ui.Redraw()
	}()

	if len(c) == 0 {
		ui.cmdPane.Form().SetButtonsAlign(tview.AlignCenter)
		ui.cmdPane.SetText("Commands", "No commands found!")
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
	ui.pages.ShowPage(helpPageName)
	ui.pages.SendToFront(helpPageName)
	ui.app.SetFocus(ui.pages)
}

// GetRosterJID gets the currently selected roster JID.
func (ui *UI) GetRosterJID() jid.JID {
	item, _ := ui.roster.GetSelected()
	return item.JID
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
		/* #nosec */
		fmt.Fprintf(tabWriter, "%s\t%s\t\n", icon, pres.From.Resourcepart())
	}
	/* #nosec */
	tabWriter.Flush()
	return buf.String()
}

// ShowRosterInfo displays more info about the currently selected roster item.
func (ui *UI) ShowRosterInfo() {
	item, ok := ui.roster.GetSelected()
	idx := ui.roster.list.GetCurrentItem()
	if !ok {
		main, secondary := ui.roster.list.GetItemText(idx)
		ui.infoModal.SetText(fmt.Sprintf(`%s
%s
`, main, secondary))
	} else {
		subscriptionIcon := "✘"
		switch item.Subscription {
		case "both":
			subscriptionIcon = "⇆"
		case "to":
			subscriptionIcon = "→"
		case "from":
			subscriptionIcon = "←"
		}
		name := item.Name
		if name == "" {
			name = item.JID.Localpart()
		}
		ui.infoModal.SetText(fmt.Sprintf(`🛈

%s
%s

Subscription: %s
Groups: %v
Resources:

%s
`, name, item.JID, subscriptionIcon, item.Group, formatPresence(item.presences))).
			ClearButtons()
		// If this isn't the status button or the "Me" item and we're not
		// subscribed, add a subscribe button.
		if idx > 1 && item.Subscription != "to" && item.Subscription != "both" {
			const subscribeBtn = "Subscribe"
			ui.infoModal.AddButtons([]string{subscribeBtn}).
				SetDoneFunc(func(_ int, buttonLabel string) {
					switch buttonLabel {
					case subscribeBtn:
						ui.handler(event.Subscribe(item.JID.Bare()))
					}
					ui.pages.HidePage(infoPageName)
				})
		}
	}
	ui.pages.ShowPage(infoPageName)
	ui.pages.SendToFront(infoPageName)
	ui.app.SetFocus(ui.pages)
}

// SelectRoster moves the input selection back to the roster and shows the logs
// view.
func (ui *UI) SelectRoster() {
	if ui.ChatsOpen() {
		item, ok := ui.roster.GetSelected()
		if ok {
			ui.handler(event.CloseChat(item.Item))
		}
	}
	ui.buffers.SwitchToPage(logsPageName)
	ui.app.SetFocus(ui.roster)
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
