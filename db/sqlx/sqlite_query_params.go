package sqlx

// all are driver specific for mattn
// shared cache is obsolete with wal enabled, see https://www.sqlite.org/sharedcache.html
//
// TODO
// busy timeout may not be necessary for write queries if SetMaxOpenConns is set to 1 because
// Go then handles pooling and protects the connection by a mutex
//
// TODO set timezone (_loc?)
// TODO set cache_size and Temp_store?
// https://kerkour.com/sqlite-for-servers
var sqliteQueryParams = "cache=private&_foreign_keys=1&_journal_mode=wal&_synchronous=normal&_busy_timeout=5000"

var SQLiteQueryParamsReadOnly = "mode=ro&" + sqliteQueryParams

// txlock immediate, makes client aquire a lock when connection is started. without, it is aquired
// when a write action is executed; the latter has the disadvantage that BusyTimeout is not applied
// and thus the client may get a `database is locked` error; see comment above
//
// TODO for does the `create` in rwc do exactly? is rw enough?
var SQLiteQueryParamsReadWrite = "mode=rwc&_txlock=immediate&" + sqliteQueryParams
