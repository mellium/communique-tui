// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// The communiqué command is an instant messaging client with a terminal user
// interface.
//
// Communiqué is compatible with the Jabber network, or with any instant
// messaging service that speaks the XMPP protocol.
package main // import "mellium.im/communique"

import (
	"bytes"
	"context"
	"crypto/tls"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"mellium.im/communique/internal/client"
	"mellium.im/communique/internal/client/gst"
	"mellium.im/communique/internal/gui"
	"mellium.im/communique/internal/logwriter"
	"mellium.im/communique/internal/storage"
	"mellium.im/communique/internal/ui"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
)

const (
	appName = "communiqué"
)

//go:embed schema.sql
var schema string

// Set at build time while linking.
var (
	Version = "devel"
	Commit  = "unknown commit"
)

func printHelp(flags *flag.FlagSet, w io.Writer) {
	flags.SetOutput(w)
	fmt.Fprint(w, `Usage of communiqué:

`)
	flags.PrintDefaults()
}

func main() {
	earlyLogs := &bytes.Buffer{}
	logger := log.New(io.MultiWriter(os.Stderr, earlyLogs), "", log.LstdFlags)
	debug := log.New(ioutil.Discard, "DEBUG ", log.LstdFlags)
	xmlInLog := log.New(ioutil.Discard, "RECV ", log.LstdFlags)
	xmlOutLog := log.New(ioutil.Discard, "SENT ", log.LstdFlags)

	var (
		configPath string
		defAcct    string
		h          bool
		help       bool
		genConfig  bool
		useQuic    bool
		useGui     bool
	)
	flags := flag.NewFlagSet(appName, flag.ContinueOnError)
	flags.StringVar(&configPath, "f", configPath, "the config file to load")
	flags.StringVar(&defAcct, "account", defAcct, "override the account set in the config file")
	flags.BoolVar(&h, "h", h, "print this help message")
	flags.BoolVar(&help, "help", help, "print this help message")
	flags.BoolVar(&genConfig, "config", genConfig, "print a default config file to stdout")
	flags.BoolVar(&useQuic, "quic", useQuic, "Start session over quic (XEP-0467)")
	flags.BoolVar(&useGui, "gui", useGui, "Start session with fyne GUI")
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

	if genConfig {
		err = printConfig(os.Stdout)
		if err != nil {
			logger.Fatalf("Error encoding default config as TOML: %v", err)
		}
		return
	}

	// GUI Code is here
	if useGui {
		gst.GstreamerInit()
		// TODO: Implement windows to display different type of debug output
		// Currently displaying all in one main terminal
		debug.SetOutput(io.MultiWriter(earlyLogs, os.Stderr))
		xmlInLog.SetOutput(io.MultiWriter(earlyLogs, os.Stderr))
		xmlOutLog.SetOutput(io.MultiWriter(earlyLogs, os.Stderr))

		timeout := 30 * time.Second

		window := gui.New(debug)

		jidChan := make(chan *gui.LoginData)
		var db *storage.DB

		go func() {
			logininp := <-jidChan

			debug.Printf("JID Account: %s", logininp.JID)
			if logininp.JID == "" {
				return
			}

			// Open the database
			dbCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			account, err := jid.Parse(logininp.JID)
			if err != nil {
				logger.Fatalf("error parsing main account as XMPP address: %v", err)
			}
			db, err = storage.OpenDB(dbCtx, appName, account.Bare().String(), "", schema, debug)
			if err != nil {
				logger.Fatalf("error opening database: %v", err)
			}

			// Get current roster version
			var rosterVer string
			func() {
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()
				rosterVer, err = db.RosterVer(ctx)
				if err != nil {
					logger.Printf("error retrieving roster version, falling back to full roster fetch: %v", err)
				}
			}()

			j, err := jid.Parse(logininp.JID)
			if err != nil {
				logger.Printf("error parsing user address: %q", err)
			}

			getPass := func(ctx context.Context) (string, error) {
				return logininp.Pass, nil
			}

			dialer := &dial.Dialer{
				TLSConfig: &tls.Config{
					ServerName: j.Domain().String(),
					MinVersion: tls.VersionTLS12,
				},
				NoLookup: false,
				NoTLS:    true,
			}

			c := client.New(
				j, logger, debug,
				client.Timeout(timeout),
				client.Dialer(dialer),
				client.NoTLS(true),
				client.Tee(logwriter.New(xmlInLog), logwriter.New(xmlOutLog)),
				client.Password(getPass),
				client.RosterVer(rosterVer),
				client.Quic(useQuic),
			)

			c.Handler(newXMPPClientHandler(window, db, c, logger, debug))
			window.Handler(newFyneGUIHandler(window, db, c, logger, debug))

			ctx, cancel := context.WithTimeout(context.Background(), 3*timeout)
			defer cancel()
			if err := c.Online(ctx); err != nil {
				logger.Printf("initial login failed: %v", err)
				return
			}
			debug.Printf("logged in as: %q", c.LocalAddr())
		}()

		window.Run(jidChan)
		db.Close()
		return
	}

	f, fpath, err := configFile(configPath)
	if err != nil {
		logger.Fatalf(`%v

Try running '%s -config' to generate a default config file.`, err, os.Args[0])
	}
	cfg := config{}
	_, err = toml.DecodeReader(f, &cfg)
	if err != nil {
		logger.Printf("error parsing config file: %v", err)
	}
	if err = f.Close(); err != nil {
		logger.Printf("error closing config file: %v", err)
	}

	if cfg.Log.Verbose {
		debug.SetOutput(io.MultiWriter(earlyLogs, os.Stderr))
	}

	if defAcct != "" {
		cfg.DefaultAcct = defAcct
	}

	var acct account
	for _, a := range cfg.Account {
		if a.Address == cfg.DefaultAcct {
			acct = a
			break
		}
	}
	if acct.Address == "" {
		logger.Fatalf("account %q not found in config file", cfg.DefaultAcct)
	}

	// Open the database
	dbCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	account, err := jid.Parse(acct.Address)
	if err != nil {
		logger.Fatalf("error parsing main account as XMPP address: %v", err)
	}
	db, err := storage.OpenDB(dbCtx, appName, account.Bare().String(), acct.DB, schema, debug)
	if err != nil {
		logger.Fatalf("error opening database: %v", err)
	}
	defer db.Close()

	// Setup the global tview styles. I hate this.
	var cfgTheme *theme
	for i := range cfg.Theme {
		t := cfg.Theme[i]
		if t.Name == cfg.UI.Theme {
			cfgTheme = &t
			break
		}
	}
	if cfgTheme != nil {
		tview.Styles.PrimitiveBackgroundColor = getColor(cfgTheme.PrimitiveBackgroundColor)
		tview.Styles.ContrastBackgroundColor = getColor(cfgTheme.ContrastBackgroundColor)
		tview.Styles.MoreContrastBackgroundColor = getColor(cfgTheme.MoreContrastBackgroundColor)
		tview.Styles.BorderColor = getColor(cfgTheme.BorderColor)
		tview.Styles.TitleColor = getColor(cfgTheme.TitleColor)
		tview.Styles.GraphicsColor = getColor(cfgTheme.GraphicsColor)
		tview.Styles.PrimaryTextColor = getColor(cfgTheme.PrimaryTextColor)
		tview.Styles.SecondaryTextColor = getColor(cfgTheme.SecondaryTextColor)
		tview.Styles.TertiaryTextColor = getColor(cfgTheme.TertiaryTextColor)
		tview.Styles.InverseTextColor = getColor(cfgTheme.InverseTextColor)
		tview.Styles.ContrastSecondaryTextColor = getColor(cfgTheme.ContrastSecondaryTextColor)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)

	pane := ui.New(
		ui.Debug(debug),
		ui.Addr(acct.Address),
		ui.ShowStatus(!cfg.UI.HideStatus),
		ui.RosterWidth(cfg.UI.Width),
		ui.InputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			// The application intercepts Ctrl-C by default and terminates itself. We
			// don't want Ctrl-C to stop the application, so disable this behavior by
			// default. Manually sending a SIGINT will still work (see the signal
			// handling goroutine in this file).

			if event.Key() == tcell.KeyCtrlC {
				return nil
			}
			return event
		}))

	if cfg.Log.XML {
		xmlInLog.SetOutput(pane)
		xmlOutLog.SetOutput(pane)
	}

	_, err = fmt.Fprintf(pane, `%s %s (%s)
Go %s %s

`, string(appName[0]^0x20)+appName[1:], Version, Commit, runtime.Version(), runtime.Compiler)
	if err != nil {
		debug.Printf("error logging to pane: %v", err)
	}

	_, err = io.Copy(pane, earlyLogs)
	logger.SetOutput(pane)
	if cfg.Log.Verbose {
		debug.SetOutput(pane)
	}
	if err != nil {
		debug.Printf("error copying early log data to output buffer: %q", err)
	}

	pass := &bytes.Buffer{}
	if len(acct.PassCmd) > 0 {
		args := strings.Fields(acct.PassCmd)
		debug.Printf("running command: %q", acct.PassCmd)
		// The config file is considered a safe source since it is never written
		// except by the user, so consider this use of exec to be safe.
		/* #nosec */
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stderr = io.MultiWriter(os.Stderr, pane)
		cmd.Stdout = pass
		/* #nosec */
		err := cmd.Run()
		if err != nil {
			debug.Printf("error running password command, falling back to prompt: %v", err)
		}
	}
	getPass := func(ctx context.Context) (string, error) {
		if p := pass.String(); p != "" {
			return strings.TrimSuffix(p, "\n"), nil
		}
		return pane.ShowPasswordPrompt(), nil
	}

	var j jid.JID
	if cfg.DefaultAcct == "" {
		logger.Printf(`no user address specified, edit %q and add:

	jid="me@example.com"

`, fpath)
	} else {
		logger.Printf("user address: %q", cfg.DefaultAcct)
		j, err = jid.Parse(cfg.DefaultAcct)
		if err != nil {
			logger.Printf("error parsing user address: %q", err)
		}
	}
	timeout := 30 * time.Second
	if cfg.Timeout != "" {
		timeout, err = time.ParseDuration(cfg.Timeout)
		if err != nil {
			logger.Printf("error parsing timeout, defaulting to 30s: %q", err)
		}
	}
	// cfg.KeyLog
	var keylog io.Writer
	if acct.KeyLog != "" {
		keylog, err = os.OpenFile(acct.KeyLog, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0400)
		if err != nil {
			logger.Printf("error creating keylog file: %q", err)
		}
	}
	dialer := &dial.Dialer{
		TLSConfig: &tls.Config{
			ServerName:   j.Domain().String(),
			KeyLogWriter: keylog,
			MinVersion:   tls.VersionTLS12,
		},
		NoLookup: acct.NoSRV,
		NoTLS:    acct.NoTLS,
	}
	configPath = path.Dir(fpath)
	var rosterVer string
	func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		rosterVer, err = db.RosterVer(ctx)
		if err != nil {
			logger.Printf("error retrieving roster version, falling back to full roster fetch: %v", err)
		}
	}()
	c := client.New(
		j, logger, debug,
		client.Timeout(timeout),
		client.Dialer(dialer),
		client.NoTLS(acct.NoTLS),
		client.Tee(logwriter.New(xmlInLog), logwriter.New(xmlOutLog)),
		client.Password(getPass),
		client.RosterVer(rosterVer),
		client.Quic(useQuic),
	)
	c.Handler(newClientHandler(configPath, c, pane, db, logger, debug))
	pane.Handle(newUIHandler(configPath, acct, pane, db, c, logger, debug))

	// Hopefully nothing ever panics, but in case it does ensure that we exit
	// TUI mode so that we don't hose the users terminal.
	//defer func() {
	//	// TODO: this isn't great because we lose the stack trace. Update the
	//	// error handling so that we can attempt to recover a trace from the
	//	// error.
	//	if r := recover(); r != nil {
	//		pane.Stop()
	//		panic(r)
	//	}
	//}()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*timeout)
		defer cancel()
		if err := c.Online(ctx); err != nil {
			logger.Printf("initial login failed: %v", err)
			return
		}
		debug.Printf("logged in as: %q", c.LocalAddr())
	}()

	go func() {
		s := <-sigs
		debug.Printf("got signal: %v", s)
		pane.Stop()
	}()

	defer pane.Stop()
	if err := pane.Run(); err != nil {
		panic(err)
	}
}
