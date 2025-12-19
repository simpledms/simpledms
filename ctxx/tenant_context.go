package ctxx

// ctxx instead of ctx and context prevents naming conflicts with var names and on import

import (
	"context"

	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/user"
)

type TenantContext struct {
	*MainContext
	// context.Context
	// MainTx      *entmain.Tx
	// Account     *entmain.Account // modelmain.Account would be better, but leads to circular dependency
	TTx      *enttenant.Tx
	TenantID string
	Tenant   *entmain.Tenant // see comment on Account
	User     *enttenant.User
	// to dangerous because of newly introduced tmpStoragePrefix
	// StoragePath     string // TODO belongs to context?
	// S3StoragePrefix string
}

func NewTenantContext(
	mainContext *MainContext,
	tenantTx *enttenant.Tx,
	tenant *entmain.Tenant,
) *TenantContext {
	tenantID := tenant.PublicID.String()

	userx := tenantTx.User.Query().
		Where(user.AccountID(mainContext.Account.ID)).
		OnlyX(mainContext)

	// TODO add a cache for database entities?
	tenantCtx := &TenantContext{
		MainContext: mainContext,
		TTx:         tenantTx,
		TenantID:    tenantID,
		Tenant:      tenant,
		User:        userx,
	}
	tenantCtx.Context = context.WithValue(mainContext.Context, tenantCtxKey, tenantCtx)
	return tenantCtx
}

func (qq *TenantContext) TenantCtx() *TenantContext {
	return qq
}

func (qq *TenantContext) SpaceCtx() *SpaceContext {
	panic("context not available")
}

func (qq *TenantContext) IsTenantCtx() bool {
	return true
}

func (qq *TenantContext) IsSpaceCtx() bool {
	return false
}
