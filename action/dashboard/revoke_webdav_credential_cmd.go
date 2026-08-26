package dashboard

import (
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/model/main/common/tenantrole"
	"github.com/simpledms/simpledms/model/main/webdavcredential"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type RevokeWebDAVCredentialCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewRevokeWebDAVCredentialCmd(
	infra *common.Infra,
	actions *Actions,
) *RevokeWebDAVCredentialCmd {
	config := actionx.NewConfig(actions.Route("revoke-webdav-credential-cmd"), false)
	return &RevokeWebDAVCredentialCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *RevokeWebDAVCredentialCmd) Data(
	credentialPublicID string,
	tenantID int64,
) *RevokeWebDAVCredentialCmdData {
	return &RevokeWebDAVCredentialCmdData{
		CredentialPublicID: credentialPublicID,
		TenantID:           tenantID,
	}
}

func (qq *RevokeWebDAVCredentialCmd) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[RevokeWebDAVCredentialCmdData](rw, req, ctx)
	if err != nil {
		return err
	}
	service := webdavcredential.NewCredentialService()
	if data.TenantID == 0 {
		err = service.RevokeOwnedCredential(ctx.MainCtx(), data.CredentialPublicID)
	} else {
		if !ctx.IsTenantCtx() || ctx.TenantCtx().Tenant.ID != data.TenantID ||
			ctx.TenantCtx().User.Role != tenantrole.Owner {
			return e.NewHTTPErrorf(http.StatusForbidden, "You are not allowed to revoke this credential.")
		}
		err = service.RevokeTenantCredential(
			ctx.MainCtx(),
			data.CredentialPublicID,
			data.TenantID,
		)
	}
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.AccountUpdated.String())
	rw.AddRenderables(widget.NewSnackbarf("WebDAV credential revoked."))
	return nil
}
