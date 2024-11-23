// Code generated by running "go generate" in golang.org/x/text. DO NOT EDIT.

package main

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/message/catalog"
)

type dictionary struct {
	index []uint32
	data  string
}

func (d *dictionary) Lookup(key string) (data string, ok bool) {
	p, ok := messageKeyToIndex[key]
	if !ok {
		return "", false
	}
	start, end := d.index[p], d.index[p+1]
	if start == end {
		return "", false
	}
	return d.data[start:end], true
}

func init() {
	dict := map[string]catalog.Dictionary{
		"de":    &dictionary{index: deIndex, data: deData},
		"en_US": &dictionary{index: en_USIndex, data: en_USData},
	}
	fallback := language.MustParse("en-US")
	cat, err := catalog.NewFromMap(dict, catalog.Fallback(fallback))
	if err != nil {
		panic(err)
	}
	message.DefaultCatalog = cat
}

var messageKeyToIndex = map[string]int{
	"# This is a config file for Communiqué.\n# If the -f option is not provided, Communiqué will search for a config file in:\n#\n#   - ./communiqué.toml\n#   - $XDG_CONFIG_HOME/communiqué/config.toml\n#   - $HOME/.config/communiqué/config.toml\n#   - /etc/communiqué/config.toml\n#\n# The only required field is \"address\". The \"password_eval\" field should be set\n# to a command that writes the password to standard out. Normally this should\n# decrypt an encrypted file containing the password. If it is not specified, the\n# user will be prompted to enter a password.\n": 129,
	"%v\n\nTry running '%s -config' to generate a default config file.": 33,
	"Add":                              118,
	"Add Contact":                      98,
	"Address":                          99,
	"Are you sure you want to quit?":   114,
	"Away":                             112,
	"Away %s":                          184,
	"Busy":                             113,
	"Busy %s":                          185,
	"Cancel":                           21,
	"Channels":                         101,
	"Chat: %q (%s)":                    179,
	"Commands":                         119,
	"Complete":                         20,
	"Conversation":                     102,
	"Conversations":                    105,
	"DEBUG":                            25,
	"Data Form":                        190,
	"Enter password for: %q":           177,
	"Error closing roster stream: %q":  96,
	"Error closing the connection: %q": 86,
	"Error encoding default config as TOML: %v": 32,
	"Error going offline: %q":                   85,
	"Error while handling XMPP streams: %q":     84,
	"Exec":                                      121,
	"Join":                                      116,
	"Join Channel":                              117,
	"Loading commands…":                         120,
	"Login":                                     176,
	"Logs":                                      106,
	"Name":                                      100,
	"Next":                                      19,
	"No commands found for %v!":                 200,
	"Offline":                                   110,
	"Offline %s":                                186,
	"Online":                                    111,
	"Online %s":                                 183,
	"Password":                                  175,
	"Pick Address":                              202,
	"Prev":                                      18,
	"Quit":                                      115,
	"RECV":                                      26,
	"Remove":                                    107,
	"Remove this channel?":                      109,
	"Remove this contact from your roster?":     108,
	"Roster":                                    178,
	"Roster info:":                              203,
	"SENT":                                      27,
	"Search":                                    181,
	"Select":                                    201,
	"Set Status":                                182,
	"Status: %s":                                180,
	"To fix this, contact your server administrator and ask them to enable %q": 93,
	"Usage of communiqué:\n\n": 24,
	"Your server does not support bookmark unification, an important feature that stops newer clients from seeing a different list of chat rooms than older clients that do not yet support the latest features.": 94,
	"account %q not found in config file":                                         36,
	"bad hash type found in database: %v":                                         150,
	"cannot upload directory":                                                     137,
	"caps cache hit for %s: %s:%s":                                                15,
	"caps cache miss for %s: %s:%s, %[2]s:%[4]s":                                  16,
	"could not create or open database for writing":                               145,
	"could not get the upload services: %v":                                       73,
	"could not upload %q: %v":                                                     75,
	"encoding form failed: %v":                                                    163,
	"encoding root forms end element failed: %v":                                  164,
	"encoding root forms start element failed: %v":                                162,
	"error adding roster item %s: %v":                                             61,
	"error applying schema: %v":                                                   148,
	"error bootstraping history for %s: %v":                                       3,
	"error canceling command session: %v":                                         125,
	"error closing bookmarks stream: %v":                                          95,
	"error closing cancel command payload: %v":                                    126,
	"error closing command session: %v":                                           127,
	"error closing commands iter for %q: %v":                                      56,
	"error closing config file: %v":                                               35,
	"error closing db file: %v":                                                   144,
	"error closing feature rows: %v":                                              158,
	"error closing file: %v":                                                      136,
	"error closing identity rows: %v":                                             154,
	"error copying early log data to output buffer: %q":                           40,
	"error creating db dir, skipping: %v":                                         142,
	"error creating keylog file: %q":                                              47,
	"error decoding forms: %v":                                                    160,
	"error dialing connection: %v":                                                132,
	"error discovering bookmarks support: %v":                                     63,
	"error enabling carbons: %q":                                                  134,
	"error enabling foreign keys: %v":                                             147,
	"error executing command %q on %q: %v":                                        53,
	"error executing info template: %v":                                           206,
	"error fetching bookmarks: %q":                                                91,
	"error fetching commands for %q: %v":                                          55,
	"error fetching earliest message info for %v from database: %v":               80,
	"error fetching history after %s for %s: %v":                                  2,
	"error fetching info from cache: %v":                                          13,
	"error fetching roster: %q":                                                   90,
	"error fetching scrollback for %v: %v":                                        83,
	"error fetching version information: %v":                                      92,
	"error finding user home directory: %v":                                       140,
	"error getting caps: %v":                                                      149,
	"error getting current working directory: %v":                                 141,
	"error getting features: %v":                                                  155,
	"error getting identities: %v":                                                151,
	"error getting services: %v":                                                  171,
	"error going offline: %v":                                                     60,
	"error inserting JIDCapsForm: %v":                                             166,
	"error inserting caps: %v":                                                    161,
	"error inserting entity capbailities hash: %v":                                11,
	"error inserting feature %s: %v":                                              169,
	"error inserting feature caps joiner: %v":                                     170,
	"error inserting identity %v: %v":                                             167,
	"error inserting identity caps joiner: %v":                                    168,
	"error iterating over feature rows: %v":                                       157,
	"error iterating over identity rows: %v":                                      153,
	"error iterating over roster items: %v":                                       4,
	"error iterating over services rows: %v":                                      174,
	"error joining room %s: %v":                                                   79,
	"error loading chat: %v":                                                      76,
	"error loading scrollback into pane for %v: %v":                               131,
	"error logging to pane: %v":                                                   39,
	"error marking message %q as received: %v":                                    6,
	"error negotiating session: %v":                                               133,
	"error occured during service discovery: %v":                                  87,
	"error opening DB: %v":                                                        146,
	"error opening database: %v":                                                  38,
	"error opening or creating db, skipping: %v":                                  143,
	"error parsing config file: %v":                                               34,
	"error parsing jid-multi value for field %s: %v":                              192,
	"error parsing main account as XMPP address: %v":                              37,
	"error parsing timeout, defaulting to 30s: %q":                                46,
	"error parsing user address: %q":                                              45,
	"error publishing bookmark %s: %v":                                            65,
	"error publishing legacy bookmark %s: %v":                                     64,
	"error querying database for last seen messages: %v":                          1,
	"error querying for disco forms: %v":                                          159,
	"error querying history for %s: %v":                                           23,
	"error removing bookmark %s: %v":                                              67,
	"error removing legacy bookmark %s: %v":                                       66,
	"error removing roster item %s: %v":                                           62,
	"error retrieving roster version, falling back to full roster fetch: %v":      48,
	"error running password command, falling back to prompt: %v":                  42,
	"error saving entity caps to the database: %v":                                17,
	"error saving sent message to history: %v":                                    72,
	"error scanning feature row: %v":                                              156,
	"error scanning identity: %v":                                                 152,
	"error scanning services: %v":                                                 172,
	"error sending message: %v":                                                   71,
	"error sending presence pre-approval to %s: %v":                               68,
	"error sending presence request to %s: %v":                                    69,
	"error setting away status: %v":                                               57,
	"error setting bool form field %s: %v":                                        191,
	"error setting busy status: %v":                                               59,
	"error setting deadline: %v":                                                  135,
	"error setting jid form field %s: %v":                                         194,
	"error setting jid-multi form field %s: %v":                                   193,
	"error setting list form field %s: %v":                                        196,
	"error setting list-multi form field %s: %v":                                  195,
	"error setting online status: %v":                                             58,
	"error setting password form field %s: %v":                                    199,
	"error setting text form field %s: %v":                                        197,
	"error setting text-multi form field %s: %v":                                  198,
	"error showing next command for %q: %v":                                       54,
	"error showing next command: %v":                                              128,
	"error updating roster version: %v":                                           5,
	"error updating to roster ver %q: %v":                                         0,
	"error when closing response body: %v":                                        138,
	"error when closing the items iterator: %v":                                   88,
	"error while picking files: %v":                                               103,
	"error writing history message to chat: %v":                                   9,
	"error writing history to database: %v":                                       10,
	"error writing history: %v":                                                   22,
	"error writing message to database: %v":                                       8,
	"error writing received message to chat: %v":                                  7,
	"executing command: %+v":                                                      52,
	"failed to get the recipient":                                                 104,
	"failed to parse service JID: %v":                                             173,
	"failed to read process' standard error: %v":                                  188,
	"failed to read process' standard output: %v":                                 189,
	"failed to read stderr of the notification subprocess: %v":                    122,
	"failed to run notification command: %v":                                      123,
	"falling back to network query…":                                              14,
	"feature discovery failed for %q: %v":                                         89,
	"fetching scrollback before %v for %v…":                                       82,
	"flushing encoded form failed: %v":                                            165,
	"got signal: %v":                                                              51,
	"initial login failed: %v":                                                    49,
	"invalid nick %s in config: %v":                                               77,
	"joining room %v…":                                                            78,
	"logged in as: %q":                                                            50,
	"no file picker set, see the example configuration file for more information": 187,
	"no scrollback for %v":                                                        81,
	"no sidebar open, not showing info pane…":                                     204,
	"no upload service available":                                                 74,
	"no user address specified, edit %q and add:\n\n\tjid=\"me@example.com\"\n\n": 43,
	"notification subprocess failed: %v\n%s":                                      124,
	"override the account set in the config file":                                 29,
	"possibly spoofed history message from %s":                                    97,
	"print a default config file to stdout":                                       31,
	"print this help message":                                                     30,
	"running command: %q":                                                         41,
	"the config file to load":                                                     28,
	"unexpected status code: %d (%s)":                                             139,
	"unrecognized client event: %T(%[1]q)":                                        12,
	"unrecognized sidebar item type %T, not showing info…":                        205,
	"unrecognized ui event: %T(%[1]q)":                                            70,
	"uploaded %q as %s":                                                           130,
	"user address: %q":                                                            44,
}

