// Copyright 2018 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"os/user"
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
