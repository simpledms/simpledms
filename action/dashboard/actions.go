package dashboard

import (
	"github.com/simpledms/simpledms/action/admin"
	"github.com/simpledms/simpledms/action/auth"
	acommon "github.com/simpledms/simpledms/action/common"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/common/tenantdbs"
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
	WebDAVCredentialsPage             *WebDAVCredentialsPage
	WebDAVCredentialListPartial       *WebDAVCredentialListPartial
	WebDAVCredentialFilterDialog      *WebDAVCredentialFilterDialog
	SystemPage                        *SystemPage
	SystemCardsPartial                *SystemCardsPartial
	OrganizationSettingsPage          *OrganizationSettingsPage
	ToggleTenantPasskeyEnforcementCmd *ToggleTenantPasskeyEnforcementCmd
	CreateWebDAVCredentialCmd         *CreateWebDAVCredentialCmd
	EditWebDAVCredentialCmd           *EditWebDAVCredentialCmd
	RevokeWebDAVCredentialCmd         *RevokeWebDAVCredentialCmd
}

func NewActions(
	infra *common.Infra,
	tenantDBs *tenantdbs.TenantDBs,
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
		WebDAVCredentialsPage:             NewWebDAVCredentialsPage(infra, actions),
		WebDAVCredentialListPartial:       NewWebDAVCredentialListPartial(infra, actions),
		WebDAVCredentialFilterDialog:      NewWebDAVCredentialFilterDialog(infra, actions),
		SystemPage:                        NewSystemPage(infra, actions),
		SystemCardsPartial:                NewSystemCardsPartial(infra, actions),
		OrganizationSettingsPage:          NewOrganizationSettingsPage(infra, actions),
		ToggleTenantPasskeyEnforcementCmd: NewToggleTenantPasskeyEnforcementCmd(infra, actions),
		CreateWebDAVCredentialCmd:         NewCreateWebDAVCredentialCmd(infra, tenantDBs, actions),
		EditWebDAVCredentialCmd:           NewEditWebDAVCredentialCmd(infra, actions),
		RevokeWebDAVCredentialCmd:         NewRevokeWebDAVCredentialCmd(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.DashboardActionsRoute() + path
}
