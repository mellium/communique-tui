package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"mellium.im/communique/internal/client/event"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

func generateChatsLabel(c *conversation) []fyne.CanvasObject {
	chatsLabel := []fyne.CanvasObject{}
	messages, _ := c.messageList.Get()
	for _, messageItem := range messages {
		messageObj := messageItem.(*message)
		messageLabel := widget.NewLabel(messageObj.content)
		messageLabel.Wrapping = fyne.TextWrapWord
		if messageObj.sent {
			messageLabel.Alignment = fyne.TextAlignTrailing
		}
		messageLabel.Refresh()
		chatsLabel = append(chatsLabel, messageLabel)
	}
	return chatsLabel
}

func makeChatBox(c *conversation, g *GUI) fyne.CanvasObject {
	chatsLabel := generateChatsLabel(c)
	chatsBase := container.NewVBox(chatsLabel...)
	chats := container.NewVScroll(chatsBase)
	chats.ScrollToBottom()

	c.dataListener = binding.NewDataListener(func() {
		chatsBase.Objects = generateChatsLabel(c)
		chatsBase.Refresh()
		chats.ScrollToBottom()
	})

	c.messageList.AddListener(c.dataListener)

	toolbar := makeToolbar(c, g)
	input := makeInput(c, g)
	return container.NewBorder(toolbar, input, nil, nil, chats)
}

func makeToolbar(c *conversation, g *GUI) fyne.CanvasObject {
	addressCard := widget.NewCard(c.email, "", nil)
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.MediaPlayIcon(), func() {
		}),
		widget.NewToolbarAction(theme.MediaVideoIcon(), func() {
			fullJid := c.email
			if c.resource != "" {
				fullJid += "/" + c.resource
			}
			g.handler(event.NewOutgoingCall(jid.MustParse(fullJid)))
		}),
	)
	return container.NewBorder(nil, nil, nil, toolbar, addressCard)
}

func makeInput(c *conversation, g *GUI) fyne.CanvasObject {
	entry := widget.NewMultiLineEntry()
	entry.SetPlaceHolder("Enter your message here")
	entry.SetMinRowsVisible(2)
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.MailSendIcon(), func() {
			if entry.Text == "" {
				return
			}
			sendMessage(c, g, entry.Text)
			entry.SetText("")
		}),
	)
	return container.NewPadded(container.NewBorder(nil, nil, nil, toolbar, entry))
}

func sendMessage(c *conversation, g *GUI, messageContent string) {
	message := &message{
		content: messageContent,
		sent:    true,
	}
	c.messageList.Append(message)
	to := jid.MustParse(c.email)
	if c.resource != "" {
		to, _ = to.WithResource(c.resource)
	}
	g.handler(event.ChatMessage{
		Message: stanza.Message{
			To:   to,
			Type: stanza.ChatMessage,
		},
		Body: messageContent,
		Sent: true,
	})
	c.latestMessage.Set(messageContent)
}
