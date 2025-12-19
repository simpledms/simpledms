package dashboard

import (
	"github.com/simpledms/simpledms/action/admin"
	"github.com/simpledms/simpledms/action/auth"
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/uix/route"
)

type Actions struct {
	Common       *acommon.Actions
	AuthActions  *auth.Actions
	AdminActions *admin.Actions

	DashboardPage  *DashboardPage
	DashboardCards *DashboardCards
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

		DashboardPage:  NewDashboardPage(infra, actions),
		DashboardCards: NewDashboardCards(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.DashboardActionsRoute() + path
}
