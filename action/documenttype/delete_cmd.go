package documenttype

import (
	"log"

	autil "github.com/marcobeierer/go-core/action/util"
	wx "github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	documenttypemodel "github.com/simpledms/simpledms/model/tenant/documenttype"
	"github.com/simpledms/simpledms/ui/uix/event"
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
		actions.Route("delete-cmd"),
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
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[DeleteCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	documentTypex, err := documenttypemodel.QueryByID(ctx, ctx.SpaceCtx().Space.ID, data.ID)
	if err != nil {
		log.Println(err)
		return err
	}

	err = documentTypex.Delete(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("HX-Trigger", event.DocumentTypeDeleted.String())
	rw.AddRenderables(wx.NewSnackbarf("Document type deleted."))

	return nil
}
