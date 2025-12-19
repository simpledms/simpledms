package documenttype

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/enttenant/documenttype"
	"github.com/simpledms/simpledms/event"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type DeleteData struct {
	ID int64
}

type Delete struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDelete(infra *common.Infra, actions *Actions) *Delete {
	config := actionx.NewConfig(
		actions.Route("delete"),
		false,
	)
	return &Delete{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *Delete) Data(id int64) *DeleteData {
	return &DeleteData{
		ID: id,
	}
}

func (qq *Delete) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[DeleteData](rw, req, ctx)
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
