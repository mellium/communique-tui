// Copyright 2024 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	_ "embed"

	"mellium.im/communique/internal/storage"
)

//go:embed schema.sql
var schema string

// Migrations is a collection of migrations for upgrading the database.
// It automatically checks the expected schema version of the application and
// orders itself to upgrade or downgrade the database to the correct version.
// The offset is the version,
func Migrations() []storage.Migration {
	return []storage.Migration{
		{
			Version: 1,
			Up:      schema,
			Down: `
			PRAGMA writable_schema = 1;
			delete from sqlite_master where type in ('view', 'table', 'index', 'trigger');
			PRAGMA writable_schema = 0;`,
		},
	}
}
