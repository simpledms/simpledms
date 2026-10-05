package sqlx

import (
	"database/sql"
	"time"
)

// Idle connections keep SQLite's per-connection page cache and parsed schema. Opening a
// connection for every transaction made a short read transaction about 10x slower in a local
// measurement. The idle timeout still closes connections of tenants that are not in use, which
// bounds open file handles.
const connMaxIdleTime = 5 * time.Minute

func configureReadOnlyPool(db *sql.DB) {
	maxOpenConns := readOnlyMaxOpenConns()
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxOpenConns)
	db.SetConnMaxIdleTime(connMaxIdleTime)
}

// A single read-write connection serializes writes in Go instead of relying on SQLite's busy
// timeout; see SQLiteQueryParamsReadWrite.
func configureReadWritePool(db *sql.DB) {
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(connMaxIdleTime)
}
