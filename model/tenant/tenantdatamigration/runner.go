package tenantdatamigration

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"strings"
	"time"

	"entgo.io/ent/privacy"

	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/tenantdatamigration"
	"github.com/simpledms/simpledms/db/sqlx"
)

const migrationLeaseDuration = 15 * time.Minute

var ErrLostLease = errors.New("tenant data migration lease lost")

type Runner struct {
	now func() time.Time
}

func NewRunner() *Runner {
	return NewRunnerWithClock(time.Now)
}

func NewRunnerWithClock(now func() time.Time) *Runner {
	return &Runner{now: now}
}

func (qq *Runner) Run(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	migration Migration,
) (bool, error) {
	ctx = privacy.DecisionContext(ctx, privacy.Allow)
	state, leaseToken, claimed, err := qq.claim(ctx, tenantDB, migration.Key())
	if err != nil || !claimed {
		return false, err
	}

	result, err := migration.RunBatch(ctx, tenantDB, state)
	if err != nil {
		qq.fail(ctx, tenantDB, state.ID, leaseToken, err)
		return false, err
	}
	if err := qq.commit(ctx, tenantDB, state.ID, leaseToken, result); err != nil {
		if !errors.Is(err, ErrLostLease) {
			qq.fail(ctx, tenantDB, state.ID, leaseToken, err)
		}
		return false, err
	}
	return result.Completed, nil
}

func (qq *Runner) claim(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	key string,
) (*enttenant.TenantDataMigration, string, bool, error) {
	now := qq.now().UTC()
	token, err := migrationLeaseToken()
	if err != nil {
		return nil, "", false, err
	}
	state, err := tenantDB.ReadOnlyConn.TenantDataMigration.Query().
		Where(tenantdatamigration.KeyEQ(key)).
		Only(ctx)
	if enttenant.IsNotFound(err) {
		state, err = tenantDB.ReadWriteConn.TenantDataMigration.Create().
			SetKey(key).
			SetFirstStartedAt(now).
			SetLastAttemptedAt(now).
			SetLeaseToken(token).
			SetLeaseExpiresAt(now.Add(migrationLeaseDuration)).
			Save(ctx)
		if enttenant.IsConstraintError(err) {
			return qq.claim(ctx, tenantDB, key)
		}
		return state, token, err == nil, err
	}
	if err != nil {
		return nil, "", false, err
	}
	if state.CompletedAt != nil || state.LeaseExpiresAt != nil && state.LeaseExpiresAt.After(now) {
		return state, "", false, nil
	}

	state, err = tenantDB.ReadWriteConn.TenantDataMigration.UpdateOneID(state.ID).
		Where(
			tenantdatamigration.CompletedAtIsNil(),
			tenantdatamigration.Or(
				tenantdatamigration.LeaseExpiresAtIsNil(),
				tenantdatamigration.LeaseExpiresAtLTE(now),
			),
		).
		SetLastAttemptedAt(now).
		SetLeaseToken(token).
		SetLeaseExpiresAt(now.Add(migrationLeaseDuration)).
		Save(ctx)
	if enttenant.IsNotFound(err) {
		return state, "", false, nil
	}
	return state, token, err == nil, err
}

func (qq *Runner) commit(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	stateID int,
	leaseToken string,
	result *BatchResult,
) error {
	tx, err := tenantDB.ReadWriteConn.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Println(err)
		}
	}()

	if result.Apply != nil {
		if err := result.Apply(ctx, tx); err != nil {
			return err
		}
	}
	update := tx.TenantDataMigration.UpdateOneID(stateID).
		Where(tenantdatamigration.LeaseTokenEQ(leaseToken)).
		SetCursor(result.Cursor).
		ClearLeaseToken().
		ClearLeaseExpiresAt().
		ClearFailedAt().
		ClearLastError()
	if result.Completed {
		update.SetCompletedAt(qq.now().UTC())
	}
	if _, err := update.Save(ctx); err != nil {
		if enttenant.IsNotFound(err) {
			return ErrLostLease
		}
		return err
	}
	return tx.Commit()
}

func (qq *Runner) fail(
	ctx context.Context,
	tenantDB *sqlx.TenantDB,
	stateID int,
	leaseToken string,
	migrationErr error,
) {
	message := migrationErr.Error()
	if len(message) > 4000 {
		message = strings.Clone(message[:4000])
	}
	err := tenantDB.ReadWriteConn.TenantDataMigration.UpdateOneID(stateID).
		Where(tenantdatamigration.LeaseTokenEQ(leaseToken)).
		SetFailedAt(qq.now().UTC()).
		SetLastError(message).
		AddRetryCount(1).
		ClearLeaseToken().
		ClearLeaseExpiresAt().
		Exec(ctx)
	if err != nil {
		log.Println(err)
	}
}

func migrationLeaseToken() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
