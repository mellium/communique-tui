// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	cancelButton = "Cancel"
	execButton   = "Execute"
)

func modalClose(onEsc func()) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Key() == tcell.KeyESC {
			onEsc()
		}
		return event
	}
}

func delRosterModal(onEsc func(), onDel func()) *tview.Modal {
	const (
		removeButton = "Remove"
	)
	mod := tview.NewModal().
		SetText(`Remove this contact from your roster?`).
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

func delBookmarkModal(onEsc func(), onDel func()) *tview.Modal {
	const (
		removeButton = "Remove"
	)
	mod := tview.NewModal().
		SetText(`Remove this channel?`).
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
