package spaces

import (
	"time"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/space"
	"github.com/simpledms/simpledms/db/entx"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type DeleteSpaceData struct {
	SpaceID string `validate:"required" form_attr_type:"hidden"`
}

type DeleteSpace struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDeleteSpace(infra *common.Infra, actions *Actions) *DeleteSpace {
	config := actionx.NewConfig(actions.Route("delete-space"), false)
	return &DeleteSpace{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *DeleteSpace) Data(spaceID string) *DeleteSpaceData {
	return &DeleteSpaceData{
		SpaceID: spaceID,
	}
}

func (qq *DeleteSpace) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DeleteSpaceData](rw, req, ctx)
	if err != nil {
		return err
	}

	// assumes it is on spaces screen, not dashboard
	ctx.TenantCtx().TTx.Space.Update().
		SetDeletedAt(time.Now()).
		SetDeleter(ctx.TenantCtx().User).
		Where(space.PublicID(entx.NewCIText(data.SpaceID))).
		ExecX(ctx)

	rw.Header().Set("HX-Trigger", event.SpaceDeleted.String())
	rw.AddRenderables(wx.NewSnackbarf("Space deleted."))

	return nil
}
