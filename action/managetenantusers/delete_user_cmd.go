package managetenantusers

import (
	"log"
	"net/http"
	"time"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	enttenantprivacy "github.com/simpledms/simpledms/db/enttenant/privacy"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/db/enttenant/user"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/model/common/tenantrole"
	"github.com/simpledms/simpledms/model/modelmain"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
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

func (qq *DeleteUserCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
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

	userx := ctx.TenantCtx().TTx.User.Query().
		Where(user.PublicID(entx.NewCIText(data.UserID))).
		OnlyX(ctx)

	if userx.AccountID == ctx.MainCtx().Account.ID {
		return e.NewHTTPErrorf(
			http.StatusConflict,
			"You cannot delete your own user in organization management.",
		)
	}

	// TODO why is this necessary? should tenant owner not be allowed to do this?
	ctxWithPrivacyOverride := enttenantprivacy.DecisionContext(ctx, enttenantprivacy.Allow)

	ctx.TenantCtx().TTx.SpaceUserAssignment.Delete().
		Where(spaceuserassignment.UserID(userx.ID)).
		ExecX(ctxWithPrivacyOverride)

	ctx.TenantCtx().TTx.User.UpdateOneID(userx.ID).
		SetDeletedAt(time.Now()).
		SetDeletedBy(ctx.TenantCtx().User.ID).
		ExecX(ctx)

	result, err := modelmain.NewTenantAccountLifecycleService().RemoveAccountFromTenant(
		ctx,
		ctx.TenantCtx().Tenant.ID,
		userx.AccountID,
	)
	if err != nil {
		log.Println(err)
		return err
	}

	if result.IsOwningTenantAssignment {
		rw.AddRenderables(wx.NewSnackbarf("User removed from organization and account deleted globally."))
	} else {
		rw.AddRenderables(wx.NewSnackbarf("User removed from organization."))
	}
	rw.Header().Set("HX-Trigger", event.UserDeleted.String())

	return nil
}
