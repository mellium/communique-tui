# Communiqué

An experimental instant messaging client written in [Go] for services that
support the [XMPP protocol] (including the public Jabber network).

## Building

To build Communiqué you will need Go 1.11 or higher.
To bootstrap Go 1.11 from an existing Go install and build, try the following:

    go get golang.org/dl/go1.11
    go1.11 download
    GO=go1.11 bmake

You can also build to a temporary directory and run it during development:

    go1.11 run .

[Go]: https://golang.org/
[XMPP protocol]: https://tools.ietf.org/html/rfc6121
