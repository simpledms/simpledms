package documenttype

// package action

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/ui/uix/partial"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/httpx"
)

type DocumentTypePageState struct {
}

type DocumentTypePage struct {
	infra   *common.Infra
	actions *Actions
}

func NewDocumentTypePage(infra *common.Infra, actions *Actions) *DocumentTypePage {
	return &DocumentTypePage{
		infra:   infra,
		actions: actions,
	}
}

func (qq *DocumentTypePage) WidgetHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
	selectedTypeID int64,
) *wx.ListDetailLayout {
	state := autil.StateX[DocumentTypePageState](rw, req)

	return qq.Widget(ctx, state, selectedTypeID)
}

func (qq *DocumentTypePage) Widget(
	ctx ctxx.Context,
	state *DocumentTypePageState,
	selectedTypeID int64,
) *wx.ListDetailLayout {
	var nullableDetail *wx.DetailsWithSheet

	if selectedTypeID != 0 {
		documentTypex := ctx.SpaceCtx().Space.QueryDocumentTypes().Where(documenttype.ID(selectedTypeID)).OnlyX(ctx)
		nullableDetail = qq.actions.DetailsPartial.Widget(ctx, state, documentTypex)
		/*&wx.DetailWithSheet{
			AppBar: qq.appBar(ctx),
			Child:  qq.actions.DetailsPartial.Widget(ctx, state, documentTypex),
		}*/
	}

	return &wx.ListDetailLayout{
		AppBar: qq.appBar(ctx),
		List: &wx.Column{
			Children: []wx.IWidget{
				qq.actions.ListDocumentTypesPartial.Widget(
					ctx,
					selectedTypeID,
				),
			},
		},
		Detail: nullableDetail,
	}
}

func (qq *DocumentTypePage) appBar(ctx ctxx.Context) *wx.AppBar {
	return &wx.AppBar{
		Leading: &wx.Icon{
			Name: "category",
		},
		LeadingAltMobile: partial.NewMainMenu(ctx, qq.infra),
		Title: &wx.AppBarTitle{
			Text: wx.T("Document types"),
		},
		Actions: []wx.IWidget{},
	}
}
