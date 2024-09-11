// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// The communiqué command is an instant messaging client with a terminal user
// interface.
//
// Communiqué is compatible with the Jabber network, or with any instant
// messaging service that speaks the XMPP protocol.
package main // import "mellium.im/communique"

//go:generate go run -tags=tools golang.org/x/text/cmd/gotext update -out catalog.go

import (
	"bytes"
	"context"
	"crypto/tls"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/BurntSushi/toml"
	"github.com/rivo/tview"

	"mellium.im/communique/internal/client"
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

func printHelp(flags *flag.FlagSet, w io.Writer, p *message.Printer) {
	flags.SetOutput(w)
	_, err := p.Fprintf(w, `Usage of communiqué:

`)
	if err != nil {
		panic(err)
	}
	flags.PrintDefaults()
}

func main() {
	// Setup internationalization and translations.
	lang := os.Getenv("LC_ALL")
	if lang == "" {
		lang = os.Getenv("LANG")
	}
	if lang == "" {
		lang = "en"
	} else {
		lang, _, _ = strings.Cut(lang, ".")
	}
	langTag := language.Make(lang)
	langTag, _, _ = message.DefaultCatalog.Matcher().Match(langTag)
	p := message.NewPrinter(langTag)

	earlyLogs := &bytes.Buffer{}
	logger := log.New(io.MultiWriter(os.Stderr, earlyLogs), "", log.LstdFlags)
	debug := log.New(io.Discard, p.Sprintf("DEBUG")+" ", log.LstdFlags)
	xmlInLog := log.New(io.Discard, p.Sprintf("RECV")+" ", log.LstdFlags)
	xmlOutLog := log.New(io.Discard, p.Sprintf("SENT")+" ", log.LstdFlags)

	var (
		configPath string
		defAcct    string
		h          bool
		help       bool
		genConfig  bool
	)
	flags := flag.NewFlagSet(appName, flag.ContinueOnError)
	flags.StringVar(&configPath, "f", configPath, p.Sprintf("the config file to load"))
	flags.StringVar(&defAcct, "account", defAcct, p.Sprintf("override the account set in the config file"))
	flags.BoolVar(&h, "h", h, p.Sprintf("print this help message"))
	flags.BoolVar(&help, "help", help, p.Sprintf("print this help message"))
	flags.BoolVar(&genConfig, "config", genConfig, p.Sprintf("print a default config file to stdout"))
	// Even with ContinueOnError set, it still prints for some reason. Discard the
	// first defaults so we can write our own.
	flags.SetOutput(io.Discard)
	err := flags.Parse(os.Args[1:])
	if err != nil {
		logger.Println(err)
		printHelp(flags, os.Stderr, p)
		os.Exit(2)
	}

	if help || h {
		printHelp(flags, os.Stdout, p)
		return
	}

	if genConfig {
		err = printConfig(os.Stdout)
		if err != nil {
			logger.Fatal(p.Sprintf("Error encoding default config as TOML: %v", err))
		}
		return
	}

	f, fpath, err := configFile(configPath)
	if err != nil {
		logger.Fatalf(p.Sprintf(`%v

Try running '%s -config' to generate a default config file.`, err, os.Args[0]))
	}
	cfg := config{}
	_, err = toml.NewDecoder(f).Decode(&cfg)
	if err != nil {
		logger.Print(p.Sprintf("error parsing config file: %v", err))
	}
	if err = f.Close(); err != nil {
		logger.Print(p.Sprintf("error closing config file: %v", err))
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
		logger.Fatal(p.Sprintf("account %q not found in config file", cfg.DefaultAcct))
	}

	// Open the database
	dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	account, err := jid.Parse(acct.Address)
	if err != nil {
		logger.Fatal(p.Sprintf("error parsing main account as XMPP address: %v", err))
	}
	db, err := storage.OpenDB(dbCtx, appName, account.Bare().String(), acct.DB, schema, debug)
	if err != nil {
		logger.Fatal(p.Sprintf("error opening database: %v", err))
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
		p,
		logger,
		ui.Debug(debug),
		ui.Addr(acct.Address),
		ui.ShowStatus(!cfg.UI.HideStatus),
		ui.RosterWidth(cfg.UI.Width))

	if cfg.Log.XML {
		xmlInLog.SetOutput(pane)
		xmlOutLog.SetOutput(pane)
	}

	_, err = fmt.Fprintf(pane, `%s %s (%s)
Go %s %s

`, string(appName[0]^0x20)+appName[1:], Version, Commit, runtime.Version(), runtime.Compiler)
	if err != nil {
		debug.Print(p.Sprintf("error logging to pane: %v", err))
	}

	_, err = io.Copy(pane, earlyLogs)
	logger.SetOutput(pane)
	if cfg.Log.Verbose {
		debug.SetOutput(pane)
	}
	if err != nil {
		debug.Print(p.Sprintf("error copying early log data to output buffer: %q", err))
	}

	pass := &bytes.Buffer{}
	if len(acct.PassCmd) > 0 {
		args := strings.Fields(acct.PassCmd)
		debug.Print(p.Sprintf("running command: %q", acct.PassCmd))
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
			debug.Print(p.Sprintf("error running password command, falling back to prompt: %v", err))
		}
	}
	getPass := func(ctx context.Context) (string, error) {
		if p := pass.String(); p != "" {
			return strings.TrimSuffix(p, "\n"), nil
		}
		if account.Localpart() == "" {
			return "", nil
		}
		return pane.ShowPasswordPrompt(), nil
	}

	var j jid.JID
	if cfg.DefaultAcct == "" {
		logger.Print(p.Sprintf(`no user address specified, edit %q and add:

	jid="me@example.com"

`, fpath))
	} else {
		logger.Print(p.Sprintf("user address: %q", cfg.DefaultAcct))
		j, err = jid.Parse(cfg.DefaultAcct)
		if err != nil {
			logger.Print(p.Sprintf("error parsing user address: %q", err))
		}
	}
	timeout := 30 * time.Second
	if cfg.Timeout != "" {
		timeout, err = time.ParseDuration(cfg.Timeout)
		if err != nil {
			logger.Print(p.Sprintf("error parsing timeout, defaulting to 30s: %q", err))
		}
	}
	// cfg.KeyLog
	var keylog io.Writer
	if acct.KeyLog != "" {
		keylog, err = os.OpenFile(acct.KeyLog, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0400)
		if err != nil {
			logger.Print(p.Sprintf("error creating keylog file: %q", err))
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
	var rosterVer string
	func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		rosterVer, err = db.RosterVer(ctx)
		if err != nil {
			logger.Print(p.Sprintf("error retrieving roster version, falling back to full roster fetch: %v", err))
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
		client.Printer(p),
	)
	c.Handler(newClientHandler(c, pane, db, logger, debug))
	pane.Handle(newUIHandler(acct, pane, db, c, logger, debug))

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
			logger.Print(p.Sprintf("initial login failed: %v", err))
			return
		}
		debug.Print(p.Sprintf("logged in as: %q", c.LocalAddr()))
	}()

	go func() {
		s := <-sigs
		debug.Print(p.Sprintf("got signal: %v", s))
		pane.Stop()
	}()

	defer pane.Stop()
	if err := pane.Run(); err != nil {
		panic(err)
	}
}
