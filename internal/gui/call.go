package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"mellium.im/communique/internal/client/event"
	"mellium.im/communique/internal/client/jingle"
	"mellium.im/xmpp/jid"
)

func (g *GUI) ShowOutgoingCall(account jid.JID) {
	if g.outgoingCallWindow == nil {
		g.outgoingCallWindow = g.app.NewWindow("Calling")
		g.outgoingCallWindow.SetCloseIntercept(func() {
			g.handler(event.CancelCall(""))
			g.outgoingCallWindow.Close()
		})

		callCard := widget.NewCard(account.Bare().String(), "Calling....", nil)
		cancelButton := widget.NewButton("Cancel", func() {
			g.handler(event.CancelCall(""))
			g.outgoingCallWindow.Close()
		})

		g.outgoingCallWindow.SetContent(container.NewBorder(nil, cancelButton, nil, nil, callCard))
		g.outgoingCallWindow.CenterOnScreen()
		g.outgoingCallWindow.SetFixedSize(true)
		g.outgoingCallWindow.Show()
	}
}

func (g *GUI) ShowIncomingCall(jingleRequest *jingle.Jingle) {
	if g.incomingCallWindow == nil {
		callerJid := jid.MustParse(jingleRequest.Initiator)
		g.incomingCallWindow = g.app.NewWindow("Incoming Call")
		g.incomingCallWindow.SetCloseIntercept(func() {
			g.handler(event.CancelCall(""))
			g.incomingCallWindow.Close()
		})

		callCard := widget.NewCard(callerJid.Bare().String(), "Incoming Call", nil)
		declineButton := widget.NewButton("Decline", func() {
			g.handler(event.CancelCall(""))
			g.incomingCallWindow.Close()
		})
		acceptButton := widget.NewButton("Accept", func() {
			g.handler(event.AcceptIncomingCall(jingleRequest))
			g.incomingCallWindow.Close()
		})

		g.incomingCallWindow.SetContent(container.NewBorder(nil, container.NewHBox(declineButton, acceptButton), nil, nil, callCard))
		g.incomingCallWindow.CenterOnScreen()
		g.incomingCallWindow.SetFixedSize(true)
		g.incomingCallWindow.Show()
	}
}

func (g *GUI) ShowCallSession() {
	if g.callSessionWindow == nil {
		g.TerminateCallSession()
		g.callSessionWindow = g.app.NewWindow("Call Session")
		g.callSessionWindow.SetCloseIntercept(func() {
			g.handler(event.TerminateCall(""))
			g.callSessionWindow.Close()
		})

		endButton := widget.NewButton("End Call", func() {
			g.handler(event.TerminateCall(""))
			g.callSessionWindow.Close()
		})

		g.callSessionWindow.SetContent(container.New(layout.NewMaxLayout(), endButton))
		g.callSessionWindow.CenterOnScreen()
		g.callSessionWindow.Resize(fyne.NewSize(300, 100))
		g.callSessionWindow.SetFixedSize(true)
		g.callSessionWindow.Show()
	}
}

func (g *GUI) TerminateCallSession() {
	if g.outgoingCallWindow != nil {
		g.outgoingCallWindow.Close()
		g.outgoingCallWindow = nil
	}
	if g.incomingCallWindow != nil {
		g.incomingCallWindow.Close()
		g.incomingCallWindow = nil
	}
	if g.callSessionWindow != nil {
		g.callSessionWindow.Close()
		g.callSessionWindow = nil
	}
}