var deIndex = []uint32{ // 208 elements
	// Entry 0 - 1F
	0x00000000, 0x0000003a, 0x00000081, 0x000000bd,
	0x000000f4, 0x00000127, 0x0000015c, 0x00000199,
	0x000001d0, 0x000001fb, 0x0000022f, 0x0000026a,
	0x000002a3, 0x000002cf, 0x00000308, 0x00000329,
	0x00000354, 0x0000038d, 0x000003cb, 0x000003d4,
	0x000003de, 0x000003eb, 0x000003f5, 0x0000041f,
	0x00000453, 0x00000475, 0x0000047b, 0x00000485,
	0x0000048e, 0x000004b1, 0x000004da, 0x000004f7,
	// Entry 20 - 3F
	0x00000531, 0x00000570, 0x000005c8, 0x000005fa,
	0x00000630, 0x00000672, 0x000006b0, 0x000006d9,
	0x00000706, 0x0000074e, 0x00000767, 0x000007b9,
	0x00000822, 0x0000083a, 0x00000869, 0x000008b6,
	0x000008e4, 0x00000940, 0x00000969, 0x0000097f,
	0x00000996, 0x000009b4, 0x000009ed, 0x00000a2a,
	0x00000a5d, 0x00000a9b, 0x00000acb, 0x00000af9,
	0x00000b2e, 0x00000b4e, 0x00000b87, 0x00000bbe,
	// Entry 40 - 5F
	0x00000bfb, 0x00000c3c, 0x00000c74, 0x00000cae,
	0x00000ce1, 0x00000d20, 0x00000d58, 0x00000d91,
	0x00000db9, 0x00000dfe, 0x00000e30, 0x00000e4e,
	0x00000e7b, 0x00000e9e, 0x00000ed6, 0x00000ef2,
	0x00000f18, 0x00000f79, 0x00000fa0, 0x00000fcb,
	0x00000ffb, 0x0000102b, 0x0000104b, 0x00001078,
	0x000010a6, 0x000010db, 0x0000110f, 0x00001133,
	0x0000115c, 0x0000118f, 0x0000120c, 0x000012d1,
	// Entry 60 - 7F
	0x00001307, 0x00001338, 0x0000136d, 0x00001381,
	0x0000138a, 0x0000138f, 0x00001397, 0x000013a4,
	0x000013cd, 0x000013f3, 0x00001402, 0x0000140d,
	0x00001417, 0x00001435, 0x00001446, 0x0000144e,
	0x00001455, 0x0000145e, 0x0000146c, 0x0000149b,
	0x000014a3, 0x000014ad, 0x000014bd, 0x000014c9,
	0x000014d1, 0x000014e1, 0x000014e6, 0x0000153e,
	0x00001579, 0x000015b0, 0x000015b0, 0x000015b0,
	// Entry 80 - 9F
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	// Entry A0 - BF
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	// Entry C0 - DF
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
	0x000015b0, 0x000015b0, 0x000015b0, 0x000015b0,
} // Size: 856 bytes

