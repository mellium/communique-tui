// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/text/message"

	"mellium.im/xmpp/jid"
)

func modalClose(onEsc func()) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Key() == tcell.KeyESC {
			onEsc()
		}
		return event
	}
}

func delRosterModal(p *message.Printer, onEsc func(), onDel func()) *tview.Modal {
	var (
		removeButton = p.Sprintf("Remove")
		cancelButton = p.Sprintf("Cancel")
	)
	mod := tview.NewModal().
		SetText(p.Sprintf("Remove this contact from your roster?")).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		AddButtons([]string{cancelButton, removeButton}).
		SetDoneFunc(func(_ int, buttonLabel string) {
			if buttonLabel == removeButton {
				onDel()
			}
			onEsc()
		})
	mod.SetInputCapture(modalClose(onEsc))
	return mod
}

func delBookmarkModal(p *message.Printer, onEsc func(), onDel func()) *tview.Modal {
	var (
		removeButton = p.Sprintf("Remove")
		cancelButton = p.Sprintf("Cancel")
	)
	mod := tview.NewModal().
		SetText(p.Sprintf("Remove this channel?")).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		AddButtons([]string{cancelButton, removeButton}).
		SetDoneFunc(func(_ int, buttonLabel string) {
			if buttonLabel == removeButton {
				onDel()
			}
			onEsc()
		})
	mod.SetInputCapture(modalClose(onEsc))
	return mod
}

// getJID creates a modal that asks for a JID. Eg. to add a bookmark or start a
// new conversation.
func getJID(p *message.Printer, title, addButton string, bare bool, f func(jid.JID, string), autocomplete []jid.JID) *Modal {
	var (
		cancelButton = p.Sprintf("Cancel")
	)
	mod := NewModal().
		SetText(title)
	var inputJID jid.JID
	jidInput := jidInput(p, &inputJID, bare, autocomplete, nil)
	modForm := mod.Form()
	modForm.AddFormItem(jidInput)
	mod.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		AddButtons([]string{cancelButton, addButton}).
		SetDoneFunc(func(_ int, buttonLabel string) {
			f(inputJID.Bare(), buttonLabel)
		})
	return mod
}

// jidInput returns a field that asks for a JID and validates it.
// As the user types the label will change to indicate if the JID is valid or
// invalid.
// If the JID is valid, it is unmarshaled into the intputJID pointer.
func jidInput(p *message.Printer, inputJID *jid.JID, bare bool, autocomplete []jid.JID, onChange func(string)) *tview.InputField {
	jidInput := tview.NewInputField()
	jidInput.SetPlaceholder("me@example.net")
	jidInput.SetChangedFunc(func(text string) {
		if text == "" {
			jidInput.SetLabel(p.Sprintf("Address"))
			return
		}
		j, err := jid.Parse(text)
		if err == nil && (!bare || j.Equal(j.Bare())) {
			jidInput.SetLabel("✅")
			*inputJID = j
		} else {
			jidInput.SetLabel("❌")
		}
		if onChange != nil {
			onChange(text)
		}
	})
	jidInput.SetAutocompleteFunc(func(s string) []string {
		idx := strings.IndexByte(s, '@')
		if idx < 0 {
			// If we're still typing the localpart of the JID, filter on all JIDs that
			// start out with the same local part.
			if s == "" {
				return nil
			}
			entriesSet := make(map[string]struct{})
			for _, item := range autocomplete {
				local := item.Localpart()
				if strings.HasPrefix(local, s) {
					entriesSet[item.String()] = struct{}{}
				}
			}
			entries := make([]string, 0, len(entriesSet))
			for entry := range entriesSet {
				entries = append(entries, entry)
			}
			return entries
		}
		// If we're now typing the domainpart of the JID, ignore the local part and
		// auto-complete the domainpart using the user entered localpart and the
		// domainparts we know about from existing JIDs.
		search := s[idx+1:]
		entriesSet := make(map[string]struct{})
		for _, item := range autocomplete {
			domainpart := item.Domainpart()
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
	})
	return jidInput
}
