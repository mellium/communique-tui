package gui

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/widget"
	"mellium.im/communique/internal/client/event"
	"mellium.im/xmpp/jid"
)

type GUI struct {
	app                fyne.App
	conversationsList  binding.StringList
	conversationsMap   map[string]*conversation
	isRunning          bool
	mainWindow         fyne.Window
	outgoingCallWindow fyne.Window
	incomingCallWindow fyne.Window
	callSessionWindow  fyne.Window
	debug              *log.Logger
	accountCard        *widget.Card
	handler            func(interface{})
}

type LoginData struct {
	JID  string
	Pass string
}

func (gui *GUI) Run(jidChan chan *LoginData) {
	// Account input GUI Setup
	loginWindow := gui.app.NewWindow("Enter JID")

	loginWindow.SetCloseIntercept(func() {
		gui.debug.Println("Close window button triggered")
		jidChan <- &LoginData{
			JID:  "",
			Pass: "",
		}
		loginWindow.Close()
	})

	email := widget.NewEntry()
	email.SetPlaceHolder("alice@example.com")
	email.Validator = validation.NewRegexp(`\w{1,}@\w{1,}\.\w{1,4}`, "not a valid email")

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Password")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "JID", Widget: email},
			{Text: "Password", Widget: password},
		},
		OnCancel: func() {
			gui.debug.Println("Cancelled JID Entry")
			jidChan <- &LoginData{
				JID:  "",
				Pass: "",
			}
			loginWindow.Close()
		},
		OnSubmit: func() {
			gui.debug.Println("Submitted JID entry")
			jidChan <- &LoginData{
				JID:  email.Text,
				Pass: password.Text,
			}
			gui.accountCard.SetTitle(email.Text)
			gui.accountCard.SetSubTitle("Offline")
			gui.mainWindow.Show()
			loginWindow.Close()
		},
	}

	loginWindow.SetContent(form)
	loginWindow.Resize(fyne.NewSize(300, 100))
	loginWindow.SetFixedSize(true)
	loginWindow.CenterOnScreen()
	loginWindow.Show()

	gui.mainWindow.SetOnClosed(func() {
		gui.handler(event.StatusOffline(jid.JID{}))
	})

	gui.isRunning = true
	gui.app.Run()
	gui.isRunning = false
}

func (gui *GUI) Quit() {
	gui.app.Quit()
}

func (gui *GUI) Handler(h func(interface{})) {
	if h == nil {
		gui.handler = func(i interface{}) {}
		return
	}
	gui.handler = h
}

func (gui *GUI) Away(j jid.JID) {
	gui.accountCard.SetTitle(j.Bare().String())
	gui.accountCard.SetSubTitle("Away")
}

func (gui *GUI) Busy(j jid.JID) {
	gui.accountCard.SetTitle(j.Bare().String())
	gui.accountCard.SetSubTitle("Busy")
}

func (gui *GUI) Online(j jid.JID) {
	gui.accountCard.SetTitle(j.Bare().String())
	gui.accountCard.SetSubTitle("Online")
}

func (gui *GUI) Offline(j jid.JID) {
	gui.accountCard.SetTitle(j.Bare().String())
	gui.accountCard.SetSubTitle("Offline")
}

func (gui *GUI) WriteMessage(msg event.ChatMessage) {
	if msg.Body == "" {
		return
	}

	chatAddr := msg.From
	if msg.Sent {
		chatAddr = msg.To
	}
	bareJid := chatAddr.Bare().String()

	conversation, ok := gui.conversationsMap[bareJid]
	if !ok {
		gui.conversationsList.Append(bareJid)
		gui.conversationsMap[bareJid] = newConversation(bareJid, chatAddr.Resourcepart())
		conversation = gui.conversationsMap[bareJid]
	}

	// Replace resourcepart in case we are the one who initiate the chat
	conversation.resource = chatAddr.Resourcepart()

	conversation.messageList.Append(&message{
		content: msg.Body,
		sent:    msg.Sent,
	})
	conversation.latestMessage.Set(msg.Body)
}

func New(debug *log.Logger) *GUI {
	app := app.New()

	// Main Window GUI Setup
	mainWindow := app.NewWindow("XMPP Client")
	mainWindow.SetMaster()

	conversationsList := binding.NewStringList()
	conversationsMap := map[string]*conversation{}

	chatbox := container.NewStack(container.NewCenter(widget.NewLabel("Your conversations will appear here")))

	gui := &GUI{
		app:               app,
		conversationsList: conversationsList,
		conversationsMap:  conversationsMap,
		isRunning:         false,
		mainWindow:        mainWindow,
		debug:             debug,
		handler:           func(i interface{}) {},
	}

	setChatBox := func(c *conversation) {
		chatbox.Objects = []fyne.CanvasObject{makeChatBox(c, gui)}
		chatbox.Refresh()
	}
	sidebar := makeSideBar(conversationsList, conversationsMap, setChatBox)
	accountCard := sidebar.(*fyne.Container).Objects[1].(*widget.Card)

	split := container.NewHSplit(sidebar, chatbox)
	split.Offset = 0.2

	mainWindow.SetContent(split)
	mainWindow.Resize(fyne.NewSize(1280, 720))

	gui.accountCard = accountCard

	return gui
}
