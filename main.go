// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// The communiqué command is an instant messaging client with a terminal user
// interface.
//
// Communiqué is compatible with the Jabber network, or with any instant
// messaging service that speaks the XMPP protocol.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
	"github.com/rivo/tview"

	"mellium.im/communiqué/internal/ui"
)

const (
	appName = "communiqué"
)

// Set at build time while linking.
var (
	Version = "devel"
	Commit  = "unknown commit"
)

type config struct {
	JID     string `toml:"jid"`
	Verbose bool   `toml:"verbose"`

	Roster struct {
		HideJIDs bool `toml:"hide_jids"`
		Width    int  `toml:"width"`
	}
}

// configFile attempts to open the config file for reading.
// If a file is provided, only that file is checked, otherwise it attempts to
// open the following (falling back if the file does not exist or cannot be
// read):
//
// ./communiqué.toml, $XDG_CONFIG_HOME/communiqué/config.toml,
// $HOME/.config/communiqué/config.toml, /etc/communiqué/config.toml
func configFile(f string) (*os.File, string, error) {
	if f != "" {
		cfgFile, err := os.Open(f)
		return cfgFile, f, err
	}

	fPath := filepath.Join(".", appName+".toml")
	if cfgFile, err := os.Open(fPath); err == nil {
		return cfgFile, fPath, err
	}

	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir != "" {
		fPath = filepath.Join(cfgDir, appName)
		if cfgFile, err := os.Open(fPath); err == nil {
			return cfgFile, fPath, nil
		}
	}

	u, err := user.Current()
	if err != nil || u.HomeDir == "" {
		fPath = filepath.Join("/etc", appName)
		cfgFile, err := os.Open(fPath)
		return cfgFile, fPath, err
	}

	fPath = filepath.Join(u.HomeDir, ".config", appName)
	cfgFile, err := os.Open(fPath)
	return cfgFile, fPath, err
}

func main() {
	earlyLogs := &bytes.Buffer{}
	logger := log.New(io.MultiWriter(os.Stderr, earlyLogs), "", log.LstdFlags)
	debug := log.New(ioutil.Discard, "DEBUG ", log.LstdFlags)

	var (
		configPath string
	)
	flags := flag.NewFlagSet(appName, flag.ContinueOnError)
	flags.StringVar(&configPath, "f", configPath, "the config file to load")
	err := flags.Parse(os.Args[1:])
	if err != nil {
		logger.Printf("error parsing command line flags: %q", err)
	}

	f, fPath, err := configFile(configPath)
	if err != nil {
		logger.Printf("error loading config %q: %q", fPath, err)
	}
	cfg := config{}
	_, err = toml.DecodeReader(f, &cfg)
	if err != nil {
		logger.Printf("error parsing config file: %q", err)
	}

	app := tview.NewApplication()
	pane := ui.New(app,
		ui.ShowJIDs(!cfg.Roster.HideJIDs),
		ui.RosterWidth(cfg.Roster.Width),
		ui.Log(fmt.Sprintf(`%s %s (%s)
Go %s %s %s
`, string(appName[0]^0x20)+appName[1:], Version, Commit, runtime.Version(), runtime.GOOS, runtime.GOARCH)))
	_, err = io.Copy(pane, earlyLogs)
	logger.SetOutput(pane)
	if cfg.Verbose {
		debug.SetOutput(pane)
	}
	if err != nil {
		debug.Printf("Error copying early log data to output buffer: %q", err)
	}

	if err := app.SetRoot(pane, true).SetFocus(pane.Roster()).Run(); err != nil {
		panic(err)
	}
}
