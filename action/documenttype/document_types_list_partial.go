package documenttype

// package action

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type DocumentTypesListPartialData struct {
}

type DocumentTypesListPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewListDocumentTypesPartial(infra *common.Infra, actions *Actions) *DocumentTypesListPartial {
	return &DocumentTypesListPartial{
		infra:   infra,
		actions: actions,
		Config: actionx.NewConfig(
			actions.Route("document-types-list-partial"),
			true,
		),
	}
}

func (qq *DocumentTypesListPartial) Data() *DocumentTypesListPartialData {
	return &DocumentTypesListPartialData{}
}

func (qq *DocumentTypesListPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	_, err := autil.FormData[DocumentTypesListPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, 0),
	)
}

func (qq *DocumentTypesListPartial) Widget(ctx ctxx.Context, selectedTypeID int64) renderable.Renderable {
	types := ctx.SpaceCtx().Space.QueryDocumentTypes().Order(documenttype.ByName()).AllX(ctx)
	var items []*wx.ListItem

	id := "documentTypesList"

	/*
		if len(types) == 0 {
			return &wx.EmptyState{
				Widget: wx.Widget[wx.EmptyState]{
					ID: id,
				},
				Icon:     wx.NewIcon("description"),
				Headline: wx.T("No document types available yet."),
				// TODO actions
			}
		}
	*/

	items = append(items, &wx.ListItem{
		Headline: wx.T("Add document type"),
		Type:     wx.ListItemTypeHelper,
		Leading:  wx.NewIcon("add"),
		HTMXAttrs: qq.actions.CreateCmd.ModalLinkAttrs(
			qq.actions.CreateCmd.Data(""),
			"",
		),
	})

	for _, typex := range types {
		items = append(items, qq.ListItem(ctx, typex, typex.ID == selectedTypeID))
	}

	return &wx.List{
		Widget: wx.Widget[wx.List]{
			ID: id,
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxPost: qq.Endpoint(),
			HxTrigger: event.HxTrigger(
				event.DocumentTypeCreated,
				event.DocumentTypeUpdated,
				event.DocumentTypeDeleted,
			),
			// HxVals:    util.JSON(qq.Data()),
			HxTarget: "#" + id,
			HxSwap:   "outerHTML",
		},
		Children: items,
	}
}

func (qq *DocumentTypesListPartial) ListItem(ctx ctxx.Context, typex *enttenant.DocumentType, isSelected bool) *wx.ListItem {
	icon := "category"
	if typex.Icon != "" {
		icon = typex.Icon
	}
	return &wx.ListItem{
		Widget: wx.Widget[wx.ListItem]{},
		HTMXAttrs: wx.HTMXAttrs{
			HxGet: route.ManageDocumentTypesWithSelection(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, typex.ID),
		},
		RadioGroupName: "documentTypes",
		Leading:        wx.NewIcon(icon),
		Headline:       wx.Tu(typex.Name),
		IsSelected:     isSelected,
		ContextMenu:    NewContextMenuPartial(qq.actions).Widget(ctx, typex),
	}
}
