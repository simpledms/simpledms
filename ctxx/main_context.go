package ctxx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/sqlx"
	"github.com/simpledms/simpledms/i18n"
)

type MainContext struct {
	context.Context
	*VisitorContext
	Account *entmain.Account // modelmain.Account would be better, but leads to circular dependency
	// should never be exposed directly;
	// unsafe because must be used with care
	unsafeMainDB    *sqlx.MainDB
	unsafeTenantDBs *tenantdbs.TenantDBs
	isReadOnly      bool
}

func NewMainContext(
	ctx *VisitorContext,
	account *entmain.Account,
	i18nx *i18n.I18n,
	mainDB *sqlx.MainDB,
	tenantDBs *tenantdbs.TenantDBs,
	isReadOnly bool,
) *MainContext {
	ctx.Printer = i18nx.Printer(account.Language.Tag())
	langTagBase, _ := account.Language.Tag().Base() // TODO evaluate confidence?
	ctx.LanguageBCP47 = langTagBase.String()

	mainCtx := &MainContext{
		VisitorContext:  ctx,
		Account:         account,
		unsafeMainDB:    mainDB,
		unsafeTenantDBs: tenantDBs,
		isReadOnly:      isReadOnly,
	}
	mainCtx.Context = context.WithValue(ctx.Context, mainCtxKey, mainCtx)
	return mainCtx
}

func (qq *MainContext) UnsafeMainDB() *sqlx.MainDB {
	return qq.unsafeMainDB
}

// TODO remove one of those two methods
func (qq *MainContext) UnsafeTenantDB(tenantID int64) (*sqlx.TenantDB, bool) {
	return qq.unsafeTenantDBs.Load(tenantID)
}
func (qq *MainContext) UnsafeTenantDBs() *tenantdbs.TenantDBs {
	return qq.unsafeTenantDBs
}

// ReadOnlyAccountSpacesByTenant reuses the transaction of the tenant requestCtx belongs to, if
// any. Opening a second read transaction on a tenant DB the request already holds one on can
// exhaust the bounded read pool, so concurrent page requests would wait on each other forever.
//
// TODO cache?
func (qq *MainContext) ReadOnlyAccountSpacesByTenant(
	requestCtx context.Context,
) (map[*entmain.Tenant][]*enttenant.Space, error) {
	var spacesByTenant = make(map[*entmain.Tenant][]*enttenant.Space)

	// similar code in DashboardCards
	tenants, err := qq.MainTx.Account.QueryTenants(qq.Account).All(qq)
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("failed to query tenants for account %d: %w", qq.Account.ID, err)
	}

	currentTenantCtx, hasCurrentTenant := TenantCtx(requestCtx)
	for _, tenantx := range tenants {
		if hasCurrentTenant && currentTenantCtx.Tenant.ID == tenantx.ID {
			spaces, err := currentTenantCtx.TTx.Space.Query().All(currentTenantCtx)
			if err != nil && !enttenant.IsNotFound(err) {
				log.Println("failed to query spaces for tenant", tenantx.ID, err)
				continue
			}
			spacesByTenant[tenantx] = spaces
			continue
		}

		spaces, err := qq.readOnlySpacesForTenant(tenantx)
		if err != nil {
			log.Println(err)
			continue
		}

		spacesByTenant[tenantx] = spaces
	}

	return spacesByTenant, nil
}

func (qq *MainContext) readOnlySpacesForTenant(tenantx *entmain.Tenant) ([]*enttenant.Space, error) {
	tenantDB, ok := qq.unsafeTenantDBs.Load(tenantx.ID)
	if !ok {
		return nil, fmt.Errorf("tenant db not found, tenant id was %d", tenantx.ID)
	}
	tenantTx, err := tenantDB.ReadOnlyConn.Tx(qq)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction for tenant %d: %w", tenantx.ID, err)
	}

	// necessary for permissions
	tenantCtx := NewTenantContext(qq, tenantTx, tenantx, true)

	// spaces = append(spaces, tenantDB.Space.Query().AllX(ctx)...)
	spaces, err := tenantTx.Space.Query().All(tenantCtx)
	if err != nil && !enttenant.IsNotFound(err) {
		qq.rollbackTenantTx(tenantTx, tenantx.ID)
		return nil, fmt.Errorf("failed to query spaces for tenant %d: %w", tenantx.ID, err)
	}

	// TODO not sure if necessary... may could also just use db directly or rollback if faster?
	// TODO is it a problem that spaces get used in calling function after the tx is committed?
	if err := tenantTx.Commit(); err != nil {
		qq.rollbackTenantTx(tenantTx, tenantx.ID)
		return nil, fmt.Errorf("failed to commit transaction for tenant %d: %w", tenantx.ID, err)
	}
	return spaces, nil
}

func (qq *MainContext) rollbackTenantTx(tenantTx *enttenant.Tx, tenantID int64) {
	if err := tenantTx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		log.Println("failed to rollback transaction for tenant", tenantID, err)
	}
}

func (qq *MainContext) MainCtx() *MainContext {
	return qq
}

func (qq *MainContext) IsReadOnlyTx() bool {
	return qq.isReadOnly
}

func (qq *MainContext) TenantCtx() *TenantContext {
	panic("context not available")
}

func (qq *MainContext) SpaceCtx() *SpaceContext {
	panic("context not available")
}

func (qq *MainContext) IsMainCtx() bool {
	return true
}

func (qq *MainContext) IsTenantCtx() bool {
	return false
}

func (qq *MainContext) IsSpaceCtx() bool {
	return false
}
