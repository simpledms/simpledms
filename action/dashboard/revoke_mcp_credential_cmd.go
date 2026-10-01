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

type RevokeMCPCredentialCmd struct {
	infra       *common.Infra
	actions     *Actions
	credentialx *mcpcredential.CredentialService
	*actionx.Config
}

func NewRevokeMCPCredentialCmd(infra *common.Infra, actions *Actions) *RevokeMCPCredentialCmd {
	config := actionx.NewConfig(actions.Route("revoke-mcp-credential-cmd"), false).
		EnableCommittedResponse()
	return &RevokeMCPCredentialCmd{
		infra:       infra,
		actions:     actions,
		credentialx: mcpcredential.NewCredentialService(),
		Config:      config,
	}
}

func (qq *RevokeMCPCredentialCmd) Data(credentialPublicID string) *RevokeMCPCredentialCmdData {
	return &RevokeMCPCredentialCmdData{
		CredentialPublicID: credentialPublicID,
	}
}

func (qq *RevokeMCPCredentialCmd) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	if ctx.VisitorCtx().IsTemporarySession {
		return e.NewHTTPErrorf(http.StatusForbidden, "A full Session is required.")
	}
	data, err := autil.FormData[RevokeMCPCredentialCmdData](rw, req, ctx)
	if err != nil {
		return err
	}
	if _, err := qq.credentialx.Revoke(ctx.MainCtx(), data.CredentialPublicID); err != nil {
		return err
	}

	rw.Header().Set("Cache-Control", "no-store")
	rw.Header().Set("HX-Trigger", event.AccountUpdated.String())
	rw.AddRenderables(widget.NewSnackbarf("MCP credential revoked."))
	return nil
}
