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
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/rivo/tview"

	"mellium.im/communiqué/internal/ui"
	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/jid"
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
	PassCmd string `toml:"password_eval"`
	Verbose bool   `toml:"verbose"`
	KeyLog  string `toml:"keylog_file"`

	UI struct {
		HideStatus bool `toml:"hide_status"`
		Width      int  `toml:"width"`
	} `toml:"ui"`
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
		ui.ShowStatus(!cfg.UI.HideStatus),
		ui.RosterWidth(cfg.UI.Width),
		ui.Log(fmt.Sprintf(`%s %s (%s)
Go %s %s

`, string(appName[0]^0x20)+appName[1:], Version, Commit, runtime.Version(), runtime.Compiler)))
	_, err = io.Copy(pane, earlyLogs)
	logger.SetOutput(pane)
	if cfg.Verbose {
		debug.SetOutput(pane)
	}
	if err != nil {
		debug.Printf("Error copying early log data to output buffer: %q", err)
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		args := strings.Fields(cfg.PassCmd)
		if len(args) < 1 {
			logger.Println("No `password_eval' command specified in config file")
			return
		}
		logger.Printf("Running command: %q", cfg.PassCmd)
		pass, err := exec.CommandContext(ctx, args[0], args[1:]...).Output()
		if err != nil {
			logger.Println(err)
			return
		}
		client(ctx, cfg.JID, string(pass), cfg.KeyLog, logger, debug)
	}()

	if err := app.SetRoot(pane, true).SetFocus(pane.Roster()).Run(); err != nil {
		panic(err)
	}
}

func client(ctx context.Context, addr, pass, keylogFile string, logger, debug *log.Logger) {
	logger.Printf("User address: %q", addr)
	j, err := jid.Parse(addr)
	if err != nil {
		logger.Printf("Error parsing user address: %q", err)
	}

	conn, err := xmpp.DialClient(ctx, "tcp", j)
	if err != nil {
		logger.Printf("Error connecting to %q: %q", j.Domain(), err)
		return
	}

	var keylog io.Writer
	if keylogFile != "" {
		keylog, err = os.OpenFile(keylogFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0400)
		if err != nil {
			logger.Printf("Error creating keylog file: %q", err)
		}
	}
	_, err = xmpp.NewClientSession(
		ctx, j, "en", conn,
		xmpp.StartTLS(true, &tls.Config{
			ServerName:   j.Domain().String(),
			KeyLogWriter: keylog,
		}),
		xmpp.SASL("", pass, sasl.ScramSha256Plus, sasl.ScramSha1Plus, sasl.ScramSha256, sasl.ScramSha1),
		xmpp.BindResource(),
	)
	if err != nil {
		logger.Printf("Error establishing stream: %q", err)
		return
	}
}
