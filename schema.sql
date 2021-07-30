-- Things to note if you're not familiar with sqlite3:
--
--  - After connecting always run "PRAGMA foreign_keys = ON" or foreign key
--    constraints will not be honored.
--  - Primary keys may be NULL (and thus always need a NOT NULL constraint)
--  - Tables with primary keys that are not integers and that don't need auto
--    incrementing counters can use WITHOUT ROWID to save some space.

CREATE TABLE IF NOT EXISTS sids (
	id        INTEGER PRIMARY KEY NOT NULL,
	message   INTEGER NOT NULL,
	sid       TEXT    NOT NULL,
	byAttr    TEXT    NOT NULL,

	FOREIGN KEY (message) REFERENCES messages(id) ON DELETE CASCADE,
	UNIQUE      (sid, byAttr),
	UNIQUE      (message, byAttr)
);

CREATE TABLE IF NOT EXISTS messages (
	id         INTEGER  PRIMARY KEY NOT NULL,
	sent       BOOLEAN  NOT NULL,
	toAttr     TEXT,
	fromAttr   TEXT,
	idAttr     TEXT,
	body       TEXT,
	originID   TEXT,
	stanzaType TEXT     NOT NULL DEFAULT "normal", -- RFC 6121 § 5.2.2
	delay      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

	UNIQUE (originID, fromAttr)
);
