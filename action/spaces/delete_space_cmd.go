package spaces

import (
	"log"
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

func (qq *DeleteSpaceCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DeleteSpaceCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	// assumes it is on spaces screen, not dashboard
	err = ctx.TenantCtx().TTx.Space.Update().
		SetDeletedAt(time.Now()).
		SetDeleter(ctx.TenantCtx().User).
		Where(space.PublicID(entx.NewCIText(data.SpaceID))).
		Exec(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("HX-Trigger", event.SpaceDeleted.String())
	rw.AddRenderables(wx.NewSnackbarf("Space deleted."))

	return nil
}
