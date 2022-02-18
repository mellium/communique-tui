// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// Package storage implements the database layer of the client.
package storage // import "mellium.im/communique/internal/storage"

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"mellium.im/communique/internal/client/event"
	"mellium.im/xmpp/disco"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// DB represents a SQL database with common pre-prepared statements.
type DB struct {
	*sql.DB
	txM               sync.Mutex
	truncateRoster    *sql.Stmt
	delRoster         *sql.Stmt
	insertRoster      *sql.Stmt
	insertGroup       *sql.Stmt
	insertRosterVer   *sql.Stmt
	selectRosterVer   *sql.Stmt
	selectRoster      *sql.Stmt
	insertMsg         *sql.Stmt
	markRecvd         *sql.Stmt
	queryMsg          *sql.Stmt
	afterID           *sql.Stmt
	beforeID          *sql.Stmt
	insertCaps        *sql.Stmt
	insertJIDCaps     *sql.Stmt
	insertIdent       *sql.Stmt
	insertIdentCaps   *sql.Stmt
	insertFeature     *sql.Stmt
	insertFeatureCaps *sql.Stmt
	checkFeature      *sql.Stmt
	debug             *log.Logger
}

// OpenDB attempts to open the database at dbFile.
// If no database can be found one is created.
// If dbFile is empty a fallback sequence of names is used starting with
// $XDG_DATA_HOME, then falling back to $HOME/.local/share, then falling back to
// the current working directory.
func OpenDB(ctx context.Context, appName, account, dbFile, schema string, debug *log.Logger) (*DB, error) {
	const (
		dbDriver = "sqlite"
	)
	var fPath string
	var paths []string
	dbFileName := account + ".db"

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
			paths = append(paths, filepath.Join(fPath, dbFileName))
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
		return nil, fmt.Errorf("error opening DB: %w", err)
	}
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("error enabling foreign keys: %w", err)
	}
	_, err = db.Exec(schema)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("error applying schema: %w", err)
	}
	return prepareQueries(ctx, db, debug)
}

