package auth

import (
	"net/http"

	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	account2 "github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/util/e"
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

	PasskeySignInBeginCmd      *PasskeySignInBeginCmd
	PasskeySignInFinishCmd     *PasskeySignInFinishCmd
	PasskeyRegisterDialog      *PasskeyRegisterDialog
	PasskeyRegisterBeginCmd    *PasskeyRegisterBeginCmd
	PasskeyRegisterFinishCmd   *PasskeyRegisterFinishCmd
	PasskeyRecoveryCodesDialog *PasskeyRecoveryCodesDialog
	RenamePasskeyCmd           *RenamePasskeyCmd
	DeletePasskeyCmd           *DeletePasskeyCmd
	RegeneratePasskeyCodesCmd  *RegeneratePasskeyCodesCmd
	PasskeyRecoverySignInCmd   *PasskeyRecoverySignInCmd
	ClearPasskeysCmd           *ClearPasskeysCmd
	AdminPasskeyRecoveryCmd    *AdminPasskeyRecoveryCmd
}

func NewActions(infra *common.Infra) *Actions {
	recoveryCodesStore := account2.NewPasskeyRecoveryCodesStore()
	requestRateLimiter := account2.NewRequestRateLimiter()
	passkeyService := account2.NewPasskeyService(
		infra.SystemConfig().PublicOrigin(),
		infra.SystemConfig().WebAuthnRPID(),
		infra.SystemConfig().WebAuthnRPName(),
	)
	actions := &Actions{}

	*actions = Actions{
		SignInPage:                NewSignInPage(infra, actions),
		SignInCmd:                 NewSignInCmd(infra, actions, requestRateLimiter),
		SignOutCmd:                NewSignOutCmd(infra, actions),
		ResetPasswordCmd:          NewResetPasswordCmd(infra, actions, requestRateLimiter),
		ChangePasswordCmd:         NewChangePasswordCmd(infra, actions),
		SetInitialPasswordCmd:     NewSetInitialPasswordCmd(infra, actions),
		ClearTemporaryPasswordCmd: NewClearTemporaryPasswordCmd(infra, actions),
		EditAccountCmd:            NewEditAccountCmd(infra, actions),

		PasskeySignInBeginCmd:      NewPasskeySignInBeginCmd(infra, actions, passkeyService, requestRateLimiter),
		PasskeySignInFinishCmd:     NewPasskeySignInFinishCmd(infra, actions, passkeyService),
		PasskeyRegisterDialog:      NewPasskeyRegisterDialog(infra, actions),
		PasskeyRegisterBeginCmd:    NewPasskeyRegisterBeginCmd(infra, actions, passkeyService),
		PasskeyRegisterFinishCmd:   NewPasskeyRegisterFinishCmd(infra, actions, passkeyService, recoveryCodesStore),
		PasskeyRecoveryCodesDialog: NewPasskeyRecoveryCodesDialog(infra, actions, recoveryCodesStore),
		RenamePasskeyCmd:           NewRenamePasskeyCmd(infra, actions, passkeyService),
		DeletePasskeyCmd:           NewDeletePasskeyCmd(infra, actions, passkeyService),
		RegeneratePasskeyCodesCmd:  NewRegeneratePasskeyCodesCmd(infra, actions, passkeyService, recoveryCodesStore),
		PasskeyRecoverySignInCmd:   NewPasskeyRecoverySignInCmd(infra, actions, passkeyService),
		ClearPasskeysCmd:           NewClearPasskeysCmd(infra, actions),
		AdminPasskeyRecoveryCmd:    NewAdminPasskeyRecoveryCmd(infra, actions, passkeyService),
	}

	return actions
}

func (qq *Actions) Route(path string) string {
	return route.AuthActionsRoute() + path
}

func (qq *Actions) RequireMainCtx(
	ctx ctxx.Context,
	unauthorizedMessage string,
) (*ctxx.MainContext, error) {
	if ctx.IsMainCtx() {
		return ctx.MainCtx(), nil
	}

	return nil, e.NewHTTPErrorf(http.StatusUnauthorized, unauthorizedMessage)
}

func (qq *Actions) RequireMainRole(
	mainCtx *ctxx.MainContext,
	forbiddenMessage string,
	roles ...mainrole.MainRole,
) error {
	for _, role := range roles {
		if mainCtx.Account.Role == role {
			return nil
		}
	}

	return e.NewHTTPErrorf(http.StatusForbidden, forbiddenMessage)
}
