package ctxx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	ctxx2 "github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/db/entmain"
	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/sqlx"
)

type AppContext struct {
	*ctxx2.TenantContext
	TTx             *enttenant.Tx // TODO rename to AppTx? and enttenant to entapp?
	User            *enttenant.User
	isReadOnly      bool
	unsafeTenantDBs *tenantdbs.TenantDBs
}

func NewAppContext(
	tenantContext *ctxx2.TenantContext,
	tenantTx *enttenant.Tx,
	isReadOnly bool,
	unsafeTenantDBs *tenantdbs.TenantDBs,
) *AppContext {
	appCtx, err := NewAppContextWithError(
		tenantContext,
		tenantTx,
		isReadOnly,
		unsafeTenantDBs,
	)
	if err != nil {
		panic(err)
	}
	return appCtx
}

func NewAppContextWithError(
	tenantContext *ctxx2.TenantContext,
	tenantTx *enttenant.Tx,
	isReadOnly bool,
	unsafeTenantDBs *tenantdbs.TenantDBs,
) (*AppContext, error) {
	userx, err := tenantTx.User.Query().
		Where(user.AccountID(tenantContext.Account.ID)).
		Only(tenantContext)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	appCtx := &AppContext{
		TenantContext:   tenantContext,
		TTx:             tenantTx,
		User:            userx,
		isReadOnly:      isReadOnly,
		unsafeTenantDBs: unsafeTenantDBs,
	}
	appCtx.Context = context.WithValue(tenantContext.Context, appCtxKey, appCtx)
	return appCtx, nil
}

func (qq *AppContext) AppCtx() *AppContext {
	return qq
}

func (qq *AppContext) IsReadOnlyTx() bool {
	return qq.isReadOnly
}

func (qq *AppContext) IsAppCtx() bool {
	return true
}

func (qq *AppContext) SpaceCtx() *SpaceContext {
	panic("context not available")
}

func (qq *AppContext) IsSpaceCtx() bool {
	return false
}

// TODO cache?
func (qq *AppContext) ReadOnlyAccountSpacesByTenant() (map[*entmain.Tenant][]*enttenant.Space, error) {
	var spacesByTenant = make(map[*entmain.Tenant][]*enttenant.Space)

	// similar code in DashboardCards
	tenants, err := qq.Account.QueryTenants().All(qq)
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("failed to query tenants for account %d: %w", qq.Account.ID, err)
	}

	for _, tenantx := range tenants {
		var spaces []*enttenant.Space

		tenantDB, ok := qq.unsafeTenantDBs.Load(tenantx.ID)
		if !ok {
			log.Println("tenant db not found, tenant id was", tenantx.ID)
			continue
		}

		tenantTx, err := tenantDB.ReadOnlyConn.Tx(qq)
		if err != nil {
			log.Println("failed to start transaction for tenant", tenantx.ID, err)
			continue
		}

		// necessary for permissions
		tenantCtx := ctxx2.NewTenantContext(qq.MainContext, tenantx)

		// spaces = append(spaces, tenantDB.Space.Query().AllX(ctx)...)
		spacesx, err := tenantDB.ReadOnlyConn.Space.Query().All(tenantCtx)
		if err != nil && !enttenant.IsNotFound(err) {
			log.Println("failed to query spaces for tenant", tenantx.ID, err)
			qq.rollbackTenantTx(tenantTx, tenantx.ID)
			continue
		}
		spaces = append(spaces, spacesx...)

		// TODO not sure if necessary... may could also just use db directly or rollback if faster?
		// TODO is it a problem that spaces get used in calling function after the tx is committed?
		if err := tenantTx.Commit(); err != nil {
			log.Println("failed to commit transaction for tenant", tenantx.ID, err)
			qq.rollbackTenantTx(tenantTx, tenantx.ID)
			continue
		}

		spacesByTenant[tenantx] = spaces
	}

	return spacesByTenant, nil
}

func (qq *AppContext) rollbackTenantTx(tenantTx *enttenant.Tx, tenantID int64) {
	if err := tenantTx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		log.Println("failed to rollback transaction for tenant", tenantID, err)
	}
}

func (qq *AppContext) UnsafeTenantDB(tenantID int64) (*sqlx.TenantDB, bool) {
	return qq.unsafeTenantDBs.Load(tenantID)
}
func (qq *AppContext) UnsafeTenantDBs() *tenantdbs.TenantDBs {
	return qq.unsafeTenantDBs
}
