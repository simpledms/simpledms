package auth

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type Actions struct {
	SignInPage                *SignInPage
	SignInCmd                 *SignInCmd
	SignOutCmd                *SignOutCmd
	ResetPasswordCmd          *ResetPasswordCmd
	ChangePasswordCmd         *ChangePasswordCmd
	SetInitialPasswordCmd     *SetInitialPasswordCmd
	ClearTemporaryPasswordCmd *ClearTemporaryPasswordCmd
	EditAccountCmd            *EditAccountCmd
}

func NewActions(infra *common.Infra) *Actions {
	actions := new(Actions)

	*actions = Actions{
		SignInPage:                NewSignInPage(infra, actions),
		SignInCmd:                 NewSignInCmd(infra, actions),
		SignOutCmd:                NewSignOutCmd(infra, actions),
		ResetPasswordCmd:          NewResetPasswordCmd(infra, actions),
		ChangePasswordCmd:         NewChangePasswordCmd(infra, actions),
		SetInitialPasswordCmd:     NewSetInitialPasswordCmd(infra, actions),
		ClearTemporaryPasswordCmd: NewClearTemporaryPasswordCmd(infra, actions),
		EditAccountCmd:            NewEditAccountCmd(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.AuthActionsRoute() + path
}
