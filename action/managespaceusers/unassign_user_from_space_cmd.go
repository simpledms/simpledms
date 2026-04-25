package managespaceusers

import (
	"net/http"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	"github.com/marcobeierer/go-core/util/e"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/model/tenant/common/spacerole"
	spacemodel "github.com/simpledms/simpledms/model/tenant/space"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type UnassignUserFromSpaceCmdData struct {
	// TODO or assignment ID?
	UserAssignmentID int64 `validate:"required"`
}

type UnassignUserFromSpaceCmd struct {
	infra           *common.Infra
	actions         *Actions
	spaceRepository spacemodel.SpaceRepository
	*actionx.Config
	*autil.FormHelper[UnassignUserFromSpaceCmdData]
}

func NewUnassignUserFromSpaceCmd(infra *common.Infra, actions *Actions) *UnassignUserFromSpaceCmd {
	config := actionx.NewConfig(actions.Route("unassign-user-from-space-cmd"), false)
	return &UnassignUserFromSpaceCmd{
		infra:           infra,
		actions:         actions,
		spaceRepository: spacemodel.NewEntSpaceRepository(),
		Config:          config,
		FormHelper:      autil.NewFormHelper[UnassignUserFromSpaceCmdData](infra, config, widget.T("Unassign user from space")),
	}
}

func (qq *UnassignUserFromSpaceCmd) Data(userAssignmentID int64) *UnassignUserFromSpaceCmdData {
	return &UnassignUserFromSpaceCmdData{
		UserAssignmentID: userAssignmentID,
	}
}

func (qq *UnassignUserFromSpaceCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

	err = spacemodel.NewSpaceWithRepository(ctx.SpaceCtx().Space, qq.spaceRepository).UnassignUser(
		ctx,
		data.UserAssignmentID,
		ctx.SpaceCtx().User.ID,
	)
	if err != nil {
		return mapSpaceError(err)
	}

	rw.AddRenderables(widget.NewSnackbarf("User unassigned from space successfully."))
	rw.Header().Set("HX-Trigger", event.UserUnassignedFromSpace.String())

	return nil
}
