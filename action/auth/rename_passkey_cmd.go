package auth

import (
	"net/http"
	"strings"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	account2 "github.com/simpledms/simpledms/model/account"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type RenamePasskeyCmdData struct {
	PasskeyID string `validate:"required" form_attr_type:"hidden"`
	Name      string `validate:"required" form_attrs:"autofocus"`
}

type RenamePasskeyCmd struct {
	infra          *common.Infra
	actions        *Actions
	passkeyService *account2.PasskeyService
	*actionx.Config
	*autil.FormHelper[RenamePasskeyCmdData]
}

func NewRenamePasskeyCmd(
	infra *common.Infra,
	actions *Actions,
	passkeyService *account2.PasskeyService,
) *RenamePasskeyCmd {
	config := actionx.NewConfig(actions.Route("rename-passkey-cmd"), false)
	return &RenamePasskeyCmd{
		infra:          infra,
		actions:        actions,
		passkeyService: passkeyService,
		Config:         config,
		FormHelper: autil.NewFormHelperX[RenamePasskeyCmdData](
			infra,
			config,
			wx.T("Rename passkey"),
			wx.T("Rename"),
		),
	}
}

func (qq *RenamePasskeyCmd) Data(passkeyID, name string) *RenamePasskeyCmdData {
	return &RenamePasskeyCmdData{
		PasskeyID: passkeyID,
		Name:      name,
	}
}

func (qq *RenamePasskeyCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	mainCtx, err := qq.actions.RequireMainCtx(ctx, "You must be logged in to manage passkeys.")
	if err != nil {
		return err
	}

	data, err := autil.FormData[RenamePasskeyCmdData](rw, req, mainCtx)
	if err != nil {
		return err
	}

	name := strings.TrimSpace(data.Name)
	if name == "" {
		return e.NewHTTPErrorf(http.StatusBadRequest, "Passkey name is required.")
	}

	credentialx, err := qq.passkeyService.OwnCredentialByPublicID(mainCtx, mainCtx.Account.ID, data.PasskeyID)
	if err != nil {
		return err
	}

	credentialx.Update().SetName(name).SaveX(mainCtx)

	rw.AddRenderables(wx.NewSnackbarf("Passkey renamed."))
	rw.Header().Set("HX-Trigger", event.AccountUpdated.String())

	return nil
}
