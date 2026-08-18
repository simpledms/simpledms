package tenantdatamigration

import (
	"context"
	"errors"
	"testing"
	"time"

	"entgo.io/ent/privacy"
	_ "github.com/mattn/go-sqlite3"

	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/enttest"
	"github.com/simpledms/simpledms/db/enttenant/tenantdatamigration"
	"github.com/simpledms/simpledms/db/sqlx"
)

type testMigration struct {
	key   string
	run   func(*enttenant.TenantDataMigration) (*BatchResult, error)
	state []*enttenant.TenantDataMigration
}

func newTestMigration(
	key string,
	run func(*enttenant.TenantDataMigration) (*BatchResult, error),
) *testMigration {
	return &testMigration{key: key, run: run}
}

func (qq *testMigration) Key() string {
	return qq.key
}

func (qq *testMigration) RunBatch(
	_ context.Context,
	_ *sqlx.TenantDB,
	state *enttenant.TenantDataMigration,
) (*BatchResult, error) {
	qq.state = append(qq.state, state)
	return qq.run(state)
}

func TestRunnerPersistsFirstStartCursorAndCompletion(t *testing.T) {
	ctx, tenantDB := newMigrationTestDB(t)
	now := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
	runner := NewRunnerWithClock(func() time.Time { return now })
	migration := newTestMigration("test", func(state *enttenant.TenantDataMigration) (*BatchResult, error) {
		return NewBatchResult(state.Cursor+1, state.Cursor == 1, nil), nil
	})

	completed, err := runner.Run(ctx, tenantDB, migration)
	if err != nil || completed {
		t.Fatalf("first batch completed=%t, err=%v", completed, err)
	}
	now = now.Add(time.Hour)
	completed, err = runner.Run(ctx, tenantDB, migration)
	if err != nil || !completed {
		t.Fatalf("second batch completed=%t, err=%v", completed, err)
	}

	state := tenantDB.ReadOnlyConn.TenantDataMigration.Query().OnlyX(ctx)
	if !state.FirstStartedAt.Equal(migration.state[0].FirstStartedAt) || state.Cursor != 2 || state.CompletedAt == nil {
		t.Fatalf("unexpected migration state: %#v", state)
	}
	if _, err := runner.Run(ctx, tenantDB, migration); err != nil {
		t.Fatal(err)
	}
	if len(migration.state) != 2 {
		t.Fatalf("completed migration ran %d batches, want 2", len(migration.state))
	}
}

func TestRunnerRecordsAndRetriesFailures(t *testing.T) {
	ctx, tenantDB := newMigrationTestDB(t)
	runner := NewRunner()
	shouldFail := true
	migration := newTestMigration("failing", func(state *enttenant.TenantDataMigration) (*BatchResult, error) {
		if shouldFail {
			return nil, errors.New("broken input")
		}
		return NewBatchResult(state.Cursor, true, nil), nil
	})

	if _, err := runner.Run(ctx, tenantDB, migration); err == nil {
		t.Fatal("expected migration failure")
	}
	state := tenantDB.ReadOnlyConn.TenantDataMigration.Query().
		Where(tenantdatamigration.KeyEQ("failing")).
		OnlyX(ctx)
	if state.FailedAt == nil || state.LastError == nil || state.RetryCount != 1 || state.Cursor != 0 {
		t.Fatalf("failure was not recorded: %#v", state)
	}

	shouldFail = false
	completed, err := runner.Run(ctx, tenantDB, migration)
	if err != nil || !completed {
		t.Fatalf("retry completed=%t, err=%v", completed, err)
	}
}

func newMigrationTestDB(t *testing.T) (context.Context, *sqlx.TenantDB) {
	t.Helper()
	ctx := privacy.DecisionContext(context.Background(), privacy.Allow)
	client := enttest.Open(t, "sqlite3", "file:tenant-migration?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Fatal(err)
		}
	})
	return ctx, &sqlx.TenantDB{DB: &sqlx.DB[*enttenant.Client, *enttenant.Tx]{
		ReadOnlyConn:  client,
		ReadWriteConn: client,
	}}
}