const deData string = "" + // Size: 5552 bytes
	"\x02Error beim Aktuellisieren von roster Version %[1]q: %[2]v\x02Fehler " +
	"bei einer Datenbankabfrage über die letzten Nachrichten: %[1]v\x02Fehler" +
	" beim laden des Verlaufs nach %[1]s für %[2]s: %[3]v\x02Error beim Boots" +
	"trappen des Verlaufs für %[1]s: %[2]v\x02Fehler beim iterieren über rost" +
	"er-Elemente: %[1]v\x02Fehler beim aktuellisieren der roster-Version: %[1" +
	"]v\x02Fehler beim erstellen der Lesebestätigung für %[1]q: %[2]v\x02Fehl" +
	"er beim Schreiben der Nachricht in den Chat: %[1]v\x02Fehler beim Schrei" +
	"ben der Datenbank: %[1]v\x02Fehler beim Schreiben des Nachrchtenverlaufs" +
	": %[1]v\x02Fehler beim Schreiben des Verlaufs in die Datenbank: %[1]v" +
	"\x02Fehler beim Einfügen des Entity-Capability-Hashs: %[1]v\x02Unbekannt" +
	"es Client-Ereigniss: %[1]T (%[1]q)\x02Fehler beim Laden von Informatione" +
	"n auf dem Cache: %[1]v\x02Rückfall zu Netzwerkanfrage.…\x02Caps-Cache tr" +
	"effer für %[1]s: %[2]s:%[3]s\x02Caps-Cache verfehlt für %[1]s: %[2]s:%[3" +
	"]s, %[2]s:%[4]s\x02Fehler beim Speichern der Entity-Caps in die Datenban" +
	"k: %[1]v\x02Vorherig\x02Nächstes\x02Vollständig\x02Abbrechen\x02Fehler b" +
	"eim Speichern des Verlaufs: %[1]v\x02Fehler beim Abfragen des Verlaufs f" +
	"ür %[1]s: %[2]v\x04\x00\x02\x0a\x0a\x1c\x02Verwendung von communiqué:" +
	"\x02DEBUG\x02EMPFANGEN\x02GESENDET\x02Die zu ladente Konfigurationsdatei" +
	"\x02den eingestellten Account überschreiben\x02Druckt diese Hilfe-Nachri" +
	"cht\x02Gibt die Standartkonfiguration in der Standartausgabe aus\x02Fehl" +
	"er beim Kodieren der Standartkonfiguration als TOML: %[1]v\x02%[1]v\x0a" +
	"\x0aVersuche '%[2]s -config' auszuführen, um die Standartkonfiguration z" +
	"u erzeugen.\x02Fehler beim parsen der Konfigurationsdatei: %[1]v\x02Fehl" +
	"er beim Schließen der Konfigurationsdatei: %[1]v\x02Der Account %[1]q wu" +
	"rde nicht in der Konfigurationsdatei gefunden\x02Fehler beim parsen der " +
	"XMPP-Addresse des Hauptaccounts: %[1]v\x02Fehler beim Öffnen der Datenba" +
	"nk: %[1]v\x02Fehler beim Protokollieren zum Paneel: %[1]v\x02Fehler beim" +
	" Kopieren früher Protokolldaten in den Ausgabepuffer: %[1]q\x02Führt Bef" +
	"ehl aus: %[1]q\x02Fehler beim Ausführen des Passwort-Befehls; Rückfall z" +
	"ur Passwortabfrage: %[1]v\x04\x00\x02\x0a\x0ac\x02Keine Benutzeraddresse" +
	" festgelegt, bearbeite %[1]q und füge folgendes ein:\x0a\x0a\x09jid=\x22" +
	"me@example.com\x22\x02Benutzeraddresse: %[1]q\x02Fehler beim Parsen der " +
	"Benutzeraddresse: %[1]q\x02Fehler beim Parsen des Zeitlimits, rückfall z" +
	"um Standartwert von 30s: %[1]q\x02Fehler beim Erstellen der Keylog-Datei" +
	": %[1]q\x02Fehler beim Feststellen der roster-Version, rückfall zu einer" +
	" vollen roster-Abfrage: %[1]v\x02Initiale Anmeldung fehlgeschlagen: %[1]" +
	"v\x02Angemeldet als: %[1]q\x02Signal bekommen: %[1]v\x02ausführen von Be" +
	"fehl: %+[1]v\x02Fehler beim Ausführen vom Befehl %[1]q auf %[2]q: %[3]v" +
	"\x02Fehler beim anzeigen des nächsten Befehls für %[1]q: %[2]v\x02Fehler" +
	" beim Abfragen der Befehle für %[1]q: %[2]v\x02Fehler beim Schließen des" +
	" Befehlsiterators für %[1]q: %[2]v\x02Fehler beim Setzen des Abwesent-St" +
	"atuses: %[1]v\x02Fehler beim Setzen des Online-Statuses: %[1]v\x02Fehler" +
	" beim Setzen des Beschäfftigt-Statuses: %[1]v\x02Fehler beim Offlinegehe" +
	"n: %[1]v\x02Fehler beim Hinzufügen des roster-Elements %[1]s: %[2]v\x02F" +
	"ehler beim Entfernen des roster-Elements %[1]s: %[2]v\x02Fehler beim Fes" +
	"tstellen von Lesezeichenunterstützung: %[1]v\x02Fehler beim Veröffentlic" +
	"hen von Legacy-Lesezeichen %[1]s: %[2]v\x02Fehler beim Veröffentlich von" +
	" Lesezeichen %[1]s: %[2]v\x02Error beim Entfernen des Legacy-Lesezeichen" +
	"s %[1]s: %[2]v\x02Fehler beim Entfernen von Lesezeichen %[1]s: %[2]v\x02" +
	"Fehler beim Senden der Präsenzvorbestätigung to %[1]s: %[2]v\x02Error be" +
	"im Senden einer Präsenzanfrage an %[1]s: %[2]v\x02Unbekanntes Benutzerob" +
	"erflächenereigniss: %[1]T (%[1]q)\x02Fehler beim Senden der Nachricht: %" +
	"[1]v\x02Fehler beim Speichern der gesendeten Nachricht in den Verlauf: %" +
	"[1]v\x02Upload-Dienst konnte nicht gefunden werden: %[1]v\x02Kein Upload" +
	"-Dienst verfügbar\x02%[1]q konnte nicht Hochgeladen werden: %[2]v\x02Feh" +
	"ler beim Laden des Chats: %[1]v\x02Ungültiger Spitzname %[1]s in der Kon" +
	"figuration: %[2]v\x02Tritt dem Raum %[1]v bei…\x02Fehler beim Beitreten " +
	"zu %[1]s: %[2]v\x02Fehler beim Laden der Informationen für die erste Nac" +
	"hricht für %[1]v aus der Datenbank: %[2]v\x02Kein Chatverlauf verfügbar " +
	"für %[1]v\x02Laden des Verlaufs vor %[1]v für %[2]v…\x02Fehler beim lade" +
	"n des Verlauf für %[1]v: %[2]v\x02Fehler beim Verarbeiten des XMPP-Strea" +
	"ms: %[1]q\x02Fehler beim Offlinegehen: %[1]q\x02Fehler beim Schließen de" +
	"r Verbindung: %[1]q\x02Fehler während der Dienste-Entdeckung: %[1]v\x02F" +
	"ehler beim Schließen des Elemente-Iterators: %[1]v\x02Funktionsernennung" +
	" fehlgeschlagen für %[1]q: %[2]v\x02Fehler beim Laden von roster: %[1]q" +
	"\x02Fehler beim Laden der Lesezeichen: %[1]q\x02Fehler beim Laden von Ve" +
	"rsionsinformationen: %[1]v\x02Um dieses Problem zu lösen, kontaktiere bi" +
	"tte den Administrator des Servers und frage nach, ob %[1]q aktiviert wer" +
	"den kann.\x02Dein Server unterstützt Lesezeichenunifikation nicht. Diese" +
	" Funktion verhindert, dass neuere Clients eine andere Chatliste sehen, a" +
	"ls Ältere, die noch nicht die neusten Funktionen unterstützen.\x02Fehler" +
	" beim Schließen des Lesezeichen-Streams: %[1]v\x02Fehler beim Schließen " +
	"des roster-Streams: %[1]q\x02potentiell gefälschter Nachrichtenverlauf v" +
	"on %[1]s\x02Kontakt hinzufügen\x02Addresse\x02Name\x02Kanäle\x02Konversa" +
	"tion\x02Fehler beim Auswählen von Datein: %[1]v\x02Empfänger kann nicht " +
	"gefunden werden\x02Konversationen\x02Protokolle\x02Entfernen\x02Kontakt " +
	"vom Roster entfernen?\x02Kanal entfernen?\x02Offline\x02Online\x02Abwese" +
	"nd\x02Beschäfftigt\x02Bist du dir sicher, dass du Schließen willst?\x02B" +
	"eenden\x02Beitreten\x02Kanal beitreten\x02Hinzufügen\x02Befehle\x02Lade " +
	"Befehle…\x02Exec\x02Standartfehlerausgabe vom Benachrichtigungsprozesses" +
	" konnte nicht gelesen werden: %[1]v\x02Fehler beim Ausführen des Benachr" +
	"ichtigungsbefehls: %[1]v\x02Benachrichtigungsunterprozess abgestürzt: %[" +
	"1]v\x0a%[2]s"

