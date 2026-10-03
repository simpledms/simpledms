package execution

import (
	"log"
	"net/http"

	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/entmain"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/main/tenantaccess"
	"github.com/simpledms/simpledms/util/e"
)

// ScopeResolver constructs privacy-scoped contexts in the caller's transactions.
type ScopeResolver struct{}

func NewScopeResolver() *ScopeResolver {
	return &ScopeResolver{}
}

func (qq *ScopeResolver) Resolve(
	mainCtx *ctxx.MainContext,
	tenantTx *enttenant.Tx,
	tenantx *entmain.Tenant,
	spaceID string,
	isReadOnly bool,
) (ctxx.Context, error) {
	hasAccess, err := tenantaccess.NewTenantAccessService().HasActiveTenantAssignmentForTenant(
		mainCtx, mainCtx.MainTx, mainCtx.Account.ID, tenantx.ID,
	)
	if err != nil {
		log.Println(err)
		return mainCtx, err
	}
	if !hasAccess {
		return mainCtx, e.NewHTTPErrorf(http.StatusForbidden, "You are not allowed to access this organization.")
	}
	userx, err := tenantTx.User.Query().Where(
		user.AccountID(mainCtx.Account.ID), user.DeletedAtIsNil(),
	).Only(mainCtx)
	if err != nil {
		log.Println(err)
		if enttenant.IsNotFound(err) {
			return mainCtx, e.NewHTTPErrorf(http.StatusForbidden, "You are not allowed to access this organization.")
		}
		return mainCtx, err
	}
	tenantCtx := ctxx.NewTenantContextWithUser(mainCtx, tenantTx, tenantx, userx, isReadOnly)
	if spaceID == "" {
		return tenantCtx, nil
	}
	// Space.Policy supplies membership filtering and tenant-owner implicit access.
	spacex, err := tenantTx.Space.Query().Where(space.PublicID(entx.NewCIText(spaceID))).Only(tenantCtx)
	if err != nil {
		log.Println(err)
		if enttenant.IsNotFound(err) {
			return tenantCtx, e.NewHTTPErrorf(http.StatusForbidden, "You are not allowed to access this Space.")
		}
		return tenantCtx, err
	}
	return ctxx.NewSpaceContext(tenantCtx, spacex), nil
}
