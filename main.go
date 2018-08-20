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
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/gdamore/tcell"
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

func main() {
	earlyLogs := &bytes.Buffer{}
	logger := log.New(io.MultiWriter(os.Stderr, earlyLogs), "", log.LstdFlags)
	debug := log.New(ioutil.Discard, "DEBUG ", log.LstdFlags)
	var xmlInLog, xmlOutLog *log.Logger

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
		logger.Println(err)
	}
	cfg := config{}
	_, err = toml.DecodeReader(f, &cfg)
	if err != nil {
		logger.Printf("error parsing config file: %q", err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)

	pages := tview.NewPages()
	app := tview.NewApplication()
	quitModal := tview.NewModal().
		SetText("Are you sure you want to quit?").
		AddButtons([]string{"Quit", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 0 {
				app.Stop()
			}
			pages.HidePage("quit")
		})
	pane := ui.New(app,
		ui.ShowStatus(!cfg.UI.HideStatus),
		ui.RosterWidth(cfg.UI.Width),
		ui.Log(fmt.Sprintf(`%s %s (%s)
Go %s %s

`, string(appName[0]^0x20)+appName[1:], Version, Commit, runtime.Version(), runtime.Compiler)))

	pages.AddPage("ui", pane, true, true)
	pages.AddPage("quit", quitModal, true, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// The application intercepts Ctrl-C by default and terminates itself. We
		// don't want Ctrl-C to stop the application, so disable this behavior by
		// default. Manually sending a SIGINT will still work (see the signal
		// handling goroutine in this file).
		if event.Key() == tcell.KeyCtrlC {
			return nil
		}

		if event.Rune() == 'q' {
			pages.ShowPage("quit")
			app.Draw()
			return nil
		}
		return event
	})

	if cfg.Log.XML {
		xmlInLog = log.New(pane, "RECV ", log.LstdFlags)
		xmlOutLog = log.New(pane, "SENT ", log.LstdFlags)
	}

	_, err = io.Copy(pane, earlyLogs)
	logger.SetOutput(pane)
	if cfg.Log.Verbose {
		debug.SetOutput(pane)
	}
	if err != nil {
		debug.Printf("Error copying early log data to output buffer: %q", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	getPass := func(ctx context.Context) (string, error) {
		var pass []byte
		args := strings.Fields(cfg.PassCmd)
		if len(args) < 1 {
			// TODO: No command was specified, prompt for a password.
			return "", fmt.Errorf(`No password command specified, edit %q and add:

	password_eval="password_command"

`, fPath)
		} else {
			debug.Printf("Running command: %q", cfg.PassCmd)
			pass, err = exec.CommandContext(ctx, args[0], args[1:]...).Output()
			if err != nil {
				return "", err
			}
		}
		return string(pass), nil
	}

	c := newClient(fPath, cfg.JID, cfg.KeyLog, pane, xmlInLog, xmlOutLog, logger, debug, getPass)
	pane.Handle(newUIHandler(c, debug, logger))
	pane.Offline()

	go c.Online(ctx)

	go func() {
		s := <-sigs
		debug.Printf("Got signal: %v", s)
		app.Stop()
	}()
	if err := app.SetRoot(pages, true).SetFocus(pane.Roster()).Run(); err != nil {
		panic(err)
	}
}
