package managetenantusers

import (
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/model/common/tenantrole"
	tenantusermodel "github.com/simpledms/simpledms/core/model/tenantuser"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	"github.com/simpledms/simpledms/core/util/e"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
)

type DeleteUserCmdData struct {
	UserID string `validate:"required"`
}

type DeleteUserCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDeleteUserCmd(infra *common.Infra, actions *Actions) *DeleteUserCmd {
	config := actionx.NewConfig(actions.Route("delete-user-cmd"), false)
	return &DeleteUserCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *DeleteUserCmd) Data(userID string) *DeleteUserCmdData {
	return &DeleteUserCmdData{
		UserID: userID,
	}
}

func (qq *DeleteUserCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	if !ctx.IsTenantCtx() {
		return e.NewHTTPErrorf(http.StatusBadRequest, "You are not allowed to delete users. No organization selected.")
	}
	if ctx.TenantCtx().User.Role != tenantrole.Owner {
		return e.NewHTTPErrorf(http.StatusForbidden, "You are not allowed to delete users because you are not the owner.")
	}

	data, err := autil.FormData[DeleteUserCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	result, err := tenantusermodel.Delete(
		ctx,
		ctx.TenantCtx().Tenant.ID,
		data.UserID,
		ctx.MainCtx().Account.ID,
		ctx.TenantCtx().User.ID,
	)
	if err != nil {
		return err
	}

	if result.IsOwningTenantAssignment {
		rw.AddRenderables(wx.NewSnackbarf("User removed from organization and account deleted globally."))
	} else {
		rw.AddRenderables(wx.NewSnackbarf("User removed from organization."))
	}
	rw.Header().Set("HX-Trigger", events.UserDeleted.String())

	return nil
}
