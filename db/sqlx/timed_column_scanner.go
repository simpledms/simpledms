package sqlx

import (
	"database/sql"
	"sync"
	"time"

	entsql "entgo.io/ent/dialect/sql"
)

type timedColumnScanner struct {
	columnScanner entsql.ColumnScanner
	operation     string
	queryID       uint64
	startedAt     time.Time
	openedAt      time.Time
	rowCount      int64
	nilableErr    error
	closeOnce     sync.Once
}

func newTimedColumnScanner(
	columnScanner entsql.ColumnScanner,
	operation string,
	queryID uint64,
	startedAt time.Time,
	openedAt time.Time,
) entsql.ColumnScanner {
	return &timedColumnScanner{
		columnScanner: columnScanner,
		operation:     operation,
		queryID:       queryID,
		startedAt:     startedAt,
		openedAt:      openedAt,
	}
}

func (qq *timedColumnScanner) Close() error {
	closeErr := qq.columnScanner.Close()

	qq.closeOnce.Do(func() {
		if queryErr := qq.columnScanner.Err(); queryErr != nil {
			qq.nilableErr = queryErr
		}
		queryTimingLogger.LogQueryConsumption(
			qq.operation,
			qq.queryID,
			qq.startedAt,
			qq.openedAt,
			qq.rowCount,
			qq.nilableErr,
			closeErr,
		)
	})

	return closeErr
}

func (qq *timedColumnScanner) ColumnTypes() ([]*sql.ColumnType, error) {
	return qq.columnScanner.ColumnTypes()
}

func (qq *timedColumnScanner) Columns() ([]string, error) {
	return qq.columnScanner.Columns()
}

func (qq *timedColumnScanner) Err() error {
	err := qq.columnScanner.Err()
	if err != nil {
		qq.nilableErr = err
	}
	return err
}

func (qq *timedColumnScanner) Next() bool {
	hasNext := qq.columnScanner.Next()
	if hasNext {
		qq.rowCount++
	} else if err := qq.columnScanner.Err(); err != nil {
		qq.nilableErr = err
	}
	return hasNext
}

func (qq *timedColumnScanner) NextResultSet() bool {
	return qq.columnScanner.NextResultSet()
}

func (qq *timedColumnScanner) Scan(dest ...any) error {
	err := qq.columnScanner.Scan(dest...)
	if err != nil {
		qq.nilableErr = err
	}
	return err
}

var _ entsql.ColumnScanner = (*timedColumnScanner)(nil)
