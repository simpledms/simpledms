package sqlx

import (
	"log"
	"sync/atomic"
	"time"
)

type QueryTimingLogger struct {
	queryTimingLoggingEnabled atomic.Bool
	queryLogIDCounter         atomic.Uint64
}

func NewQueryTimingLogger() *QueryTimingLogger {
	return &QueryTimingLogger{}
}

var queryTimingLogger = NewQueryTimingLogger()

func (qq *QueryTimingLogger) Enable() {
	qq.queryTimingLoggingEnabled.Store(true)
}

func (qq *QueryTimingLogger) IsEnabled() bool {
	return qq.queryTimingLoggingEnabled.Load()
}

func (qq *QueryTimingLogger) NextQueryLogID() uint64 {
	return qq.queryLogIDCounter.Add(1)
}

func (qq *QueryTimingLogger) LogQuery(operation string, queryID uint64, query string, args any) {
	log.Printf("ent query: id=%d op=%s query=%v args=%v", queryID, operation, query, args)
}

func (qq *QueryTimingLogger) LogOperation(operation string, queryID uint64) {
	log.Printf("ent query: id=%d op=%s", queryID, operation)
}

func (qq *QueryTimingLogger) LogQueryTiming(
	operation string,
	queryID uint64,
	startedAt time.Time,
	err error,
) {
	if err != nil {
		log.Printf("ent query timing: id=%d op=%s elapsed=%s err=%v", queryID, operation, time.Since(startedAt), err)
		return
	}
	log.Printf("ent query timing: id=%d op=%s elapsed=%s", queryID, operation, time.Since(startedAt))
}

func (qq *QueryTimingLogger) LogQueryConsumption(
	operation string,
	queryID uint64,
	startedAt time.Time,
	openedAt time.Time,
	rowCount int64,
	queryErr error,
	closeErr error,
) {
	totalElapsed := time.Since(startedAt)
	openElapsed := openedAt.Sub(startedAt)
	if openElapsed < 0 {
		openElapsed = 0
	}

	consumeElapsed := totalElapsed - openElapsed
	if consumeElapsed < 0 {
		consumeElapsed = 0
	}

	if queryErr != nil || closeErr != nil {
		log.Printf(
			"ent query consume timing: id=%d op=%s open_elapsed=%s consume_elapsed=%s total_elapsed=%s rows=%d query_err=%v close_err=%v",
			queryID,
			operation,
			openElapsed,
			consumeElapsed,
			totalElapsed,
			rowCount,
			queryErr,
			closeErr,
		)
		return
	}

	log.Printf(
		"ent query consume timing: id=%d op=%s open_elapsed=%s consume_elapsed=%s total_elapsed=%s rows=%d",
		queryID,
		operation,
		openElapsed,
		consumeElapsed,
		totalElapsed,
		rowCount,
	)
}
