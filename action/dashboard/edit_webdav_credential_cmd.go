package dashboard

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/webdavcredential"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type EditWebDAVCredentialCmd struct {
	infra       *common.Infra
	actions     *Actions
	credentialx *webdavcredential.CredentialService
	*actionx.Config
	*autil.FormHelper[EditWebDAVCredentialCmdData]
}

func NewEditWebDAVCredentialCmd(
	infra *common.Infra,
	actions *Actions,
) *EditWebDAVCredentialCmd {
	config := actionx.NewConfig(actions.Route("edit-webdav-credential-cmd"), false)
	return &EditWebDAVCredentialCmd{
		infra:       infra,
		actions:     actions,
		credentialx: webdavcredential.NewCredentialService(),
		Config:      config,
		FormHelper: autil.NewFormHelper[EditWebDAVCredentialCmdData](
			infra,
			config,
			widget.T("Edit"),
		),
	}
}

func (qq *EditWebDAVCredentialCmd) Data(
	credentialPublicID string,
	deviceLabel string,
) *EditWebDAVCredentialCmdData {
	return &EditWebDAVCredentialCmdData{
		CredentialPublicID: credentialPublicID,
		DeviceLabel:        deviceLabel,
	}
}

func (qq *EditWebDAVCredentialCmd) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[EditWebDAVCredentialCmdData](rw, req, ctx)
	if err != nil {
		return err
	}
	if err := qq.credentialx.EditOwnedCredentialLabel(
		ctx.MainCtx(),
		data.CredentialPublicID,
		data.DeviceLabel,
	); err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.AccountUpdated.String())
	rw.AddRenderables(widget.NewSnackbarf("Changes saved."))
	return nil
}
