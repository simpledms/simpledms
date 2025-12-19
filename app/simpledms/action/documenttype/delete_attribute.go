package documenttype

import (
	"log"

	autil "github.com/simpledms/simpledms/app/simpledms/action/util"
	"github.com/simpledms/simpledms/app/simpledms/common"
	"github.com/simpledms/simpledms/app/simpledms/ctxx"
	"github.com/simpledms/simpledms/app/simpledms/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type DeleteAttributeData struct {
	ID int64
}

type DeleteAttribute struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDeleteAttribute(infra *common.Infra, actions *Actions) *DeleteAttribute {
	config := actionx.NewConfig(
		actions.Route("delete-attribute"),
		false,
	)
	return &DeleteAttribute{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *DeleteAttribute) Data(id int64) *DeleteAttributeData {
	return &DeleteAttributeData{
		ID: id,
	}
}

func (qq *DeleteAttribute) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DeleteAttributeData](rw, req, ctx)
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
