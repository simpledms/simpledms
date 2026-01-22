package documenttype

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type DeleteCmdData struct {
	ID int64
}

type DeleteCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDeleteCmd(infra *common.Infra, actions *Actions) *DeleteCmd {
	config := actionx.NewConfig(
		actions.Route("delete"),
		false,
	)
	return &DeleteCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *DeleteCmd) Data(id int64) *DeleteCmdData {
	return &DeleteCmdData{
		ID: id,
	}
}

func (qq *DeleteCmd) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[DeleteCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	err = ctx.TenantCtx().TTx.DocumentType.
		DeleteOneID(data.ID).
		Where(documenttype.SpaceID(ctx.SpaceCtx().Space.ID)).
		Exec(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("HX-Trigger", event.DocumentTypeDeleted.String())
	rw.AddRenderables(wx.NewSnackbarf("Document type deleted."))

	return nil
}
