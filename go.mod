module mellium.im/communiqué

go 1.12

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/gdamore/encoding v0.0.0-20151215212835-b23993cbb635 // indirect
	github.com/gdamore/tcell v1.1.0
	github.com/gopherjs/gopherjs v0.0.0-20181103185306-d547d1d9531e // indirect
	github.com/jtolds/gls v4.2.1+incompatible // indirect
	github.com/lucasb-eyer/go-colorful v0.0.0-20181028223441-12d3b2882a08 // indirect
	github.com/mattn/go-runewidth v0.0.3 // indirect
	github.com/rivo/tview v0.0.0-20181203172859-599127851351
	github.com/smartystreets/assertions v0.0.0-20180927180507-b2de0cb4f26d // indirect
	github.com/smartystreets/goconvey v0.0.0-20181108003508-044398e4856c // indirect
	golang.org/x/crypto v0.0.0-20181203042331-505ab145d0a9 // indirect
	golang.org/x/net v0.0.0-20181207154023-610586996380 // indirect
	golang.org/x/text v0.3.0 // indirect
	gopkg.in/DATA-DOG/go-sqlmock.v1 v1.3.0 // indirect
	mellium.im/sasl v0.2.1
	mellium.im/xmlstream v0.13.1
	mellium.im/xmpp v0.9.0
)

replace (
	github.com/mattn/go-runewidth v0.0.3 => github.com/mattn/go-runewidth v0.0.0-20181118013326-c88d7e5f2e24
	github.com/rivo/tview v0.0.0-20180821142722-77bcb6c6b900 => /home/sam/go/src/github.com/rivo/tview
	mellium.im/xmpp v0.9.0 => /home/sam/go/src/mellium.im/xmpp
)
