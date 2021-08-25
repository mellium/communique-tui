// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Label is a tview primitive that can be added to forms and draws a single line
// of text.
// It does not draw any actual form item.
type label struct {
	*tview.Box
	label    string
	finished func(key tcell.Key)
}

func newLabel(l string) *label {
	return &label{
		Box:   tview.NewBox(),
		label: l,
	}
}

// GetLabel always returns the empty string (the label itself is drawn by the
// widget, not as a form label which would change indentation of other, shorter,
// labels around it).
func (l label) GetLabel() string {
	return ""
}

// SetFormAttributes is a noop.
func (l *label) SetFormAttributes(labelWidth int, labelColor, bgColor, fieldTextColor, fieldBgColor tcell.Color) tview.FormItem {
	return l
}

// GetFieldWidth always returns 0 (dynamic width).
func (l *label) GetFieldWidth() int {
	return 0
}

// SetFinishedFunc is a noop.
func (l *label) SetFinishedFunc(handler func(key tcell.Key)) tview.FormItem {
	l.finished = handler
	return l
}

// Draw draws the text.
func (l *label) Draw(screen tcell.Screen) {
	l.Box.DrawForSubclass(screen, l)
	//totalWidth, totalHeight := screen.Size()
	x, y, _, _ := l.GetInnerRect()
	tview.PrintSimple(screen, l.label, x, y)
}

func (l *label) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return l.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		// Process key event.
		switch key := event.Key(); key {
		//case tcell.KeyRune, tcell.KeyEnter: // Check.
		case tcell.KeyTab, tcell.KeyBacktab, tcell.KeyEscape: // We're done.
			if l.finished != nil {
				l.finished(key)
			}
		}
	})
}
