package dashboard

import (
	"github.com/marcobeierer/go-core/action/admin"
	"github.com/marcobeierer/go-core/action/auth"
	acommon "github.com/marcobeierer/go-core/action/common"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ui/uix/route"
)

type Actions struct {
	Common       *acommon.Actions
	AuthActions  *auth.Actions
	AdminActions *admin.Actions

	DashboardPage                     *DashboardPage
	DashboardCardsPartial             *DashboardCardsPartial
	ToggleTenantPasskeyEnforcementCmd *ToggleTenantPasskeyEnforcementCmd
}

func NewActions(
	infra *common.Infra,
	commonActions *acommon.Actions,
	authActions *auth.Actions,
	adminActions *admin.Actions,
) *Actions {
	actions := new(Actions)
	*actions = Actions{
		Common:       commonActions,
		AuthActions:  authActions,
		AdminActions: adminActions,

		DashboardPage:                     NewDashboardPage(infra, actions),
		DashboardCardsPartial:             NewDashboardCardsPartial(infra, actions),
		ToggleTenantPasskeyEnforcementCmd: NewToggleTenantPasskeyEnforcementCmd(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.DashboardActionsRoute() + path
}