func prepareQueries(ctx context.Context, db *sql.DB, debug *log.Logger) (*DB, error) {
	var err error
	wrapDB := &DB{
		DB:    db,
		debug: debug,
	}
	wrapDB.truncateRoster, err = db.PrepareContext(ctx, `
DELETE FROM rosterJIDs;
`)
	wrapDB.delRoster, err = db.PrepareContext(ctx, `
DELETE FROM rosterJIDs WHERE jid=$1`)
	if err != nil {
		return nil, err
	}
	wrapDB.insertRoster, err = db.PrepareContext(ctx, `
INSERT INTO rosterJIDs (jid, name, subs)
	VALUES ($1, $2, $3)
	ON CONFLICT(jid) DO UPDATE SET name=$2, subs=$3`)
	if err != nil {
		return nil, err
	}
	wrapDB.insertGroup, err = db.PrepareContext(ctx, `
INSERT INTO rosterGroups (jid, name)
	VALUES (?, ?)
	ON CONFLICT DO NOTHING`)
	if err != nil {
		return nil, err
	}
	wrapDB.insertRosterVer, err = db.PrepareContext(ctx, `
INSERT INTO rosterVer (id, ver)
	VALUES (FALSE, $1)
	ON CONFLICT(id) DO UPDATE SET ver=$1`)
	if err != nil {
		return nil, err
	}
	wrapDB.selectRosterVer, err = db.PrepareContext(ctx, `
SELECT ver FROM rosterVer WHERE id=0`)
	if err != nil {
		return nil, err
	}
	wrapDB.selectRoster, err = db.PrepareContext(ctx, `
SELECT jid,name,subs FROM rosterJIDs`)
	if err != nil {
		return nil, err
	}

	wrapDB.insertMsg, err = db.PrepareContext(ctx, `
INSERT INTO messages
	(sent, toAttr, fromAttr, idAttr, body, stanzaType, originID, delay, rosterJID, archiveID)
	VALUES ($1, $2, $3, $4, $5, $6, $7, IFNULL(NULLIF($8, 0), CAST(strftime('%s', 'now') AS INTEGER)), $9, $10)
	ON CONFLICT (originID, fromAttr) DO UPDATE SET archiveID=$10
	ON CONFLICT (archiveID) DO NOTHING
	RETURNING id`)
	if err != nil {
		return nil, err
	}

	wrapDB.markRecvd, err = db.PrepareContext(ctx, `
UPDATE messages SET received=TRUE WHERE sent=TRUE AND (idAttr=$1 OR originID=$1)`)

	wrapDB.queryMsg, err = db.PrepareContext(ctx, `
SELECT sent, toAttr, fromAttr, idAttr, body, stanzaType
	FROM messages
	WHERE rosterJID=$1
		AND stanzaType=COALESCE(NULLIF($2, ''), stanzaType)
	ORDER BY delay ASC`)
	if err != nil {
		return nil, err
	}
	wrapDB.afterID, err = db.PrepareContext(ctx, `
SELECT j.jid, m.archiveID, MAX(m.delay)
	FROM messages AS m
		INNER JOIN rosterJIDs AS j ON m.rosterJID=j.jid
	GROUP BY j.jid`)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	wrapDB.beforeID, err = db.PrepareContext(ctx, `
SELECT archiveID, MIN(delay) AS mindelay
	FROM messages
	WHERE rosterJID=$1
	GROUP BY delay
	HAVING mindelay NOT NULL`)
	if err != nil {
		return nil, err
	}

	wrapDB.insertCaps, err = db.PrepareContext(ctx, `
INSERT INTO entityCaps (hash, ver)
	VALUES ($1, $2)
	ON CONFLICT (hash, ver) DO NOTHING
	RETURNING (id)`)
	if err != nil {
		return nil, err
	}
	wrapDB.insertJIDCaps, err = db.PrepareContext(ctx, `
INSERT INTO discoJIDCaps (jid, caps)
	SELECT $1, entityCaps.id
		FROM entityCaps
		WHERE entityCaps.ver=$2
	ON CONFLICT (jid) DO UPDATE SET caps=excluded.caps
	RETURNING (discoJIDCaps.id)`)
	if err != nil {
		return nil, err
	}
	wrapDB.insertIdent, err = db.PrepareContext(ctx, `
INSERT INTO discoIdentity (cat, name, typ)
	VALUES ($1, $2, $3)
	ON CONFLICT (cat, name, typ) DO UPDATE SET cat=$1, name=$2, typ=$3
	RETURNING (id)`)
	if err != nil {
		return nil, err
	}
	wrapDB.insertIdentCaps, err = db.PrepareContext(ctx, `
INSERT INTO discoIdentityCaps (caps, ident)
	VALUES ($1, $2)
	ON CONFLICT (caps, ident) DO NOTHING`)
	if err != nil {
		return nil, err
	}
	wrapDB.insertFeature, err = db.PrepareContext(ctx, `
INSERT INTO discoFeatures (var)
	VALUES ($1)
	ON CONFLICT (var) DO UPDATE SET var=$1
	RETURNING (id)`)
	if err != nil {
		return nil, err
	}
	wrapDB.insertFeatureCaps, err = db.PrepareContext(ctx, `
INSERT INTO discoFeatureCaps (caps, feat)
	VALUES ($1, $2)
	ON CONFLICT(caps, feat) DO NOTHING`)
	if err != nil {
		return nil, err
	}
	wrapDB.checkFeature, err = db.PrepareContext(ctx, `
SELECT EXISTS(
	SELECT 1
		FROM discoFeatures AS f
			INNER JOIN discoFeatureCaps AS fc ON fc.feat=f.id
			INNER JOIN entityCaps       AS c  ON fc.caps=c.id
			INNER JOIN discoJIDCaps     AS jc ON jc.caps=c.id
		WHERE jc.jid=$1 AND f.var=$2)`)
	if err != nil {
		return nil, err
	}
	return wrapDB, nil
}

