package ctxx

import (
	"context"
	"log"

	"github.com/simpledms/simpledms/common/tenantdbs"
	"github.com/simpledms/simpledms/entmain"
	"github.com/simpledms/simpledms/enttenant"
	"github.com/simpledms/simpledms/i18n"
)

type MainContext struct {
	*VisitorContext
	Account *entmain.Account // modelmain.Account would be better, but leads to circular dependency
	// should never be exposed directly;
	// unsafe because must be used with care
	unsafeTenantDBs *tenantdbs.TenantDBs
}

func NewMainContext(
	ctx *VisitorContext,
	account *entmain.Account,
	i18nx *i18n.I18n,
	tenantDBs *tenantdbs.TenantDBs,
) *MainContext {
	ctx.Printer = i18nx.Printer(account.Language.Tag())
	langTagBase, _ := account.Language.Tag().Base() // TODO evaluate confidence?
	ctx.LanguageBCP47 = langTagBase.String()

	mainCtx := &MainContext{
		VisitorContext:  ctx,
		Account:         account,
		unsafeTenantDBs: tenantDBs,
	}
	mainCtx.Context = context.WithValue(ctx.Context, mainCtxKey, mainCtx)
	return mainCtx
}

// TODO cache?
func (qq *MainContext) ReadOnlyAccountSpacesByTenant() map[*entmain.Tenant][]*enttenant.Space {
	var spacesByTenant = make(map[*entmain.Tenant][]*enttenant.Space)

	// similar code in DashboardCards
	tenants := qq.Account.QueryTenants().AllX(qq)
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
		tenantCtx := NewTenantContext(qq, tenantTx, tenantx)

		// spaces = append(spaces, tenantDB.Space.Query().AllX(ctx)...)
		spacesx, err := tenantDB.ReadOnlyConn.Space.Query().All(tenantCtx)
		if err != nil && !enttenant.IsNotFound(err) {
			log.Println("failed to query spaces for tenant", tenantx.ID, err)
			if err := tenantTx.Rollback(); err != nil {
				log.Println("failed to rollback transaction for tenant", tenantx.ID, err)
			}
			continue
		}
		spaces = append(spaces, spacesx...)

		// TODO not sure if necessary... may could also just use db directly or rollback if faster?
		// TODO is it a problem that spaces get used in calling function after the tx is committed?
		if err := tenantTx.Commit(); err != nil {
			log.Println("failed to commit transaction for tenant", tenantx.ID, err)
			if err := tenantTx.Rollback(); err != nil {
				log.Println("failed to rollback transaction for tenant", tenantx.ID, err)
			}
		}

		spacesByTenant[tenantx] = spaces
	}

	return spacesByTenant
}

func (qq *MainContext) MainCtx() *MainContext {
	return qq
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
