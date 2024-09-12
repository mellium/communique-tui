# Communiqué

[![Issue Tracker][badge]](https://mellium.im/issue/)
[![Chat](https://img.shields.io/badge/XMPP-users@mellium.chat-orange.svg)](https://mellium.chat)
[![License](https://img.shields.io/badge/license-FreeBSD-blue.svg)](https://opensource.org/licenses/BSD-2-Clause)
[![CI](https://ci.codeberg.org/api/badges/mellium/communique-tui/status.svg)](https://ci.codeberg.org/mellium/communique-tui)

<a href="https://opencollective.com/mellium" alt="Donate on Open Collective"><img src="https://opencollective.com/mellium/donate/button@2x.png?color=blue" width="200"/></a>

![Screenshot](https://mellium.im/screenshot.png)

An instant messaging client written in [Go] for services that support the [XMPP
protocol] and the public Jabber network.


## Building

To build Communiqué you will need a supported Go version (see the `go.mod`
file).
If an appropriate version of Go is already installed, try running `make`.

If you'd like to contribute to the project, see [CONTRIBUTING.md].


## Translations

Translations can be found in the `locales/` tree and are licensed separately under
a Creative Commons Attribution 4.0 International License ([CC BY 4.0]).
To contribute to translations see the project on [Codeberg Translate].


## License

The package may be used under the terms of the BSD 2-Clause License a copy of
which may be found in the file "[LICENSE]".

Unless you explicitly state otherwise, any contribution submitted for inclusion
in the work by you shall be licensed as above, without any additional terms or
conditions.

[XMPP protocol]: https://xmpp.org
[CONTRIBUTING.md]: https://mellium.im/docs/CONTRIBUTING
[badge]: https://img.shields.io/badge/style-mellium%2fxmpp-green.svg?longCache=true&style=popout-square&label=issues
[Go]: https://golang.org/
[CC BY 4.0]: https://creativecommons.org/licenses/by/4.0/
[Codeberg Translate]: https://translate.codeberg.org/projects/mellium/communique/
[LICENSE]: https://codeberg.org/mellium/xmpp/src/branch/main/LICENSE
