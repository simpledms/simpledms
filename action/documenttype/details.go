package documenttype

// package action

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/event"
	"github.com/simpledms/simpledms/uix/route"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type DetailsData struct {
	ID int64
}

type Details struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewDetails(infra *common.Infra, actions *Actions) *Details {
	config := actionx.NewConfig(
		actions.Route("details"),
		true,
	)
	return &Details{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *Details) Data(id int64) *DetailsData {
	return &DetailsData{
		ID: id,
	}
}

func (qq *Details) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[DetailsData](rw, req, ctx)
	if err != nil {
		return err
	}

	state := autil.StateX[PageState](rw, req)
	documentTypex := ctx.SpaceCtx().Space.QueryDocumentTypes().Where(documenttype.ID(data.ID)).OnlyX(ctx)

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, state, documentTypex),
	)
}

func (qq *Details) Widget(
	ctx ctxx.Context,
	state *PageState,
	documentTypex *enttenant.DocumentType,
) *wx.DetailsWithSheet {
	return &wx.DetailsWithSheet{
		AppBar: qq.appBar(ctx, documentTypex),
		Child: []wx.IWidget{
			qq.actions.Properties.Widget(ctx, &AttributesData{
				DocumentTypeID: documentTypex.ID,
			}),
		},
	}
}

func (qq *Details) appBar(ctx ctxx.Context, documentTypex *enttenant.DocumentType) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.IconButton{
			Icon: "close",
			// TODO use link instead?
			HTMXAttrs: wx.HTMXAttrs{
				HxGet:     route.ManageDocumentTypes(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID),
				HxOn:      event.DetailsClosed.HxOn("click"),
				HxHeaders: autil.CloseDetailsHeader(),
			},
		},
		Title: &wx.AppBarTitle{
			Text: wx.Tu(documentTypex.Name),
		},
		Actions: []wx.IWidget{
			/*&wx.IconButton{
				Icon: "more_vert",
				Children: &wx.Menu{
					Items: []*wx.MenuItem{}, // TODO
				},
			},*/
		},
	}
}
