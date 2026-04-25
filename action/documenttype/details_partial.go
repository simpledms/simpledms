package documenttype

// package action

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/uix/events"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type DetailsPartialData struct {
	ID int64
}

type DetailsPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDetailsPartial(infra *common.Infra, actions *Actions) *DetailsPartial {
	config := actionx.NewConfig(
		actions.Route("details-partial"),
		true,
	)
	return &DetailsPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *DetailsPartial) Data(id int64) *DetailsPartialData {
	return &DetailsPartialData{
		ID: id,
	}
}

func (qq *DetailsPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DetailsPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	state := autil.StateX[DocumentTypePageState](rw, req)
	documentTypex := ctx.SpaceCtx().Space.QueryDocumentTypes().Where(documenttype.ID(data.ID)).OnlyX(ctx)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, state, documentTypex),
	)
}

func (qq *DetailsPartial) Widget(
	ctx ctxx.Context,
	state *DocumentTypePageState,
	documentTypex *enttenant.DocumentType,
) *widget.DetailsWithSheet {
	return &widget.DetailsWithSheet{
		AppBar: qq.appBar(ctx, documentTypex),
		Child: []widget.IWidget{
			qq.actions.Properties.Widget(ctx, &AttributesPartialData{
				DocumentTypeID: documentTypex.ID,
			}),
		},
	}
}

func (qq *DetailsPartial) appBar(ctx ctxx.Context, documentTypex *enttenant.DocumentType) *widget.AppBar {
	return &widget.AppBar{
		Leading: &widget.IconButton{
			Icon:    "close",
			Tooltip: widget.T("Close details"),
			// TODO use link instead?
			HTMXAttrs: widget.HTMXAttrs{
				HxGet:     route.ManageDocumentTypes(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				HxOn:      events.DetailsClosed.HxOn("click"),
				HxHeaders: autil.CloseDetailsHeader(),
			},
		},
		Title: &widget.AppBarTitle{
			Text: widget.Tu(documentTypex.Name),
		},
		Actions: []widget.IWidget{
			/*&wx.IconButton{
				Icon: "more_vert",
				Children: &wx.Menu{
					Items: []*wx.MenuItem{}, // TODO
				},
			},*/
		},
	}
}
