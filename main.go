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
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"

	"mellium.im/communiqué/internal/ui"

	"github.com/BurntSushi/toml"
	"github.com/rivo/tview"
)

const (
	app = "communiqué"
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

	fPath := filepath.Join(".", app+".toml")
	if cfgFile, err := os.Open(fPath); err == nil {
		return cfgFile, fPath, err
	}

	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir != "" {
		fPath = filepath.Join(cfgDir, app)
		if cfgFile, err := os.Open(fPath); err == nil {
			return cfgFile, fPath, nil
		}
	}

	u, err := user.Current()
	if err != nil || u.HomeDir == "" {
		fPath = filepath.Join("/etc", app)
		cfgFile, err := os.Open(fPath)
		return cfgFile, fPath, err
	}

	fPath = filepath.Join(u.HomeDir, ".config", app)
	cfgFile, err := os.Open(fPath)
	return cfgFile, fPath, err
}

func main() {
	logger := log.New(os.Stderr, app+" ", log.LstdFlags)
	debug := log.New(ioutil.Discard, app+" DEBUG ", log.LstdFlags)

	var (
		configPath string
	)
	flags := flag.NewFlagSet(app, flag.ContinueOnError)
	flags.StringVar(&configPath, "f", configPath, "the config file to load")
	err := flags.Parse(os.Args[1:])
	if err != nil {
		logger.Fatalf("error parsing command line flags: %q", err)
	}

	f, fPath, err := configFile(configPath)
	if err != nil {
		logger.Fatalf("error loading config %q: %q", fPath, err)
	}
	cfg := config{}
	_, err = toml.DecodeReader(f, &cfg)
	if err != nil {
		logger.Fatalf("error parsing config file: %q", err)
	}
	if cfg.Verbose {
		debug.SetOutput(os.Stderr)
	}
	debug.Printf("Parsed config as: `%+v'", cfg)

	app := tview.NewApplication()
	pane := ui.New(app,
		ui.ShowJIDs(!cfg.Roster.HideJIDs),
		ui.RosterWidth(cfg.Roster.Width),
	)

	if err := app.SetRoot(pane, true).SetFocus(pane.Roster()).Run(); err != nil {
		panic(err)
	}
}
