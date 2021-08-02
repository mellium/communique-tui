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
	received   BOOLEAN  NOT NULL DEFAULT FALSE,
	delay      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	rosterJID  TEXT,

	UNIQUE (originID, fromAttr, sent)
);


-- Roster storage

CREATE TABLE IF NOT EXISTS rosterVer (
	id  BOOLEAN PRIMARY KEY DEFAULT FALSE CHECK (id = FALSE),
	ver TEXT    NOT NULL
) WITHOUT ROWID;
-- Go ahead and populate the row if it doesn't exist so that we don't run into
-- errors the first time we try to fetch it.
INSERT INTO rosterVer (ver) VALUES ("") ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS rosterJIDs (
	jid  TEXT PRIMARY KEY NOT NULL,
	name TEXT             NOT NULL DEFAULT "",
	subs TEXT             NOT NULL
) WITHOUT ROWID;


CREATE TABLE IF NOT EXISTS rosterGroups (
	id   INTEGER  PRIMARY KEY NOT NULL,
	jid  TEXT                 NOT NULL,
	name TEXT                 NOT NULL,

	FOREIGN KEY (jid) REFERENCES rosterJIDs(jid) ON DELETE CASCADE,
	UNIQUE (jid, name)
);
