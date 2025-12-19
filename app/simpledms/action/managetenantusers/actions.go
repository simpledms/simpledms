package managetenantusers

import (
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ui/route"
)

type Actions struct {
	ManageUsersOfTenantPage *ManageUsersOfTenantPage
	UserListPartial         *UserListPartial
	CreateUserCmd           *CreateUserCmd
}

func NewActions(infra *common.Infra) *Actions {
	actions := new(Actions)

	*actions = Actions{
		ManageUsersOfTenantPage: NewManageUsersOfTenantPage(infra, actions),
		UserListPartial:         NewUserListPartial(infra, actions),
		CreateUserCmd:           NewCreateUserCmd(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.ManageUsersOfTenantActionsRoute() + path
}
