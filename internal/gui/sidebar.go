package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/widget"
)

func makeSideBar(conversationsList binding.StringList, conversationsMap map[string]*conversation, setChatBox func(*conversation)) fyne.CanvasObject {

	list := widget.NewListWithData(
		conversationsList,
		func() fyne.CanvasObject {
			address := widget.NewLabel("alice@example.com")
			address.TextStyle = fyne.TextStyle{Bold: true}
			address.Refresh()

			latestMessage := widget.NewLabel("This is a long test message that should be truncated for the sake of good display")
			latestMessage.Truncation = fyne.TextTruncateEllipsis
			latestMessage.Refresh()

			lastInteraction := widget.NewLabel("Oct 24")

			return container.NewBorder(nil, nil, nil, lastInteraction, container.NewVBox(address, latestMessage))
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			email := item.(binding.String)
			emailVal, _ := email.Get()
			borderContainer := obj.(*fyne.Container)
			vbox := borderContainer.Objects[0].(*fyne.Container)

			latestMessage := vbox.Objects[1].(*widget.Label)
			address := vbox.Objects[0].(*widget.Label)

			address.Bind(email)
			latestMessage.Bind(conversationsMap[emailVal].latestMessage)
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		email, _ := conversationsList.GetValue(id)
		conversation := conversationsMap[email]
		setChatBox(conversation)
	}

	list.OnUnselected = func(id widget.ListItemID) {
		email, _ := conversationsList.GetValue(id)
		conversation := conversationsMap[email]
		conversation.messageList.RemoveListener(conversation.dataListener)
	}

	accountCard := widget.NewCard("kenshin@slickerius.com", "Online", nil)

	button := widget.NewButton("Start New Chat", func() {
		w := fyne.CurrentApp().NewWindow("Start New Chat")

		emailEntry := widget.NewEntry()
		emailEntry.SetPlaceHolder("alice@example.com")
		emailEntry.Validator = validation.NewRegexp(`\w{1,}@\w{1,}\.\w{1,4}`, "not a valid JID")

		form := &widget.Form{
			Items: []*widget.FormItem{
				{Text: "JID", Widget: emailEntry, HintText: "Enter JID you want to chat with"},
			},
			OnCancel: func() {
				w.Close()
			},
			OnSubmit: func() {
				email := emailEntry.Text
				conversation := newConversation(email, "")
				conversationsList.Append(email)
				conversationsMap[email] = conversation
				w.Close()
			},
		}

		w.SetContent(form)
		w.Resize(fyne.NewSize(300, 100))
		w.SetFixedSize(true)
		w.CenterOnScreen()
		w.Show()
	})

	return container.NewBorder(accountCard, container.NewPadded(button), nil, nil, list)
}
