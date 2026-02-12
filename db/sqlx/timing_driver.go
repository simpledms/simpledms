package sqlx

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
)

type timingDriver struct {
	driver dialect.Driver
}

func newTimingDriver(driver dialect.Driver) *timingDriver {
	return &timingDriver{
		driver: driver,
	}
}

func (qq *timingDriver) Exec(ctx context.Context, query string, args, v any) error {
	if !queryTimingLogger.IsEnabled() {
		return qq.driver.Exec(ctx, query, args, v)
	}

	queryID := queryTimingLogger.NextQueryLogID()
	queryTimingLogger.LogQuery("Exec", queryID, query, args)
	startedAt := time.Now()
	err := qq.driver.Exec(ctx, query, args, v)
	queryTimingLogger.LogQueryTiming("Exec", queryID, startedAt, err)
	return err
}

func (qq *timingDriver) Query(ctx context.Context, query string, args, v any) error {
	if !queryTimingLogger.IsEnabled() {
		return qq.driver.Query(ctx, query, args, v)
	}

	queryID := queryTimingLogger.NextQueryLogID()
	queryTimingLogger.LogQuery("Query", queryID, query, args)
	startedAt := time.Now()
	err := qq.driver.Query(ctx, query, args, v)
	queryTimingLogger.LogQueryTiming("Query", queryID, startedAt, err)
	return err
}

func (qq *timingDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	tx, err := qq.driver.Tx(ctx)
	if err != nil {
		return nil, err
	}
	return newTimingTx(tx), nil
}

func (qq *timingDriver) BeginTx(ctx context.Context, opts *sql.TxOptions) (dialect.Tx, error) {
	driverWithBeginTx, ok := qq.driver.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	})
	if !ok {
		return nil, fmt.Errorf("Driver.BeginTx is not supported")
	}

	tx, err := driverWithBeginTx.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newTimingTx(tx), nil
}

func (qq *timingDriver) Close() error {
	return qq.driver.Close()
}

func (qq *timingDriver) Dialect() string {
	return qq.driver.Dialect()
}

func (qq *timingDriver) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	driverWithExecContext, ok := qq.driver.(interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	})
	if !ok {
		return nil, fmt.Errorf("Driver.ExecContext is not supported")
	}

	if !queryTimingLogger.IsEnabled() {
		return driverWithExecContext.ExecContext(ctx, query, args...)
	}

	queryID := queryTimingLogger.NextQueryLogID()
	queryTimingLogger.LogQuery("ExecContext", queryID, query, args)
	startedAt := time.Now()
	result, err := driverWithExecContext.ExecContext(ctx, query, args...)
	queryTimingLogger.LogQueryTiming("ExecContext", queryID, startedAt, err)
	return result, err
}

func (qq *timingDriver) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	driverWithQueryContext, ok := qq.driver.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, fmt.Errorf("Driver.QueryContext is not supported")
	}

	if !queryTimingLogger.IsEnabled() {
		return driverWithQueryContext.QueryContext(ctx, query, args...)
	}

	queryID := queryTimingLogger.NextQueryLogID()
	queryTimingLogger.LogQuery("QueryContext", queryID, query, args)
	startedAt := time.Now()
	rows, err := driverWithQueryContext.QueryContext(ctx, query, args...)
	queryTimingLogger.LogQueryTiming("QueryContext", queryID, startedAt, err)
	return rows, err
}

var _ dialect.Driver = (*timingDriver)(nil)
