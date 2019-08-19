// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
)

type theme struct {
	Name                        string `toml:"name"`
	PrimitiveBackgroundColor    string `toml:"primitive_background"`
	ContrastBackgroundColor     string `toml:"contrast_background"`
	MoreContrastBackgroundColor string `toml:"contrast_background"`
	BorderColor                 string `toml:"border"`
	TitleColor                  string `toml:"title"`
	GraphicsColor               string `toml:"graphics"`
	PrimaryTextColor            string `toml:"primary_text"`
	SecondaryTextColor          string `toml:"secondary_text"`
	TertiaryTextColor           string `toml:"tertiary_text"`
	InverseTextColor            string `toml:"inverse_text"`
	ContrastSecondaryTextColor  string `toml:"contrast_secondary_text"`
}

type config struct {
	JID     string `toml:"jid"`
	PassCmd string `toml:"password_eval"`
	KeyLog  string `toml:"keylog_file"`
	Timeout string `toml:"timeout"`

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
	/* #nosec */
	if cfgFile, err := os.Open(fPath); err == nil {
		return cfgFile, fPath, err
	}

	cfgDir := os.Getenv("XDG_CONFIG_HOME")
	if cfgDir != "" {
		fPath = filepath.Join(cfgDir, appName, configFileName)
		/* #nosec */
		if cfgFile, err := os.Open(fPath); err == nil {
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
	cfgFile, err := os.Open(fPath)
	return cfgFile, fPath, err
}
