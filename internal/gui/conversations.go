package gui

import "fyne.io/fyne/v2/data/binding"

type message struct {
	content string
	sent    bool
}

type conversation struct {
	email         string
	resource      string
	messageList   binding.UntypedList
	latestMessage binding.String
	dataListener  binding.DataListener
}

func newConversation(email string, resource string) *conversation {
	return &conversation{
		email:         email,
		resource:      resource,
		messageList:   binding.NewUntypedList(),
		latestMessage: binding.NewString(),
	}
}
