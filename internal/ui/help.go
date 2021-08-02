// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func modalClose(onEsc func()) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'q' || event.Key() == tcell.KeyESC {
			onEsc()
		}
		return nil
	}
}

func infoModal(onEsc func()) *tview.Modal {
	mod := tview.NewModal().
		SetText(`Roster info:`).
		SetDoneFunc(func(int, string) {
			onEsc()
		}).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	mod.SetInputCapture(modalClose(onEsc))
	return mod
}

func helpModal(onEsc func()) *tview.Modal {
	// U+20E3 COMBINING ENCLOSING KEYCAP
	mod := tview.NewModal().
		SetText(`Global :

q⃣: quit or close
⎋⃣: close
K⃣: help


Navigation:

⇥⃣, ⇤⃣ focus to next/prev
g⃣ g⃣, ⇱⃣ scroll to top
G⃣, ⇲⃣ scroll to bottom
h⃣, ←⃣ move left
j⃣, ↓⃣ move down
k⃣, ↑⃣ move up
l⃣, →⃣ move right
⇞⃣, ⇟⃣ move up/down one page
1⃣ 0⃣ k⃣ move 10 lines up
1⃣ 0⃣ j⃣ move 10 lines down
/⃣ search forward
?⃣ search backward
n⃣ next search result
N⃣ previous search result


Roster:

i⃣, ⏎⃣ open chat
I⃣ more info
o⃣, O⃣: open next/prev unread
`).
		SetDoneFunc(func(int, string) {
			onEsc()
		}).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	mod.SetInputCapture(modalClose(onEsc))
	return mod
}
