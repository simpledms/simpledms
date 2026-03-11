package managetenantusers

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type Actions struct {
	ManageUsersOfTenantPage *ManageUsersOfTenantPage
	UserListPartial         *UserListPartial
	CreateUserCmd           *CreateUserCmd
	DeleteUserCmd           *DeleteUserCmd
}

func NewActions(infra *common.Infra) *Actions {
	actions := new(Actions)

	*actions = Actions{
		ManageUsersOfTenantPage: NewManageUsersOfTenantPage(infra, actions),
		UserListPartial:         NewUserListPartial(infra, actions),
		CreateUserCmd:           NewCreateUserCmd(infra, actions),
		DeleteUserCmd:           NewDeleteUserCmd(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.ManageUsersOfTenantActionsRoute() + path
}
