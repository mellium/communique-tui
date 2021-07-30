// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package storage implements the database layer of the client.
package storage // import "mellium.im/communique/internal/storage"

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"mellium.im/communique/internal/client/event"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// DB represents a SQL database with common pre-prepared statements.
type DB struct {
	*sql.DB
	insertSID *sql.Stmt
	insertMsg *sql.Stmt
	markRecvd *sql.Stmt
	queryMsg  *sql.Stmt
}

// OpenDB attempts to open the database at dbFile.
// If no database can be found one is created.
// If dbFile is empty a fallback sequence of names is used starting with
// $XDG_DATA_HOME, then falling back to $HOME/.local/share, then falling back to
// the current working directory.
func OpenDB(ctx context.Context, appName, dbFile, schema string, debug *log.Logger) (*DB, error) {
	const (
		dbDriver   = "sqlite"
		dbFileName = "db"
	)
	var fPath string
	var paths []string

	if dbFile != "" {
		paths = []string{dbFile}
	} else {
		fPath = os.Getenv("XDG_DATA_HOME")
		if fPath != "" {
			paths = append(paths, filepath.Join(fPath, appName, dbFileName))
		}
		home, err := os.UserHomeDir()
		if err != nil {
			debug.Printf("error finding user home directory: %v", err)
		} else {
			paths = append(paths, filepath.Join(home, ".local", "share", appName, dbFileName))
		}
		fPath, err = os.Getwd()
		if err != nil {
			debug.Printf("error getting current working directory: %v", err)
		} else {
			paths = append(paths, filepath.Join(fPath, appName+".db"))
		}
	}

	// Create the path to the db file if it does not exist.
	fPath = ""
	for _, p := range paths {
		err := os.MkdirAll(filepath.Dir(p), 0770)
		if err != nil {
			debug.Printf("error creating db dir, skipping: %v", err)
			continue
		}
		// Create the database file if it does not exist, similar to touch(1).
		fd, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE, 0700)
		if err != nil {
			debug.Printf("error opening or creating db, skipping: %v", err)
			continue
		}
		err = fd.Close()
		if err != nil {
			debug.Printf("error closing db file: %v", err)
		}
		fPath = p
		break
	}
	if fPath == "" {
		return nil, errors.New("could not create or open database for writing!")
	}

	db, err := sql.Open(dbDriver, fPath)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		db.Close()
		return nil, err
	}
	_, err = db.Exec(schema)
	if err != nil {
		db.Close()
		return nil, err
	}
	return prepareQueries(ctx, db)
}

func prepareQueries(ctx context.Context, db *sql.DB) (*DB, error) {
	var err error
	wrapDB := &DB{DB: db}
	wrapDB.insertSID, err = db.PrepareContext(ctx, `
INSERT INTO sids
	(message, sid, byAttr)
	VALUES (?, ?, ?)`)
	if err != nil {
		return nil, err
	}
	wrapDB.insertMsg, err = db.PrepareContext(ctx, `
INSERT INTO messages
	(sent, toAttr, fromAttr, idAttr, body, stanzaType, originID)
	VALUES (?, ?, ?, ?, ?, ?, NULLIF(?, ""))`)
	if err != nil {
		return nil, err
	}

	wrapDB.markRecvd, err = db.PrepareContext(ctx, `
UPDATE messages SET received=TRUE WHERE sent=TRUE AND (idAttr=$1 OR originID=$1)`)

	wrapDB.queryMsg, err = db.PrepareContext(ctx, `
SELECT sent, toAttr, fromAttr, idAttr, body, stanzaType
	FROM messages
	WHERE (toAttr=$1 OR fromAttr=$1)
		AND stanzaType=COALESCE(NULLIF($2, ''), stanzaType)
	ORDER BY delay ASC;
`)
	if err != nil {
		return nil, err
	}
	return wrapDB, nil
}

// execTx creates a transaction and executes f.
// If an error is returned the transaction is rolled back, otherwise it is
// committed.
func execTx(ctx context.Context, db *DB, f func(context.Context, *sql.Tx) error) (e error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if e != nil {
			/* #nosec */
			tx.Rollback()
		}
	}()
	err = f(ctx, tx)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// MarkReceived marks a message as having been received by the other side.
func (db *DB) MarkReceived(ctx context.Context, e event.Receipt) error {
	_, err := db.markRecvd.ExecContext(ctx, string(e))
	return err
}

// InsertMsg adds a message to the database.
func (db *DB) InsertMsg(ctx context.Context, msg event.ChatMessage) error {
	return execTx(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
		res, err := tx.Stmt(db.insertMsg).ExecContext(ctx, msg.Sent, msg.To.Bare().String(), msg.From.Bare().String(), msg.ID, msg.Body, msg.Type, msg.OriginID.ID)
		if err != nil {
			return err
		}
		msgRID, err := res.LastInsertId()
		if err != nil {
			return err
		}

		if len(msg.SID) > 0 {
			insertSID := tx.Stmt(db.insertSID)
			for _, sid := range msg.SID {
				res, err = insertSID.ExecContext(ctx, msgRID, sid.ID, sid.By.String())
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// MessageIter iterates over a collection of SQL rows, returning messages.
type MessageIter struct {
	err  error
	rows *sql.Rows
	cur  event.ChatMessage
}

// Next advances the iterator and returns whether the next row is ready to be
// read.
func (i *MessageIter) Next() bool {
	if i.err != nil {
		return false
	}
	next := i.rows.Next()
	if !next {
		return next
	}

	i.cur = event.ChatMessage{}
	var to, from, typ string
	i.err = i.rows.Scan(&i.cur.Sent, &to, &from, &i.cur.ID, &i.cur.Body, &typ)
	if i.err != nil {
		return false
	}
	i.cur.Type = stanza.MessageType(typ)
	var unsafeTo, unsafeFrom jid.Unsafe
	unsafeTo, i.err = jid.ParseUnsafe(to)
	if i.err != nil {
		return false
	}
	i.cur.To = unsafeTo.JID
	unsafeFrom, i.err = jid.ParseUnsafe(from)
	if i.err != nil {
		return false
	}
	i.cur.From = unsafeFrom.JID
	return true
}

// Message returns the next event.
// Sent is whether the message was sent or received.
func (i *MessageIter) Message() event.ChatMessage {
	return i.cur
}

// Err returns the error, if any, that was encountered during iteration.
// Err may be called after an explicit or implicit Close.
func (i *MessageIter) Err() error {
	if i.err != nil {
		return i.err
	}
	return i.rows.Err()
}

// Close stops iteration.
// If Next is called and returns false, Close is called automatically.
// Close is idempotent and does not affect the result of Err.
func (i *MessageIter) Close() error {
	return i.rows.Close()
}

// QueryHistory returns all rows to or from the given JID.
// Any errors encountered while querying are deferred until the iter is used.
func (db *DB) QueryHistory(ctx context.Context, j string, typ stanza.MessageType) *MessageIter {
	rows, err := db.queryMsg.QueryContext(ctx, j, string(typ))
	return &MessageIter{
		err:  err,
		rows: rows,
	}
}
