package server

import (
	"context"
	"testing"

	"github.com/marcobeierer/go-core/db/entmain"

	ctxx2 "github.com/marcobeierer/go-core/ctxx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/sqlx"
)

func withTenantContext(
	t *testing.T,
	harness *actionTestHarness,
	accountx *entmain.Account,
	tenantx *entmain.Tenant,
	tenantDB *sqlx.TenantDB,
	fn func(mainTx *entmain.Tx, tenantTx *enttenant.Tx, tenantCtx *ctxx.AppContext) error,
) error {
	t.Helper()

	mainTx, tenantTx, tenantCtx := newTenantContext(t, harness, accountx, tenantx, tenantDB)
	committed := false
	defer func() {
		if committed {
			return
		}
		_ = tenantTx.Rollback()
		_ = mainTx.Rollback()
	}()

	if err := fn(mainTx, tenantTx, tenantCtx); err != nil {
		return err
	}

	if err := mainTx.Commit(); err != nil {
		_ = tenantTx.Rollback()
		t.Fatalf("commit main tx: %v", err)
	}
	if err := tenantTx.Commit(); err != nil {
		t.Fatalf("commit tenant tx: %v", err)
	}
	committed = true
	return nil
}

func withMainContext(
	t *testing.T,
	harness *actionTestHarness,
	accountx *entmain.Account,
	fn func(mainTx *entmain.Tx, mainCtx ctxx.Context) error,
) error {
	t.Helper()

	mainTx, err := harness.mainDB.ReadWriteConn.Tx(context.Background())
	if err != nil {
		t.Fatalf("start main tx: %v", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		_ = mainTx.Rollback()
	}()

	visitorCtx := ctxx2.NewVisitorContext(
		context.Background(),
		mainTx,
		harness.i18n,
		"",
		"",
		true,
		false,
		harness.infra.SystemConfig().CommercialLicenseEnabled(),
	)
	mainCtxCore := ctxx2.NewMainContext(visitorCtx, accountx, harness.i18n, harness.mainDB, false)
	mainCtx := ctxx.WrapContext(mainCtxCore)

	if err := fn(mainTx, mainCtx); err != nil {
		return err
	}

	if err := mainTx.Commit(); err != nil {
		t.Fatalf("commit main tx: %v", err)
	}
	committed = true
	return nil
}
