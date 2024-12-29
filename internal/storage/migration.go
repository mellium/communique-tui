// Copyright 2024 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package storage

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"iter"
	"log"
	"slices"

	"golang.org/x/text/message"
)

// Migration is a schema version and scripts to upgrade and downgrade to or from
// that schema version.
type Migration struct {
	Version uint
	Up      string
	Down    string
}

// Migrations is a sequence of schema versions and scripts that convert from the
// current schema to the new schema.
type Migrations []Migration

// Run returns an iterator over all migration scripts that should be run to
// upgrade or downgrade from the current version to the target version in the
// order they should be run.
// Calling it will sort the slice.
func (m Migrations) Run(current, target uint) iter.Seq2[uint, string] {
	switch cmp.Compare(target, current) {
	case -1:
		// If target < current (ie. we're downgrading the schema)
		slices.SortFunc(m, func(a, b Migration) int {
			return cmp.Compare(b.Version, a.Version)
		})
		return func(yield func(uint, string) bool) {
			for _, cur := range m {
				if cur.Version > current {
					continue
				}
				if cur.Version <= target {
					return
				}
				if !yield(cur.Version, cur.Down) {
					return
				}
			}
		}
	case 0:
		// If target == current (ie. nothing to do here)
		return func(func(uint, string) bool) {}
	case 1:
		// If target > current (ie. we're upgrading the schema)
		// Re-sort from highest to lowest.
		slices.SortFunc(m, func(a, b Migration) int {
			return cmp.Compare(a.Version, b.Version)
		})
		return func(yield func(uint, string) bool) {
			for _, cur := range m {
				if cur.Version <= current {
					continue
				}
				if !yield(cur.Version, cur.Up) {
					return
				}
				if cur.Version >= target {
					return
				}
			}
		}
	}
	panic("this should be impossible to reach")
}

// runMigrations runs migrations until the schema version of db matches the
// target schema version.
// Migrations are run in a transaction that is rolled back if any errors occur.
func runMigrations(ctx context.Context, db *sql.DB, target uint, m Migrations, p *message.Printer, debug *log.Logger) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	var currentID uint
	err = tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&currentID)
	if err != nil {
		return err
	}
	debug.Println(p.Sprintf("starting migration from %d to %d…", currentID, target))
	for userVersion, script := range m.Run(currentID, target) {
		debug.Println(p.Sprintf("running migration %d→%d…", currentID, target))
		_, err := tx.ExecContext(ctx, script)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version=%d", userVersion))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
