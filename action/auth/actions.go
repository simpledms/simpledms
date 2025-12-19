package auth

import (
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type Actions struct {
	SignInPage             *SignInPage
	SignUp                 *SignUp
	SignIn                 *SignIn
	SignOut                *SignOut
	ResetPassword          *ResetPassword
	ChangePassword         *ChangePassword
	SetInitialPassword     *SetInitialPassword
	ClearTemporaryPassword *ClearTemporaryPassword
	DeleteAccountCmd       *DeleteAccountCmd
	EditAccountCmd         *EditAccountCmd
}

func NewActions(infra *common.Infra) *Actions {
	actions := new(Actions)

	*actions = Actions{
		SignInPage:             NewSignInPage(infra, actions),
		SignUp:                 NewSignUp(infra, actions),
		SignIn:                 NewSignIn(infra, actions),
		SignOut:                NewSignOut(infra, actions),
		ResetPassword:          NewResetPassword(infra, actions),
		ChangePassword:         NewChangePassword(infra, actions),
		SetInitialPassword:     NewSetInitialPassword(infra, actions),
		ClearTemporaryPassword: NewClearTemporaryPassword(infra, actions),
		DeleteAccountCmd:       NewDeleteAccountCmd(infra, actions),
		EditAccountCmd:         NewEditAccountCmd(infra, actions),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.AuthActionsRoute() + path
}