var en_USIndex = []uint32{ // 208 elements
	// Entry 0 - 1F
	0x00000000, 0x0000002a, 0x00000060, 0x00000094,
	0x000000c0, 0x000000e9, 0x0000010e, 0x0000013d,
	0x0000016b, 0x00000194, 0x000001c1, 0x000001ea,
	0x0000021a, 0x00000242, 0x00000268, 0x00000289,
	0x000002af, 0x000002e3, 0x00000313, 0x00000318,
	0x0000031d, 0x00000326, 0x0000032d, 0x0000034a,
	0x00000372, 0x0000038e, 0x00000394, 0x00000399,
	0x0000039e, 0x000003b6, 0x000003e2, 0x000003fa,
	// Entry 20 - 3F
	0x00000420, 0x0000044d, 0x00000493, 0x000004b4,
	0x000004d5, 0x000004fc, 0x0000052e, 0x0000054c,
	0x00000569, 0x0000059e, 0x000005b5, 0x000005f3,
	0x0000063f, 0x00000653, 0x00000675, 0x000006a5,
	0x000006c7, 0x00000711, 0x0000072d, 0x00000741,
	0x00000753, 0x0000076d, 0x0000079b, 0x000007c7,
	0x000007f0, 0x0000081d, 0x0000083e, 0x00000861,
	0x00000882, 0x0000089d, 0x000008c3, 0x000008eb,
	// Entry 40 - 5F
	0x00000916, 0x00000944, 0x0000096b, 0x00000997,
	0x000009bc, 0x000009f0, 0x00000a1f, 0x00000a43,
	0x00000a60, 0x00000a8c, 0x00000ab5, 0x00000ad1,
	0x00000aef, 0x00000b09, 0x00000b2d, 0x00000b43,
	0x00000b63, 0x00000ba7, 0x00000bbf, 0x00000bed,
	0x00000c18, 0x00000c41, 0x00000c5c, 0x00000c80,
	0x00000cae, 0x00000cdb, 0x00000d05, 0x00000d22,
	0x00000d42, 0x00000d6c, 0x00000db8, 0x00000e84,
	// Entry 60 - 7F
	0x00000eaa, 0x00000ecd, 0x00000ef9, 0x00000f05,
	0x00000f0d, 0x00000f12, 0x00000f1b, 0x00000f28,
	0x00000f49, 0x00000f65, 0x00000f73, 0x00000f78,
	0x00000f7f, 0x00000fa5, 0x00000fba, 0x00000fc2,
	0x00000fc9, 0x00000fce, 0x00000fd3, 0x00000ff2,
	0x00000ff7, 0x00000ffc, 0x00001009, 0x0000100d,
	0x00001016, 0x0000102a, 0x0000102f, 0x0000106b,
	0x00001095, 0x000010c1, 0x000010e8, 0x00001114,
	// Entry 80 - 9F
	0x00001139, 0x0000115b, 0x00001393, 0x000013ab,
	0x000013df, 0x000013ff, 0x00001420, 0x0000143e,
	0x0000145c, 0x00001476, 0x0000148e, 0x000014b6,
	0x000014dc, 0x00001505, 0x00001534, 0x0000155b,
	0x00001589, 0x000015a6, 0x000015d4, 0x000015ec,
	0x0000160f, 0x0000162c, 0x00001646, 0x0000166d,
	0x0000168d, 0x000016ac, 0x000016d6, 0x000016f9,
	0x00001717, 0x00001739, 0x00001762, 0x00001784,
	// Entry A0 - BF
	0x000017aa, 0x000017c6, 0x000017e2, 0x00001812,
	0x0000182e, 0x0000185c, 0x00001880, 0x000018a3,
	0x000018c9, 0x000018f5, 0x0000191a, 0x00001945,
	0x00001963, 0x00001982, 0x000019a5, 0x000019cf,
	0x000019d8, 0x000019de, 0x000019f8, 0x000019ff,
	0x00001a13, 0x00001a21, 0x00001a28, 0x00001a33,
	0x00001a40, 0x00001a4b, 0x00001a56, 0x00001a64,
	0x00001ab0, 0x00001ade, 0x00001b0d, 0x00001b17,
	// Entry C0 - DF
	0x00001b42, 0x00001b77, 0x00001ba7, 0x00001bd1,
	0x00001c02, 0x00001c2d, 0x00001c58, 0x00001c89,
	0x00001cb8, 0x00001cd5, 0x00001cdc, 0x00001ce9,
	0x00001cf6, 0x00001d20, 0x00001d5a, 0x00001d7f,
} // Size: 856 bytes

