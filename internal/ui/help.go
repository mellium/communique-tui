// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func helpModal(onEsc func()) *tview.Modal {
	// U+20E3 COMBINING ENCLOSING KEYCAP
	mod := tview.NewModal().
		SetText(`Global :

q⃣: quit or close
Esc ⎋: close
?⃣: help


Navigation:

Tab ⇥⃣: focus to next
Shift+Tab ⇧⃣ + ⇥⃣: focus to previous
g⃣ g⃣, Home ⇱⃣: scroll to top
G⃣, End ⇲⃣: scroll to bottom
h⃣, ←⃣: move left
j⃣, ↓⃣: move down
k⃣, ↑⃣: move up
l⃣, →⃣: move right
Page Up ⇞⃣: move up one page
Page Down ⇟⃣: move down one page


Roster:

i⃣, ⏎⃣: open chat
1⃣ 0⃣ k⃣: move 10 entries up
1⃣ 0⃣ j⃣: move 10 entries down
`).
		SetDoneFunc(func(int, string) {
			onEsc()
		}).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	mod.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Key() == tcell.KeyESC {
			onEsc()
		}
		return nil
	})
	return mod
}
