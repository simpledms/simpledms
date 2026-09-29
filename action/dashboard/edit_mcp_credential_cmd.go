package dashboard

import (
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/mcpcredential"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type EditMCPCredentialCmd struct {
	infra       *common.Infra
	actions     *Actions
	credentialx *mcpcredential.CredentialService
	*actionx.Config
	*autil.FormHelper[EditMCPCredentialCmdData]
}

func NewEditMCPCredentialCmd(
	infra *common.Infra,
	actions *Actions,
) *EditMCPCredentialCmd {
	config := actionx.NewConfig(actions.Route("edit-mcp-credential-cmd"), false)
	return &EditMCPCredentialCmd{
		infra:       infra,
		actions:     actions,
		credentialx: mcpcredential.NewCredentialService(),
		Config:      config,
		FormHelper: autil.NewFormHelper[EditMCPCredentialCmdData](
			infra,
			config,
			widget.T("Edit"),
		),
	}
}

func (qq *EditMCPCredentialCmd) Data(
	credentialPublicID string,
	clientLabel string,
) *EditMCPCredentialCmdData {
	return &EditMCPCredentialCmdData{
		CredentialPublicID: credentialPublicID,
		ClientLabel:        clientLabel,
	}
}

func (qq *EditMCPCredentialCmd) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	if ctx.VisitorCtx().IsTemporarySession {
		return e.NewHTTPErrorf(http.StatusForbidden, "A full Session is required.")
	}
	data, err := autil.FormData[EditMCPCredentialCmdData](rw, req, ctx)
	if err != nil {
		return err
	}
	if err := qq.credentialx.EditLabel(
		ctx.MainCtx(),
		data.CredentialPublicID,
		data.ClientLabel,
	); err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.AccountUpdated.String())
	rw.AddRenderables(widget.NewSnackbarf("Changes saved."))
	return nil
}
