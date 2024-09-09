// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/rivo/tview"
)

func statusModal(done func(buttonIndex int, buttonLabel string)) *tview.Modal {
	mod := tview.NewModal().
		SetText("Set Status").
		AddButtons([]string{"Online [green]●", "Away [orange]◓", "Busy [red]◑", "Offline ○"}).
		SetDoneFunc(done).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	mod.SetInputCapture(modalClose(func() {
		done(-1, "")
	}))
	return mod
}
