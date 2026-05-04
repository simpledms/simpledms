package dashboard

import (
	"github.com/simpledms/simpledms/action/admin"
	"github.com/simpledms/simpledms/action/auth"
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type Actions struct {
	Common       *acommon.Actions
	AuthActions  *auth.Actions
	AdminActions *admin.Actions

	DashboardPage                     *DashboardPage
	DashboardCardsPartial             *DashboardCardsPartial
	AccountPage                       *AccountPage
	AccountCardsPartial               *AccountCardsPartial
	SystemPage                        *SystemPage
	SystemCardsPartial                *SystemCardsPartial
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
		AccountPage:                       NewAccountPage(infra, actions),
		AccountCardsPartial:               NewAccountCardsPartial(infra, actions),
		SystemPage:                        NewSystemPage(infra, actions),
		SystemCardsPartial:                NewSystemCardsPartial(infra, actions),
		ToggleTenantPasskeyEnforcementCmd: NewToggleTenantPasskeyEnforcementCmd(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.DashboardActionsRoute() + path
}
