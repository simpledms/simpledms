package spaces

import (
	"github.com/simpledms/simpledms/core/db/entx"

	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/space"
	spacemodel "github.com/simpledms/simpledms/model/tenant/space"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type DeleteSpaceCmdData struct {
	SpaceID string `validate:"required" form_attr_type:"hidden"`
}

type DeleteSpaceCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDeleteSpaceCmd(infra *common.Infra, actions *Actions) *DeleteSpaceCmd {
	config := actionx.NewConfig(actions.Route("delete-space-cmd"), false)
	return &DeleteSpaceCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *DeleteSpaceCmd) Data(spaceID string) *DeleteSpaceCmdData {
	return &DeleteSpaceCmdData{
		SpaceID: spaceID,
	}
}

func (qq *DeleteSpaceCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DeleteSpaceCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	spacex, err := ctx.TenantCtx().TTx.Space.Query().
		Where(space.PublicID(entx.NewCIText(data.SpaceID))).
		Only(ctx)
	if err != nil {
		return err
	}

	err = spacemodel.NewSpace(spacex).Delete(ctx, ctx.TenantCtx().User)
	if err != nil {
		return err
	}

	rw.Header().Set("HX-Trigger", event.SpaceDeleted.String())
	rw.AddRenderables(wx.NewSnackbarf("Space deleted."))

	return nil
}
