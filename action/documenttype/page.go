package documenttype

// package action

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/partial"
	"github.com/simpledms/simpledms/util/httpx"
)

type PageState struct {
}

type Page struct {
	infra   *common.Infra
	actions *Actions
}

func NewPage(infra *common.Infra, actions *Actions) *Page {
	return &Page{
		infra:   infra,
		actions: actions,
	}
}

func (qq *Page) WidgetHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	selectedTypeID int64,
) *wx.ListDetailLayout {
	state := autil.StateX[PageState](rw, req)

	return qq.Widget(ctx, state, selectedTypeID)
}

func (qq *Page) Widget(
	ctx ctxx.Context,
	state *PageState,
	selectedTypeID int64,
) *wx.ListDetailLayout {
	var nullableDetail *wx.DetailsWithSheet

	if selectedTypeID != 0 {
		documentTypex := ctx.SpaceCtx().Space.QueryDocumentTypes().Where(documenttype.ID(selectedTypeID)).OnlyX(ctx)
		nullableDetail = qq.actions.Details.Widget(ctx, state, documentTypex)
		/*&wx.DetailWithSheet{
			AppBar: qq.appBar(ctx),
			Child:  qq.actions.Details.Widget(ctx, state, documentTypex),
		}*/
	}

	return &wx.ListDetailLayout{
		AppBar: qq.appBar(ctx),
		List: &wx.Column{
			Children: []wx.IWidget{
				qq.actions.ListDocumentTypes.Widget(
					ctx,
					selectedTypeID,
				),
			},
		},
		Detail: nullableDetail,
	}
}

func (qq *Page) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.Icon{
			Name: "category",
		},
		LeadingAltMobile: partial.NewMainMenu(ctx),
		Title: &wx.AppBarTitle{
			Text: wx.T("Document types"),
		},
		Actions: []wx.IWidget{},
	}
}
