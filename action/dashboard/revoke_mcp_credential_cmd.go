package dashboard

import (
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/mcpcredential"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type RevokeMCPCredentialCmd struct {
	service *mcpcredential.CredentialService
	*actionx.Config
}

func NewRevokeMCPCredentialCmd(_ *common.Infra, actions *Actions) *RevokeMCPCredentialCmd {
	return &RevokeMCPCredentialCmd{
		service: mcpcredential.NewCredentialService(),
		Config: actionx.NewConfig(actions.Route("revoke-mcp-credential-cmd"), false).
			EnableCommittedResponse(),
	}
}

func (qq *RevokeMCPCredentialCmd) Handler(
	rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context,
) error {
	if ctx.VisitorCtx().IsTemporarySession {
		return e.NewHTTPErrorf(http.StatusForbidden, "A full Session is required.")
	}
	data, err := autil.FormData[RevokeMCPCredentialCmdData](rw, req, ctx)
	if err != nil {
		return err
	}
	if _, err := qq.service.Revoke(ctx.MainCtx(), data.CredentialID); err != nil {
		return err
	}
	rw.Header().Set("Cache-Control", "no-store")
	rw.Header().Set("HX-Trigger", "mcpCredentialsChanged")
	rw.AddRenderables(widget.NewSnackbarf("MCP credential revoked."))
	return nil
}
