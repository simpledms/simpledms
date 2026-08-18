package tenantdatamigration

import (
	"context"

	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/sqlx"
)

type Migration interface {
	Key() string
	RunBatch(context.Context, *sqlx.TenantDB, *enttenant.TenantDataMigration) (*BatchResult, error)
}

type BatchResult struct {
	Cursor    int64
	Completed bool
	Apply     func(context.Context, *enttenant.Tx) error
}

func NewBatchResult(
	cursor int64,
	completed bool,
	apply func(context.Context, *enttenant.Tx) error,
) *BatchResult {
	return &BatchResult{
		Cursor:    cursor,
		Completed: completed,
		Apply:     apply,
	}
}
