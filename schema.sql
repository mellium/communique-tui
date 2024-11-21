-- Things to note if you're not familiar with sqlite3:
--
--  - After connecting always run "PRAGMA foreign_keys = ON" or foreign key
--    constraints will not be honored.
--  - Primary keys may be NULL (and thus always need a NOT NULL constraint)
--  - Tables with primary keys that are not integers and that don't need auto
--    incrementing counters can use WITHOUT ROWID to save some space.

PRAGMA application_id = 0x636f6d6d;

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
	delay      INTEGER  NOT NULL DEFAULT (CAST(strftime('%s', 'now') AS INTEGER)),
	rosterJID  TEXT,
	archiveID  TEXT     UNIQUE,

	UNIQUE (originID, fromAttr)
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


-- Service Discovery (disco) and Entity Capabilities (caps)

CREATE TABLE IF NOT EXISTS entityCaps (
	id   INTEGER  PRIMARY KEY NOT NULL,
	hash TEXT                 NOT NULL,
	ver  TEXT                 NOT NULL,

	UNIQUE (hash, ver)
);

CREATE TABLE IF NOT EXISTS discoFeatures (
	id  INTEGER  PRIMARY KEY NOT NULL,
	var TEXT                 NOT NULL,

	UNIQUE (var)
);

CREATE TABLE IF NOT EXISTS discoIdentity (
	id   INTEGER  PRIMARY KEY NOT NULL,
	cat  TEXT                 NOT NULL,
	name TEXT                 NOT NULL,
	typ  TEXT                 NOT NULL,
	lang TEXT                 NOT NULL,

	UNIQUE (cat, name, typ, lang)
);

CREATE TABLE IF NOT EXISTS discoJID  (
	id   INTEGER  PRIMARY KEY NOT NULL,
	jid  TEXT                 NOT NULL,
	caps INTEGER              NOT NULL,

	-- We save forms a bit differently since we don't actually use them right now
	-- except in caps calculations. Instead of saving each individual form and
	-- field, just dump all the forms associated with a JID as an XML blob that
	-- can easily be parsed out into a forms list again later.
	forms TEXT,

	FOREIGN KEY (caps) REFERENCES entityCaps(id) ON DELETE CASCADE,
	UNIQUE (jid)
);

CREATE TABLE IF NOT EXISTS discoFeatureJID (
	id   INTEGER  PRIMARY KEY NOT NULL,
	jid  INTEGER              NOT NULL,
	feat INTEGER              NOT NULL,

	FOREIGN KEY (jid)  REFERENCES discoJID(id)      ON DELETE CASCADE,
	FOREIGN KEY (feat) REFERENCES discoFeatures(id) ON DELETE CASCADE,
	UNIQUE (jid, feat)
);

CREATE TABLE IF NOT EXISTS discoIdentityJID (
	id    INTEGER  PRIMARY KEY NOT NULL,
	jid   INTEGER              NOT NULL,
	ident INTEGER              NOT NULL,

	FOREIGN KEY (jid)   REFERENCES discoJID(id)      ON DELETE CASCADE,
	FOREIGN KEY (ident) REFERENCES discoIdentity(id) ON DELETE CASCADE,
	UNIQUE (jid, ident)
);