const en_USData string = "" + // Size: 7551 bytes
	"\x02error updating to roster ver %[1]q: %[2]v\x02error querying database" +
	" for last seen messages: %[1]v\x02error fetching history after %[1]s for" +
	" %[2]s: %[3]v\x02error bootstraping history for %[1]s: %[2]v\x02error it" +
	"erating over roster items: %[1]v\x02error updating roster version: %[1]v" +
	"\x02error marking message %[1]q as received: %[2]v\x02error writing rece" +
	"ived message to chat: %[1]v\x02error writing message to database: %[1]v" +
	"\x02error writing history message to chat: %[1]v\x02error writing histor" +
	"y to database: %[1]v\x02error inserting entity capbailities hash: %[1]v" +
	"\x02unrecognized client event: %[1]T(%[1]q)\x02error fetching info from " +
	"cache: %[1]v\x02falling back to network query…\x02caps cache hit for %[1" +
	"]s: %[2]s:%[3]s\x02caps cache miss for %[1]s: %[2]s:%[3]s, %[2]s:%[4]s" +
	"\x02error saving entity caps to the database: %[1]v\x02Prev\x02Next\x02C" +
	"omplete\x02Cancel\x02error writing history: %[1]v\x02error querying hist" +
	"ory for %[1]s: %[2]v\x04\x00\x02\x0a\x0a\x16\x02Usage of communiqué:\x02" +
	"DEBUG\x02RECV\x02SENT\x02the config file to load\x02override the account" +
	" set in the config file\x02print this help message\x02print a default co" +
	"nfig file to stdout\x02Error encoding default config as TOML: %[1]v\x02%" +
	"[1]v\x0a\x0aTry running '%[2]s -config' to generate a default config fil" +
	"e.\x02error parsing config file: %[1]v\x02error closing config file: %[1" +
	"]v\x02account %[1]q not found in config file\x02error parsing main accou" +
	"nt as XMPP address: %[1]v\x02error opening database: %[1]v\x02error logg" +
	"ing to pane: %[1]v\x02error copying early log data to output buffer: %[1" +
	"]q\x02running command: %[1]q\x02error running password command, falling " +
	"back to prompt: %[1]v\x04\x00\x02\x0a\x0aF\x02no user address specified," +
	" edit %[1]q and add:\x0a\x0a\x09jid=\x22me@example.com\x22\x02user addre" +
	"ss: %[1]q\x02error parsing user address: %[1]q\x02error parsing timeout," +
	" defaulting to 30s: %[1]q\x02error creating keylog file: %[1]q\x02error " +
	"retrieving roster version, falling back to full roster fetch: %[1]v\x02i" +
	"nitial login failed: %[1]v\x02logged in as: %[1]q\x02got signal: %[1]v" +
	"\x02executing command: %+[1]v\x02error executing command %[1]q on %[2]q:" +
	" %[3]v\x02error showing next command for %[1]q: %[2]v\x02error fetching " +
	"commands for %[1]q: %[2]v\x02error closing commands iter for %[1]q: %[2]" +
	"v\x02error setting away status: %[1]v\x02error setting online status: %[" +
	"1]v\x02error setting busy status: %[1]v\x02error going offline: %[1]v" +
	"\x02error adding roster item %[1]s: %[2]v\x02error removing roster item " +
	"%[1]s: %[2]v\x02error discovering bookmarks support: %[1]v\x02error publ" +
	"ishing legacy bookmark %[1]s: %[2]v\x02error publishing bookmark %[1]s: " +
	"%[2]v\x02error removing legacy bookmark %[1]s: %[2]v\x02error removing b" +
	"ookmark %[1]s: %[2]v\x02error sending presence pre-approval to %[1]s: %[" +
	"2]v\x02error sending presence request to %[1]s: %[2]v\x02unrecognized ui" +
	" event: %[1]T(%[1]q)\x02error sending message: %[1]v\x02error saving sen" +
	"t message to history: %[1]v\x02could not get the upload services: %[1]v" +
	"\x02no upload service available\x02could not upload %[1]q: %[2]v\x02erro" +
	"r loading chat: %[1]v\x02invalid nick %[1]s in config: %[2]v\x02joining " +
	"room %[1]v…\x02error joining room %[1]s: %[2]v\x02error fetching earlies" +
	"t message info for %[1]v from database: %[2]v\x02no scrollback for %[1]v" +
	"\x02fetching scrollback before %[1]v for %[2]v…\x02error fetching scroll" +
	"back for %[1]v: %[2]v\x02Error while handling XMPP streams: %[1]q\x02Err" +
	"or going offline: %[1]q\x02Error closing the connection: %[1]q\x02error " +
	"occured during service discovery: %[1]v\x02error when closing the items " +
	"iterator: %[1]v\x02feature discovery failed for %[1]q: %[2]v\x02error fe" +
	"tching roster: %[1]q\x02error fetching bookmarks: %[1]q\x02error fetchin" +
	"g version information: %[1]v\x02To fix this, contact your server adminis" +
	"trator and ask them to enable %[1]q\x02Your server does not support book" +
	"mark unification, an important feature that stops newer clients from see" +
	"ing a different list of chat rooms than older clients that do not yet su" +
	"pport the latest features.\x02error closing bookmarks stream: %[1]v\x02E" +
	"rror closing roster stream: %[1]q\x02possibly spoofed history message fr" +
	"om %[1]s\x02Add Contact\x02Address\x02Name\x02Channels\x02Conversation" +
	"\x02error while picking files: %[1]v\x02failed to get the recipient\x02C" +
	"onversations\x02Logs\x02Remove\x02Remove this contact from your roster?" +
	"\x02Remove this channel?\x02Offline\x02Online\x02Away\x02Busy\x02Are you" +
	" sure you want to quit?\x02Quit\x02Join\x02Join Channel\x02Add\x02Comman" +
	"ds\x02Loading commands…\x02Exec\x02failed to read stderr of the notifica" +
	"tion subprocess: %[1]v\x02failed to run notification command: %[1]v\x02n" +
	"otification subprocess failed: %[1]v\x0a%[2]s\x02error canceling command" +
	" session: %[1]v\x02error closing cancel command payload: %[1]v\x02error " +
	"closing command session: %[1]v\x02error showing next command: %[1]v\x04" +
	"\x00\x01\x0a\xb2\x04\x02# This is a config file for Communiqué.\x0a# If " +
	"the -f option is not provided, Communiqué will search for a config file " +
	"in:\x0a#\x0a#   - ./communiqué.toml\x0a#   - $XDG_CONFIG_HOME/communiqué" +
	"/config.toml\x0a#   - $HOME/.config/communiqué/config.toml\x0a#   - /etc" +
	"/communiqué/config.toml\x0a#\x0a# The only required field is \x22address" +
	"\x22. The \x22password_eval\x22 field should be set\x0a# to a command th" +
	"at writes the password to standard out. Normally this should\x0a# decryp" +
	"t an encrypted file containing the password. If it is not specified, the" +
	"\x0a# user will be prompted to enter a password.\x02uploaded %[1]q as %[" +
	"2]s\x02error loading scrollback into pane for %[1]v: %[2]v\x02error dial" +
	"ing connection: %[1]v\x02error negotiating session: %[1]v\x02error enabl" +
	"ing carbons: %[1]q\x02error setting deadline: %[1]v\x02error closing fil" +
	"e: %[1]v\x02cannot upload directory\x02error when closing response body:" +
	" %[1]v\x02unexpected status code: %[1]d (%[2]s)\x02error finding user ho" +
	"me directory: %[1]v\x02error getting current working directory: %[1]v" +
	"\x02error creating db dir, skipping: %[1]v\x02error opening or creating " +
	"db, skipping: %[1]v\x02error closing db file: %[1]v\x02could not create " +
	"or open database for writing\x02error opening DB: %[1]v\x02error enablin" +
	"g foreign keys: %[1]v\x02error applying schema: %[1]v\x02error getting c" +
	"aps: %[1]v\x02bad hash type found in database: %[1]v\x02error getting id" +
	"entities: %[1]v\x02error scanning identity: %[1]v\x02error iterating ove" +
	"r identity rows: %[1]v\x02error closing identity rows: %[1]v\x02error ge" +
	"tting features: %[1]v\x02error scanning feature row: %[1]v\x02error iter" +
	"ating over feature rows: %[1]v\x02error closing feature rows: %[1]v\x02e" +
	"rror querying for disco forms: %[1]v\x02error decoding forms: %[1]v\x02e" +
	"rror inserting caps: %[1]v\x02encoding root forms start element failed: " +
	"%[1]v\x02encoding form failed: %[1]v\x02encoding root forms end element " +
	"failed: %[1]v\x02flushing encoded form failed: %[1]v\x02error inserting " +
	"JIDCapsForm: %[1]v\x02error inserting identity %[1]v: %[2]v\x02error ins" +
	"erting identity caps joiner: %[1]v\x02error inserting feature %[1]s: %[2" +
	"]v\x02error inserting feature caps joiner: %[1]v\x02error getting servic" +
	"es: %[1]v\x02error scanning services: %[1]v\x02failed to parse service J" +
	"ID: %[1]v\x02error iterating over services rows: %[1]v\x02Password\x02Lo" +
	"gin\x02Enter password for: %[1]q\x02Roster\x02Chat: %[1]q (%[2]s)\x02Sta" +
	"tus: %[1]s\x02Search\x02Set Status\x02Online %[1]s\x02Away %[1]s\x02Busy" +
	" %[1]s\x02Offline %[1]s\x02no file picker set, see the example configura" +
	"tion file for more information\x02failed to read process' standard error" +
	": %[1]v\x02failed to read process' standard output: %[1]v\x02Data Form" +
	"\x02error setting bool form field %[1]s: %[2]v\x02error parsing jid-mult" +
	"i value for field %[1]s: %[2]v\x02error setting jid-multi form field %[1" +
	"]s: %[2]v\x02error setting jid form field %[1]s: %[2]v\x02error setting " +
	"list-multi form field %[1]s: %[2]v\x02error setting list form field %[1]" +
	"s: %[2]v\x02error setting text form field %[1]s: %[2]v\x02error setting " +
	"text-multi form field %[1]s: %[2]v\x02error setting password form field " +
	"%[1]s: %[2]v\x02No commands found for %[1]v!\x02Select\x02Pick Address" +
	"\x02Roster info:\x02no sidebar open, not showing info pane…\x02unrecogni" +
	"zed sidebar item type %[1]T, not showing info…\x02error executing info t" +
	"emplate: %[1]v"

	// Total table size 14815 bytes (14KiB); checksum: 99652B5B
