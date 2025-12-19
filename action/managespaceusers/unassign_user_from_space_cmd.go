package managespaceusers

import (
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/spaceuserassignment"
	"github.com/simpledms/simpledms/event"
	"github.com/simpledms/simpledms/model/common/spacerole"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
)

type UnassignUserFromSpaceCmdData struct {
	// TODO or assignment ID?
	UserAssignmentID int64 `validate:"required"`
}

type UnassignUserFromSpaceCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
	*autil.FormHelper[UnassignUserFromSpaceCmdData]
}

func NewUnassignUserFromSpaceCmd(infra *common.Infra, actions *Actions) *UnassignUserFromSpaceCmd {
	config := actionx.NewConfig(actions.Route("unassign-user-from-space-cmd"), false)
	return &UnassignUserFromSpaceCmd{
		infra:      infra,
		actions:    actions,
		Config:     config,
		FormHelper: autil.NewFormHelper[UnassignUserFromSpaceCmdData](infra, config, wx.T("Unassign user from space")),
	}
}

func (qq *UnassignUserFromSpaceCmd) Data(userAssignmentID int64) *UnassignUserFromSpaceCmdData {
	return &UnassignUserFromSpaceCmdData{
		UserAssignmentID: userAssignmentID,
	}
}

func (qq *UnassignUserFromSpaceCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[UnassignUserFromSpaceCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	if !ctx.IsSpaceCtx() {
		return e.NewHTTPErrorf(
			http.StatusBadRequest,
			"No space selected. Please select a space first.",
		)
	}
	if ctx.SpaceCtx().UserRoleInSpace() != spacerole.Owner {
		return e.NewHTTPErrorf(
			http.StatusForbidden,
			"You are not allowed to assign users to spaces because you aren't the owner.",
		)
	}

	assignment := ctx.SpaceCtx().Space.QueryUserAssignment().Where(spaceuserassignment.ID(data.UserAssignmentID)).OnlyX(ctx)
	if assignment.UserID == ctx.SpaceCtx().User.ID {
		return e.NewHTTPErrorf(http.StatusForbidden, "You cannot unassign yourself from a space.")
	}

	ctx.SpaceCtx().TTx.SpaceUserAssignment.DeleteOneID(data.UserAssignmentID).ExecX(ctx)

	rw.AddRenderables(wx.NewSnackbarf("User unassigned from space successfully."))
	rw.Header().Set("HX-Trigger", event.UserUnassignedFromSpace.String())

	return nil
}
