package auth

import (
	"net/http"
	"strings"

	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	account2 "github.com/simpledms/simpledms/core/model/account"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
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
			widget.T("Rename passkey"),
			widget.T("Rename"),
		),
	}
}

func (qq *RenamePasskeyCmd) Data(passkeyID, name string) *RenamePasskeyCmdData {
	return &RenamePasskeyCmdData{
		PasskeyID: passkeyID,
		Name:      name,
	}
}

func (qq *RenamePasskeyCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

	rw.AddRenderables(widget.NewSnackbarf("Passkey renamed."))
	rw.Header().Set("HX-Trigger", events.AccountUpdated.String())

	return nil
}
