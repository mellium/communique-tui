// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func getColor(name string) tcell.Color {
	if name == "default" || name == "" {
		return tcell.ColorDefault
	}
	return tcell.GetColor(name)
}

func colorName(color tcell.Color) string {
	for name, c := range tcell.ColorNames {
		if color == c {
			return name
		}
	}
	return fmt.Sprintf("#%06x", color.Hex())
}

func printConfig(w io.Writer) error {
	e := toml.NewEncoder(w)
	e.Indent = "\t"
	defConfig := config{
		Timeout: "30s",
		Theme: []theme{{
			Name:                        "default",
			PrimitiveBackgroundColor:    colorName(tview.Styles.PrimitiveBackgroundColor),
			ContrastBackgroundColor:     colorName(tview.Styles.ContrastBackgroundColor),
			MoreContrastBackgroundColor: colorName(tview.Styles.MoreContrastBackgroundColor),
			BorderColor:                 colorName(tview.Styles.BorderColor),
			TitleColor:                  colorName(tview.Styles.TitleColor),
			GraphicsColor:               colorName(tview.Styles.GraphicsColor),
			PrimaryTextColor:            colorName(tview.Styles.PrimaryTextColor),
			SecondaryTextColor:          colorName(tview.Styles.SecondaryTextColor),
			TertiaryTextColor:           colorName(tview.Styles.TertiaryTextColor),
			InverseTextColor:            colorName(tview.Styles.InverseTextColor),
			ContrastSecondaryTextColor:  colorName(tview.Styles.ContrastSecondaryTextColor),
		}},
	}
	defConfig.UI.Theme = "default"
	_, err := fmt.Fprintf(w, `# This is a config file for Communiqué.
# If the -f option is not provided, Communiqué will search for a config file in:
#
#   - ./communiqué.toml
#   - $XDG_CONFIG_HOME/communiqué/config.toml
#   - $HOME/.config/communiqué/config.toml
#   - /etc/communiqué/config.toml
#
# The only required field is "address". The "password_eval" field should be set
# to a command that writes the password to standard out. Normally this should
# decrypt an encrypted file containing the password. If it is not specified, the
# user will be prompted to enter a password.
`)
	if err != nil {
		return err
	}
	return e.Encode(defConfig)
}

type theme struct {
	Name                        string `toml:"name"`
	PrimitiveBackgroundColor    string `toml:"primitive_background"`
	ContrastBackgroundColor     string `toml:"contrast_background"`
	MoreContrastBackgroundColor string `toml:"more_contrast_background"`
	BorderColor                 string `toml:"border"`
	TitleColor                  string `toml:"title"`
	GraphicsColor               string `toml:"graphics"`
	PrimaryTextColor            string `toml:"primary_text"`
	SecondaryTextColor          string `toml:"secondary_text"`
	TertiaryTextColor           string `toml:"tertiary_text"`
	InverseTextColor            string `toml:"inverse_text"`
	ContrastSecondaryTextColor  string `toml:"contrast_secondary_text"`
}

type account struct {
	Address string `toml:"address"`
	Name    string `toml:"name"`
	PassCmd string `toml:"password_eval"`
	KeyLog  string `toml:"keylog_file"`
	DB      string `toml:"db_file"`
	NoSRV   bool   `toml:"disable_srv"`
	NoTLS   bool   `toml:"disable_tls"`
}

type config struct {
	DefaultAcct string    `toml:"default_account"`
	Timeout     string    `toml:"timeout"`
	Account     []account `toml:"account"`

	Log struct {
		Verbose bool `toml:"verbose"`
		XML     bool `toml:"xml"`
	} `toml:"log"`

	UI struct {
		HideStatus bool   `toml:"hide_status"`
		Theme      string `toml:"theme"`
		Width      int    `toml:"width"`
	} `toml:"ui"`

	Theme []theme `toml:"theme"`
}

const configFileName = "config.toml"

// configFile attempts to open the config file for reading.
// If a file is provided, only that file is checked, otherwise it attempts to
// open the following (falling back if the file does not exist or cannot be
// read):
//
// ./communiqué.toml, $XDG_CONFIG_HOME/communiqué/config.toml,
// $HOME/.config/communiqué/config.toml, /etc/communiqué/config.toml
func configFile(f string) (*os.File, string, error) {
	if f != "" {
		/* #nosec */
		cfgFile, err := os.Open(f)
		return cfgFile, f, err
	}

	fPath := filepath.Join(".", appName+".toml")
	cfgFile, err := os.Open(fPath) // #nosec G304
	if err == nil {
		return cfgFile, fPath, err
	}

	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir != "" {
		fPath = filepath.Join(cfgDir, appName, configFileName)
		cfgFile, err := os.Open(fPath) // #nosec G304
		if err == nil {
			return cfgFile, fPath, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, "", err
	}

	if home != "" {
		fPath = filepath.Join(home, ".config", appName, configFileName)
		/* #nosec */
		cfgFile, err := os.Open(fPath)
		if err == nil {
			return cfgFile, fPath, err
		}
	}

	fPath = filepath.Join("/etc", appName, configFileName)
	/* #nosec */
	cfgFile, err = os.Open(fPath)
	return cfgFile, fPath, err
}
