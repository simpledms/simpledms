package sqlx

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

type timingTx struct {
	tx dialect.Tx
}

func newTimingTx(tx dialect.Tx) *timingTx {
	return &timingTx{
		tx: tx,
	}
}

func (qq *timingTx) Exec(ctx context.Context, query string, args, v any) error {
	if !queryTimingLogger.IsEnabled() {
		return qq.tx.Exec(ctx, query, args, v)
	}

	queryID := queryTimingLogger.NextQueryLogID()
	queryTimingLogger.LogQuery("Tx.Exec", queryID, query, args)
	startedAt := time.Now()
	err := qq.tx.Exec(ctx, query, args, v)
	queryTimingLogger.LogQueryTiming("Tx.Exec", queryID, startedAt, err)
	return err
}

func (qq *timingTx) Query(ctx context.Context, query string, args, v any) error {
	if !queryTimingLogger.IsEnabled() {
		return qq.tx.Query(ctx, query, args, v)
	}

	queryID := queryTimingLogger.NextQueryLogID()
	queryTimingLogger.LogQuery("Tx.Query", queryID, query, args)
	startedAt := time.Now()
	err := qq.tx.Query(ctx, query, args, v)
	openedAt := time.Now()
	queryTimingLogger.LogQueryTiming("Tx.Query", queryID, startedAt, err)
	if err != nil {
		return err
	}
	if qq.shouldSkipRowScannerWrapping() {
		return err
	}

	rows, ok := v.(*entsql.Rows)
	if ok && rows != nil && rows.ColumnScanner != nil {
		rows.ColumnScanner = newTimedColumnScanner(
			rows.ColumnScanner,
			"Tx.Query",
			queryID,
			startedAt,
			openedAt,
		)
	}

	return err
}

func (qq *timingTx) shouldSkipRowScannerWrapping() bool {
	const callDepth = 16
	pcs := make([]uintptr, callDepth)
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		if strings.Contains(frame.File, "/dialect/sql/schema/") ||
			strings.Contains(frame.File, "/ariga.io/atlas@") {
			return true
		}
		if !more {
			break
		}
	}

	return false
}

func (qq *timingTx) Commit() error {
	if !queryTimingLogger.IsEnabled() {
		return qq.tx.Commit()
	}

	queryID := queryTimingLogger.NextQueryLogID()
	queryTimingLogger.LogOperation("Tx.Commit", queryID)
	startedAt := time.Now()
	err := qq.tx.Commit()
	queryTimingLogger.LogQueryTiming("Tx.Commit", queryID, startedAt, err)
	return err
}

func (qq *timingTx) Rollback() error {
	if !queryTimingLogger.IsEnabled() {
		return qq.tx.Rollback()
	}

	queryID := queryTimingLogger.NextQueryLogID()
	queryTimingLogger.LogOperation("Tx.Rollback", queryID)
	startedAt := time.Now()
	err := qq.tx.Rollback()
	queryTimingLogger.LogQueryTiming("Tx.Rollback", queryID, startedAt, err)
	return err
}

func (qq *timingTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	txWithExecContext, ok := qq.tx.(interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	})
	if !ok {
		return nil, fmt.Errorf("Tx.ExecContext is not supported")
	}

	if !queryTimingLogger.IsEnabled() {
		return txWithExecContext.ExecContext(ctx, query, args...)
	}

	queryID := queryTimingLogger.NextQueryLogID()
	queryTimingLogger.LogQuery("Tx.ExecContext", queryID, query, args)
	startedAt := time.Now()
	result, err := txWithExecContext.ExecContext(ctx, query, args...)
	queryTimingLogger.LogQueryTiming("Tx.ExecContext", queryID, startedAt, err)
	return result, err
}

func (qq *timingTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	txWithQueryContext, ok := qq.tx.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, fmt.Errorf("Tx.QueryContext is not supported")
	}

	if !queryTimingLogger.IsEnabled() {
		return txWithQueryContext.QueryContext(ctx, query, args...)
	}

	queryID := queryTimingLogger.NextQueryLogID()
	queryTimingLogger.LogQuery("Tx.QueryContext", queryID, query, args)
	startedAt := time.Now()
	rows, err := txWithQueryContext.QueryContext(ctx, query, args...)
	queryTimingLogger.LogQueryTiming("Tx.QueryContext", queryID, startedAt, err)
	return rows, err
}

var _ dialect.Tx = (*timingTx)(nil)
