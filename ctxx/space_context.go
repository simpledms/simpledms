package ctxx

// ctxx instead of ctx and context prevents naming conflicts with var names and on import

import (
	"context"

	"github.com/simpledms/simpledms/enttenant"
	"github.com/simpledms/simpledms/enttenant/file"
	"github.com/simpledms/simpledms/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/model/common/spacerole"
	"github.com/simpledms/simpledms/model/common/tenantrole"
)

// having TTx in Context allows for easier replacement of ent with jet later
// TODO add a cache for database entities?
type SpaceContext struct {
	// ResponseWriter httpx.ResponseWriter
	// Request        *httpx.Request
	// Infra        *common.Infra // TODO is this a good idea?
	*TenantContext
	// context.Context
	// MainTx       *entmain.Tx
	// Account      *entmain.Account // modelmain.Account would be better, but leads to circular dependency
	// TTx          *enttenant.Tx
	// TenantID     string
	// Tenant       *entmain.Tenant  // see Account
	// StoragePath  string           // TODO belongs to context?
	SpaceID             string           // Unsafe because only set for Get requests via router..., not for commands
	Space               *enttenant.Space // TODO rename to NilableSpace?; model.Space?
	nilableSpaceRootDir *enttenant.File  // TODO is ID enough? read on demand... but query is necessary anyway...
}

func NewSpaceContext(
	tenantContext *TenantContext,
	space *enttenant.Space,
) *SpaceContext {
	spaceCtx := &SpaceContext{
		// TenantContext: NewTenantContext(ctx, mainTx, tenantTx, account, tenant, metaPath),
		TenantContext: tenantContext,
		SpaceID:       space.PublicID.String(),
		Space:         space,
	}
	spaceCtx.Context = context.WithValue(tenantContext.Context, spaceCtxKey, spaceCtx)
	return spaceCtx
}

func (qq *SpaceContext) SpaceRootDir() *enttenant.File {
	if qq.nilableSpaceRootDir == nil {
		// not in constructor because that violates the space privacy rules
		// because SpaceCtx is not initialized when query is executed
		// in this case
		qq.nilableSpaceRootDir = qq.Space.QueryFiles().Where(
			file.IsRootDir(true),
		).OnlyX(qq)
	}
	return qq.nilableSpaceRootDir
}

// TODO name?
func (qq *SpaceContext) UserRoleInSpace() spacerole.SpaceRole {
	// TODO not a nice way to handle this...
	if qq.User.Role == tenantrole.Owner {
		return spacerole.Owner
	}
	return qq.Space.QueryUserAssignment().
		Where(spaceuserassignment.UserID(qq.User.ID)).
		OnlyX(qq).
		Role
}

func (qq *SpaceContext) SpaceCtx() *SpaceContext {
	return qq
}

func (qq *SpaceContext) IsSpaceCtx() bool {
	return true
}