var errRollback = errors.New("rollback")

// execTx creates a transaction and executes f.
// If an error is returned the transaction is rolled back, otherwise it is
// committed.
func execTx(ctx context.Context, db *DB, f func(context.Context, *sql.Tx) error) (e error) {
	db.txM.Lock()
	defer db.txM.Unlock()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	var commit bool
	defer func() {
		if commit {
			return
		}
		switch e {
		case errRollback:
			e = tx.Rollback()
		case nil:
		default:
			/* #nosec */
			tx.Rollback()
		}
	}()
	err = f(ctx, tx)
	if err != nil {
		return err
	}
	commit = true
	return tx.Commit()
}

// MarkReceived marks a message as having been received by the other side.
func (db *DB) MarkReceived(ctx context.Context, e event.Receipt) error {
	return execTx(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.Stmt(db.markRecvd).ExecContext(ctx, string(e))
		return err
	})
}

// InsertMsg adds a message to the database.
func (db *DB) InsertMsg(ctx context.Context, respectDelay bool, msg event.ChatMessage, addr jid.JID) error {
	return execTx(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
		var delay int64
		// Only store the delay if it was actually set and if it was sent from a
		// source that's trusted to set the delay.
		if respectDelay && !msg.Delay.Time.IsZero() {
			delay = msg.Delay.Time.Unix()
		}
		if msg.From.Equal(jid.JID{}) {
			msg.From = addr
		}
		var rosterJID string
		if msg.Sent {
			rosterJID = msg.To.Bare().String()
		} else {
			rosterJID = msg.From.Bare().String()
		}
		var originID *string
		switch {
		case msg.OriginID.ID != "":
			originID = &msg.OriginID.ID
		case msg.ID != "":
			// We use origin ID in the database to de-dup messages. If none was set,
			// use the regular ID and just treat it like an origin ID. This probably
			// isn't safe, but XMPP made a stupid choice early on and there aren't
			// always stable and unique IDs.
			originID = &msg.ID
		}

		var domainSID *string
		for _, sid := range msg.SID {
			if sid.By.String() == addr.Bare().String() {
				domainSID = &sid.ID
				break
			}
		}

		var msgRID uint64
		err := tx.Stmt(db.insertMsg).QueryRowContext(ctx, msg.Sent, msg.To.Bare().String(), msg.From.Bare().String(), msg.ID, msg.Body, msg.Type, originID, delay, rosterJID, domainSID).Scan(&msgRID)
		switch err {
		case sql.ErrNoRows:
			return nil
		case nil:
		default:
			return err
		}

		return nil
	})
}

// ForRoster executes f for each roster entry.
func (db *DB) ForRoster(ctx context.Context, f func(event.UpdateRoster)) error {
	return execTx(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
		var ver string
		err := tx.Stmt(db.selectRosterVer).QueryRowContext(ctx).Scan(&ver)
		if err != nil {
			return err
		}
		rows, err := tx.Stmt(db.selectRoster).Query()
		if err != nil {
			return err
		}
		/* #nosec */
		defer rows.Close()
		for rows.Next() {
			e := event.UpdateRoster{
				Ver: ver,
			}
			var jidStr string
			err = rows.Scan(&jidStr, &e.Item.Name, &e.Item.Subscription)
			if err != nil {
				return err
			}
			j, err := jid.ParseUnsafe(jidStr)
			if err != nil {
				return err
			}
			e.Item.JID = j.JID
			f(e)
		}
		return rows.Err()
	})
}

