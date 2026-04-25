package dashboard

import (
	"net/http"

	"github.com/simpledms/simpledms/core/db/entmain"
	"github.com/simpledms/simpledms/core/db/entmain/tenant"
	"github.com/simpledms/simpledms/core/db/entmain/tenantaccountassignment"
	"github.com/simpledms/simpledms/core/db/entx"

	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	account2 "github.com/simpledms/simpledms/core/model/account"
	tenant2 "github.com/simpledms/simpledms/core/model/tenant"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type ToggleTenantPasskeyEnforcementCmdData struct {
	TenantID        string `validate:"required" form_attr_type:"hidden"`
	EnforcePasskeys bool   `form_attr_type:"hidden"`
}

type ToggleTenantPasskeyEnforcementCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewToggleTenantPasskeyEnforcementCmd(
	infra *common.Infra,
	actions *Actions,
) *ToggleTenantPasskeyEnforcementCmd {
	config := actionx.NewConfig(actions.Route("toggle-tenant-passkey-enforcement-cmd"), false)
	return &ToggleTenantPasskeyEnforcementCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *ToggleTenantPasskeyEnforcementCmd) Data(
	tenantPublicID string,
	enforcePasskeys bool,
) *ToggleTenantPasskeyEnforcementCmdData {
	return &ToggleTenantPasskeyEnforcementCmdData{
		TenantID:        tenantPublicID,
		EnforcePasskeys: enforcePasskeys,
	}
}

func (qq *ToggleTenantPasskeyEnforcementCmd) Handler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	mainCtx, err := qq.actions.AuthActions.RequireMainCtx(ctx, "You must be logged in to manage organizations.")
	if err != nil {
		return err
	}

	data, err := autil.FormData[ToggleTenantPasskeyEnforcementCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	tenantx, err := mainCtx.MainTx.Tenant.Query().
		Where(
			tenant.PublicID(entx.NewCIText(data.TenantID)),
			tenant.HasAccountAssignmentWith(
				tenantaccountassignment.AccountID(mainCtx.Account.ID),
			),
		).
		Only(mainCtx)
	if err != nil {
		if entmain.IsNotFound(err) {
			return e.NewHTTPErrorf(http.StatusNotFound, "Organization not found.")
		}

		return e.NewHTTPErrorf(http.StatusNotFound, "Organization not found.")
	}

	accountm := account2.NewAccount(mainCtx.Account)
	tenantm := tenant2.NewTenant(tenantx)
	if !tenantm.IsOwner(accountm) {
		return e.NewHTTPErrorf(http.StatusForbidden, "Only owners can change passkey enforcement.")
	}

	mainCtx.MainTx.Tenant.UpdateOneID(tenantx.ID).
		SetPasskeyAuthEnforced(data.EnforcePasskeys).
		SaveX(mainCtx)

	rw.Header().Set("HX-Trigger", events.AccountUpdated.String())
	if data.EnforcePasskeys {
		rw.AddRenderables(wx.NewSnackbarf("Passkey enforcement enabled for organization."))
	} else {
		rw.AddRenderables(wx.NewSnackbarf("Passkey enforcement disabled for organization."))
	}

	return nil
}
