// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// The communiqué command is an instant messaging client with a terminal user
// interface.
//
// Communiqué is compatible with the Jabber network, or with any instant
// messaging service that speaks the XMPP protocol.
package main // import "mellium.im/communiqué"

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

func printHelp(flags *flag.FlagSet, w io.Writer) {
	flags.SetOutput(w)
	fmt.Println(`Usage of communiqué:`)
	flags.PrintDefaults()
	return
}

func main() {
	earlyLogs := &bytes.Buffer{}
	logger := log.New(io.MultiWriter(os.Stderr, earlyLogs), "", log.LstdFlags)
	debug := log.New(ioutil.Discard, "DEBUG ", log.LstdFlags)
	var xmlInLog, xmlOutLog *log.Logger

	var (
		configPath string
		h          bool
		help       bool
	)
	flags := flag.NewFlagSet(appName, flag.ContinueOnError)
	flags.StringVar(&configPath, "f", configPath, "the config file to load")
	flags.BoolVar(&h, "h", h, "print this help message")
	flags.BoolVar(&help, "help", help, "print this help message")
	// Even with ContinueOnError set, it still prints for some reason. Discard the
	// first defaults so we can write our own.
	flags.SetOutput(ioutil.Discard)
	err := flags.Parse(os.Args[1:])
	if err != nil {
		logger.Println(err)
		printHelp(flags, os.Stderr)
		os.Exit(2)
	}

	if help || h {
		printHelp(flags, os.Stdout)
		return
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

	// Setup the global tview styles. I hate this.
	var cfgTheme *theme
	for _, t := range cfg.Theme {
		if t.Name == cfg.UI.Theme {
			cfgTheme = &t
			break
		}
	}
	if cfgTheme != nil {
		tview.Styles.PrimitiveBackgroundColor = tcell.GetColor(cfgTheme.PrimitiveBackgroundColor)
		tview.Styles.ContrastBackgroundColor = tcell.GetColor(cfgTheme.ContrastBackgroundColor)
		tview.Styles.MoreContrastBackgroundColor = tcell.GetColor(cfgTheme.MoreContrastBackgroundColor)
		tview.Styles.BorderColor = tcell.GetColor(cfgTheme.BorderColor)
		tview.Styles.TitleColor = tcell.GetColor(cfgTheme.TitleColor)
		tview.Styles.GraphicsColor = tcell.GetColor(cfgTheme.GraphicsColor)
		tview.Styles.PrimaryTextColor = tcell.GetColor(cfgTheme.PrimaryTextColor)
		tview.Styles.SecondaryTextColor = tcell.GetColor(cfgTheme.SecondaryTextColor)
		tview.Styles.TertiaryTextColor = tcell.GetColor(cfgTheme.TertiaryTextColor)
		tview.Styles.InverseTextColor = tcell.GetColor(cfgTheme.InverseTextColor)
		tview.Styles.ContrastSecondaryTextColor = tcell.GetColor(cfgTheme.ContrastSecondaryTextColor)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)

	app := tview.NewApplication()
	pane := ui.New(app,
		ui.Addr(cfg.JID),
		ui.ShowStatus(!cfg.UI.HideStatus),
		ui.RosterWidth(cfg.UI.Width),
		ui.Log(fmt.Sprintf(`%s %s (%s)
Go %s %s

`, string(appName[0]^0x20)+appName[1:], Version, Commit, runtime.Version(), runtime.Compiler)))

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// The application intercepts Ctrl-C by default and terminates itself. We
		// don't want Ctrl-C to stop the application, so disable this behavior by
		// default. Manually sending a SIGINT will still work (see the signal
		// handling goroutine in this file).
		if event.Key() == tcell.KeyCtrlC {
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

	getPass := func(ctx context.Context) (string, error) {
		args := strings.Fields(cfg.PassCmd)
		if len(args) < 1 {
			return pane.ShowPasswordPrompt(), nil
		}

		debug.Printf("Running command: %q", cfg.PassCmd)
		pass, err := exec.CommandContext(ctx, args[0], args[1:]...).Output()
		if err != nil {
			return "", err
		}
		return string(pass), nil
	}

	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		logger.Printf("Error parsing timeout, defaulting to 30s: %q", err)
		timeout = 30 * time.Second
	}
	c := newClient(timeout, fPath, cfg.JID, cfg.KeyLog, pane, xmlInLog, xmlOutLog, logger, debug, getPass)
	pane.Handle(newUIHandler(c, debug, logger))

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		c.Online(ctx)
	}()

	go func() {
		s := <-sigs
		debug.Printf("Got signal: %v", s)
		app.Stop()
	}()
	if err := app.SetRoot(pane, true).SetFocus(pane).Run(); err != nil {
		panic(err)
	}
}
