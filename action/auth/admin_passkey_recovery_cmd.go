package auth

import (
	"log"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	account2 "github.com/simpledms/simpledms/model/main/account"
	"github.com/simpledms/simpledms/model/main/common/mainrole"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type AdminPasskeyRecoveryCmdData struct {
	Email string `validate:"required,email"`
}

type AdminPasskeyRecoveryCmd struct {
	infra          *common.Infra
	actions        *Actions
	passkeyService *account2.PasskeyService
	*actionx.Config
}

func NewAdminPasskeyRecoveryCmd(
	infra *common.Infra,
	actions *Actions,
	passkeyService *account2.PasskeyService,
) *AdminPasskeyRecoveryCmd {
	config := actionx.NewConfig(actions.Route("admin-passkey-recovery-cmd"), false)
	return &AdminPasskeyRecoveryCmd{
		infra:          infra,
		actions:        actions,
		passkeyService: passkeyService,
		Config:         config,
	}
}

func (qq *AdminPasskeyRecoveryCmd) Data(email string) *AdminPasskeyRecoveryCmdData {
	return &AdminPasskeyRecoveryCmdData{
		Email: email,
	}
}

func (qq *AdminPasskeyRecoveryCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	mainCtx, err := qq.actions.RequireMainCtx(ctx, "You must be logged in to perform this action.")
	if err != nil {
		return err
	}

	err = qq.actions.RequireMainRole(
		mainCtx,
		"Only admins and supporters can run assisted passkey recovery.",
		mainrole.Admin,
		mainrole.Supporter,
	)
	if err != nil {
		return err
	}

	data, err := autil.FormData[AdminPasskeyRecoveryCmdData](rw, req, mainCtx)
	if err != nil {
		return err
	}

	targetAccountx, recoveryCodes, err := qq.passkeyService.AssistedRecoveryCodesForEmail(mainCtx, data.Email, 10)
	if err != nil {
		return err
	}

	log.Printf(
		"admin assisted passkey recovery actor_account_id=%d target_account_id=%d",
		mainCtx.Account.ID,
		targetAccountx.ID,
	)

	return writeJSONResponse(rw, http.StatusOK, struct {
		RecoveryCodes []string `json:"recoveryCodes"`
	}{
		RecoveryCodes: recoveryCodes,
	})
}
