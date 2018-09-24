# Communiqué

[![Issue Tracker on Soquee][badge]][issues]

An experimental instant messaging client written in [Go] for services that
support the [XMPP protocol] (including the public Jabber network).

## Building

To build Communiqué you will need Go 1.11 or higher.
If an appropriate version of Go is already installed, try running `make` (or
`bmake`).
To bootstrap from an existing Go install, try the following:

    go get golang.org/dl/go1.11
    go1.11 download
    GO=go1.11 bmake

[badge]: https://img.shields.io/badge/style-mellium%2fcommuniqu%c3%a9--tui-green.svg?longCache=true&style=popout-square&label=soquee
[issues]: https://www.soquee.net/issues/mellium/communiqu%c3%a9-tui
[Go]: https://golang.org/
[XMPP protocol]: https://tools.ietf.org/html/rfc6121
