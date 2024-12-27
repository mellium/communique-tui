// Copyright 2024 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"golang.org/x/text/message"
	"mellium.im/cli"
)

func aboutCmd(w io.Writer, cfgPath, version string, p *message.Printer, logger *log.Logger) *cli.Command {
	const cmdName = "about"
	flags := flag.NewFlagSet(cmdName, flag.ContinueOnError)
	var showBuildInfo bool
	flags.BoolVar(&showBuildInfo, "info", showBuildInfo, p.Sprintf("show embedded build information"))
	var (
		commit   string
		modified string
		vcs      string
	)
	buildInfo, ok := debug.ReadBuildInfo()
	if ok {
		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "vcs.revision":
				commit = setting.Value
			case "vcs.modified":
				if setting.Value == "true" {
					modified = "*"
				}
			case "vcs":
				vcs = setting.Value
			}
		}
	}

	// Search for the config file (if no file was explicitly provided), and make
	// sure the file can be read either way.
	f, cfgPath, err := configFile(cfgPath)
	if err != nil {
		cfgPath = ""
	}
	err = f.Close()
	if err != nil {
		logger.Println(err)
	}

	return &cli.Command{
		Usage:       cmdName + " [-info]",
		Description: p.Sprintf("Show information about this application."),
		Flags:       flags,
		Run: func(c *cli.Command, _ ...string) error {
			fmt.Fprintf(w, `Mellium (%s)

version:     %s
%s:    %s%s
go version:  %s
go compiler: %s
platform:    %s/%s
config file: %s
`,
				os.Args[0],
				version, strings.TrimSpace(vcs+" hash"), modified, commit,
				runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH,
				cfgPath)
			if showBuildInfo {
				if ok {
					fmt.Fprintf(w, `
build info:

%s`, buildInfo)
				} else {
					fmt.Fprintf(w, "\nFailed to read build info.\n")
				}
			}
			return nil
		},
	}
}
