package documenttype

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type DeleteAttributeCmdData struct {
	ID int64
}

type DeleteAttributeCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDeleteAttributeCmd(infra *common.Infra, actions *Actions) *DeleteAttributeCmd {
	config := actionx.NewConfig(
		actions.Route("delete-attribute"),
		false,
	)
	return &DeleteAttributeCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *DeleteAttributeCmd) Data(id int64) *DeleteAttributeCmdData {
	return &DeleteAttributeCmdData{
		ID: id,
	}
}

func (qq *DeleteAttributeCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DeleteAttributeCmdData](rw, req, ctx)
	if err != nil {
		return err
	}

	err = ctx.TenantCtx().TTx.Attribute.DeleteOneID(data.ID).Exec(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	rw.Header().Set("HX-Trigger", event.DocumentTypeAttributeDeleted.String())
	rw.AddRenderables(wx.NewSnackbarf("Attribute deleted."))

	return nil
}
