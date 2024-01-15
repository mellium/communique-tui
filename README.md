# Communiqué gui

An instant messaging client written in [Go] for services that support the [XMPP
protocol] and the public Jabber network. This version of Communiqué add some new feature.
- Graphical User Interface with [Fyne]
- Jingle Extension
    - XEP-0166: Jingle
    - XEP-0167: Jingle RTP Sessions
    - XEP-0176: Jingle ICE-UDP Transport Method
    - XEP-0293: Jingle RTP Feedback Negotiation
    - XEP-0320: Use of DTLS-SRTP in Jingle Sessions
    - XEP-0338: Jingle Grouping Framework
    - XEP-0339: Source-Specific Media Attributes in Jingle
- Video Call using Jingle and [Pion WebRTC]
- OMEMO Encryption

## Development Environment
This project use gstreamer for media handling and Fyne for GUI library. 
We need to install some c library dependencies for both of them before 
building the project.

### Windows
You can use [MSYS2] to easily set up development environments on Windows.
We will be using MSYS2 MinGW 64-bit for building instruction next.

Make sure MSYS2 is up-to-date before installing new packages.
```bash
pacman -Syu
```

The following will install some basic toolchain and git.
```bash
pacman -S git mingw-w64-x86_64-toolchain
```

The following will install gstreamer and some plugins.
```bash
pacman -S mingw-w64-x86_64-gstreamer mingw-w64-x86_64-gst-devtools mingw-w64-x86_64-gst-plugins-{base,good,bad,ugly}
```

### Linux

For Fyne, you will need Go, gcc and the graphic library header files using your package manager. For gstreamer, it is pretty much similiar to MSYS2.

#### Debian / Ubuntu
```bash
sudo apt-get install golang gcc libgl1-mesa-dev xorg-dev
sudo apt install libgstreamer1.0-0 gstreamer1.0-plugins-{base,good,bad,ugly} gstreamer1.0-tools
```

#### Arch
```bash
sudo pacman -S go xorg-server-devel libxcursor libxrandr libxinerama libxi
sudo pacman -S gstreamer gst-plugins-{base,good,bad,ugly}
```

## Building

To build Communiqué you will need a supported Go version (see the `go.mod`
file).
If an appropriate version of Go is already installed, try running `make`.

## License

The package may be used under the terms of the BSD 2-Clause License a copy of
which may be found in the file "[LICENSE]".

Unless you explicitly state otherwise, any contribution submitted for inclusion
in the work by you shall be licensed as above, without any additional terms or
conditions.

[XMPP protocol]: https://xmpp.org
[Go]: https://golang.org/
[Fyne]: https://fyne.io
[Pion WebRTC]: https://github.com/pion/webrtc
[MSYS2]: https://www.msys2.org/