// ReplaceRoster truncates the entire roster and replaces it with the provided
// items.
func (db *DB) ReplaceRoster(ctx context.Context, e event.FetchRoster) error {
	return execTx(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
		var err error
		if e.Ver != "" {
			_, err = tx.Stmt(db.insertRosterVer).ExecContext(ctx, e.Ver)
			if err != nil {
				return err
			}
		}
		var foundItems bool
		for item := range e.Items {
			if !foundItems {
				foundItems = true
				_, err := tx.Stmt(db.truncateRoster).ExecContext(ctx)
				if err != nil {
					return err
				}
			}
			bareJID := item.Item.JID.Bare().String()
			_, err = tx.Stmt(db.insertRoster).ExecContext(ctx, bareJID, item.Name, item.Subscription)
			if err != nil {
				return err
			}
			insGroup := tx.Stmt(db.insertGroup)
			for _, group := range item.Group {
				_, err = tx.Stmt(insGroup).ExecContext(ctx, bareJID, group)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// RosterVer returns the currently saved roster version.
func (db *DB) RosterVer(ctx context.Context) (string, error) {
	var ver string
	err := execTx(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
		err := tx.Stmt(db.selectRosterVer).QueryRowContext(ctx).Scan(&ver)
		return err
	})
	return ver, err
}

// UpdateRoster upserts or removes a JID from the roster.
func (db *DB) UpdateRoster(ctx context.Context, ver string, item event.UpdateRoster) error {
	if item.Subscription == "remove" {
		return execTx(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
			if ver != "" {
				_, err := tx.Stmt(db.insertRosterVer).ExecContext(ctx, ver)
				if err != nil {
					return err
				}
			}

			_, err := tx.Stmt(db.delRoster).ExecContext(ctx, item.JID.Bare().String())
			return err
		})
	}

	return execTx(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
		if ver != "" {
			_, err := tx.Stmt(db.insertRosterVer).ExecContext(ctx, ver)
			if err != nil {
				return err
			}
		}
		bareJID := item.JID.Bare().String()
		_, err := tx.Stmt(db.insertRoster).ExecContext(ctx, bareJID, item.Name, item.Subscription)
		if err != nil {
			return err
		}
		insGroup := tx.Stmt(db.insertGroup)
		for _, group := range item.Group {
			_, err = insGroup.ExecContext(ctx, bareJID, group)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// MessageIter is an iterator that can return concrete messages.
type MessageIter struct {
	*Iter
}

// Result returns the most recent result read from the iter.
func (iter MessageIter) Message() event.ChatMessage {
	cur := iter.Iter.Current()
	if cur == nil {
		return event.ChatMessage{}
	}
	return cur.(event.ChatMessage)
}

// QueryHistory returns all rows to or from the given JID.
// Any errors encountered while querying are deferred until the iter is used.
func (db *DB) QueryHistory(ctx context.Context, j string, typ stanza.MessageType) MessageIter {
	db.txM.Lock()
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-ctx.Done()
		defer db.txM.Unlock()
	}()

	rows, err := db.queryMsg.QueryContext(ctx, j, string(typ))
	return MessageIter{
		Iter: &Iter{
			cancel: cancel,
			err:    err,
			rows:   rows,
			f: func(rows *sql.Rows) (interface{}, error) {
				cur := event.ChatMessage{}
				var to, from, typ string
				err := rows.Scan(&cur.Sent, &to, &from, &cur.ID, &cur.Body, &typ)
				if err != nil {
					return cur, err
				}
				cur.Type = stanza.MessageType(typ)
				unsafeTo, err := jid.ParseUnsafe(to)
				if err != nil {
					return cur, err
				}
				cur.To = unsafeTo.JID
				unsafeFrom, err := jid.ParseUnsafe(from)
				if err != nil {
					return cur, err
				}
				cur.From = unsafeFrom.JID
				return cur, nil
			},
		},
	}
}

// AfterIDRes is returned from an AfterID query.
type AfterIDResult struct {
	Addr  jid.JID
	ID    string
	Delay time.Time
}

// AfterIDIter is an iterator that can return concrete AfterIDRes values.
type AfterIDIter struct {
	*Iter
}

// Result returns the most recent result read from the iter.
func (iter AfterIDIter) Result() AfterIDResult {
	cur := iter.Iter.Current()
	if cur == nil {
		return AfterIDResult{}
	}
	return cur.(AfterIDResult)
}

// AfterID gets the last known message ID assigned by the 'by' JID for each
// roster entry.
func (db *DB) AfterID(ctx context.Context) AfterIDIter {
	db.txM.Lock()
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-ctx.Done()
		defer db.txM.Unlock()
	}()

	rows, err := db.afterID.QueryContext(ctx)
	return AfterIDIter{
		Iter: &Iter{
			cancel: cancel,
			err:    err,
			rows:   rows,
			f: func(rows *sql.Rows) (interface{}, error) {
				cur := AfterIDResult{}
				var j string
				var delay int64
				var id *string
				err := rows.Scan(&j, &id, &delay)
				if err != nil {
					return cur, err
				}
				if id != nil {
					cur.ID = *id
				}
				cur.Delay = time.Unix(delay, 0)
				var unsafeJID jid.Unsafe
				unsafeJID, err = jid.ParseUnsafe(j)
				cur.Addr = unsafeJID.JID
				return cur, err
			},
		},
	}
}

// BeforeID gets the first known message ID and timestamp for the given JID.
func (db *DB) BeforeID(ctx context.Context, j jid.JID) (string, time.Time, error) {
	var id string
	var timestamp int64
	err := execTx(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
		return tx.Stmt(db.beforeID).QueryRowContext(ctx, j.String()).Scan(&id, &timestamp)
	})
	if err == sql.ErrNoRows {
		err = nil
	}
	var t time.Time
	if timestamp != 0 {
		t = time.Unix(timestamp, 0)
	}
	return id, t, err
}

// CheckFeature checks if the given JID has advertised a feature.
// It does not distinguish between no features having been received at all and
// the specific feature not being advertised.
func (db *DB) CheckFeature(ctx context.Context, j jid.JID, v string) (bool, error) {
	var res bool
	err := db.checkFeature.QueryRowContext(ctx, j.String(), v).Scan(&res)
	return res, err
}

// UpdateDisco checks if the entity capabilities have previously been seen and
// if not stores them and calls f to fetch and store new disco features.
func (db *DB) UpdateDisco(ctx context.Context, j jid.JID, caps disco.Caps, f func(ctx context.Context) (disco.Info, error)) error {
	return execTx(ctx, db, func(ctx context.Context, tx *sql.Tx) error {
		var rowID int64
		err := tx.Stmt(db.insertCaps).QueryRowContext(ctx, strings.ToLower(caps.Hash.String()), caps.Ver).Scan(&rowID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		_, err = tx.Stmt(db.insertJIDCaps).ExecContext(ctx, j.String(), caps.Ver)
		if err != nil {
			return err
		}
		if rowID == 0 {
			// Cache hit, no need to update anything else!
			return nil
		}

		info, err := f(ctx)
		if err != nil {
			return err
		}
		insertIdent := tx.Stmt(db.insertIdent)
		insertIdentCaps := tx.Stmt(db.insertIdentCaps)
		insertFeatures := tx.Stmt(db.insertFeature)
		insertFeatureCaps := tx.Stmt(db.insertFeatureCaps)
		for _, ident := range info.Identity {
			var identID int64
			err = insertIdent.QueryRowContext(ctx, ident.Category, ident.Name, ident.Type).Scan(&identID)
			if err != nil {
				return err
			}
			if identID != 0 {
				_, err = insertIdentCaps.ExecContext(ctx, rowID, identID)
				if err != nil {
					return err
				}
			}
		}
		for _, feat := range info.Features {
			var featID int64
			err = insertFeatures.QueryRowContext(ctx, feat.Var).Scan(&featID)
			if err != nil {
				return err
			}
			if featID != 0 {
				_, err = insertFeatureCaps.ExecContext(ctx, rowID, featID)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
}
