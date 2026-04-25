package documenttype

// package action

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/uix/partial"
	"github.com/marcobeierer/go-core/ui/widget"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
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
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
	selectedTypeID int64,
) *widget.ListDetailLayout {
	state := autil.StateX[DocumentTypePageState](rw, req)

	return qq.Widget(ctx, state, selectedTypeID)
}

func (qq *DocumentTypePage) Widget(
	ctx ctxx.Context,
	state *DocumentTypePageState,
	selectedTypeID int64,
) *widget.ListDetailLayout {
	var nullableDetail *widget.DetailsWithSheet

	if selectedTypeID != 0 {
		documentTypex := ctx.SpaceCtx().Space.QueryDocumentTypes().Where(documenttype.ID(selectedTypeID)).OnlyX(ctx)
		nullableDetail = qq.actions.DetailsPartial.Widget(ctx, state, documentTypex)
		/*&wx.DetailWithSheet{
			AppBar: qq.appBar(ctx),
			Child:  qq.actions.DetailsPartial.Widget(ctx, state, documentTypex),
		}*/
	}

	return &widget.ListDetailLayout{
		AppBar: qq.appBar(ctx),
		List: &widget.Column{
			Children: []widget.IWidget{
				qq.actions.ListDocumentTypesPartial.Widget(
					ctx,
					selectedTypeID,
				),
			},
		},
		Detail: nullableDetail,
	}
}

func (qq *DocumentTypePage) appBar(ctx ctxx.Context) *widget.AppBar {
	return &widget.AppBar{
		Leading: &widget.Icon{
			Name: "category",
		},
		LeadingAltMobile: partial.NewMainMenu(ctx, qq.infra),
		Title: &widget.AppBarTitle{
			Text: widget.T("Document types"),
		},
		Actions: []widget.IWidget{},
	}
}
