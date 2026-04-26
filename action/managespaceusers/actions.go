package managespaceusers

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type Actions struct {
	ManageUsersOfSpacePage *ManageUsersOfSpacePage

	UsersOfSpaceListPartial *UsersOfSpaceListPartial

	AssignUserToSpaceCmd     *AssignUserToSpaceCmd
	UnassignUserFromSpaceCmd *UnassignUserFromSpaceCmd
}

func NewActions(infra *common.Infra) *Actions {
	actions := new(Actions)

	*actions = Actions{
		ManageUsersOfSpacePage:   NewManageUsersOfSpace(infra, actions),
		UsersOfSpaceListPartial:  NewUsersOfSpaceListPartial(infra, actions),
		AssignUserToSpaceCmd:     NewAssignUserToSpaceCmd(infra, actions),
		UnassignUserFromSpaceCmd: NewUnassignUserFromSpaceCmd(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.ManageUsersOfSpaceActionsRoute() + path
}
