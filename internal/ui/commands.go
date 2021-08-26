// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/rivo/tview"
)

type commandsPane struct {
	*tview.Frame
	form *tview.Form
}

func cmdPane() *commandsPane {
	c := &commandsPane{
		form: tview.NewForm(),
	}
	c.Frame = tview.NewFrame(c.form)
	c.Frame.SetBorder(true)
	return c
}

func (c *commandsPane) SetText(title, text string) {
	c.Frame.SetTitle(title)
	c.Frame.Clear()
	c.Frame.AddText(text, true, tview.AlignCenter, tview.Styles.PrimaryTextColor)
}

func (c *commandsPane) Form() *tview.Form {
	return c.form
}
