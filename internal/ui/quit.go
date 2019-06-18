// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/rivo/tview"
)

func quitModal(done func(buttonIndex int, buttonLabel string)) *tview.Modal {
	return tview.NewModal().
		SetText("Are you sure you want to quit?").
		AddButtons([]string{"Quit", "Cancel"}).
		SetDoneFunc(done).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
}
