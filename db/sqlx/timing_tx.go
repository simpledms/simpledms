package sqlx

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
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
	queryTimingLogger.LogQueryTiming("Tx.Query", queryID, startedAt, err)
	return err
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
