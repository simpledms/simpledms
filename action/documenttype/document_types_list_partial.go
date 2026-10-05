package documenttype

// package action

import (
	"net/url"
	"path"
	"strconv"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/documenttype"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
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

	var selectedID int64
	if current, err := url.Parse(req.Header.Get("HX-Current-URL")); err == nil {
		selectedID, _ = strconv.ParseInt(path.Base(current.Path), 10, 64)
	}
	if selectedID != 0 && !ctx.SpaceCtx().Space.QueryDocumentTypes().Where(
		documenttype.ID(selectedID),
	).ExistX(ctx) {
		selectedID = 0
		rw.Header().Set("HX-Replace-Url", route.ManageDocumentTypes(
			ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID,
		))
	}
	return qq.infra.Renderer().Render(rw, ctx,
		qq.actions.DocumentTypePage.WidgetHandler(rw, req, ctx, selectedID))
}

func (qq *DocumentTypesListPartial) Widget(ctx ctxx.Context, selectedTypeID int64) renderable.Renderable {
	types := ctx.SpaceCtx().Space.QueryDocumentTypes().Order(documenttype.ByName()).AllX(ctx)
	var items []*widget.ListItem

	id := "documentTypesList"

	for _, typex := range types {
		items = append(items, qq.ListItem(ctx, typex, typex.ID == selectedTypeID))
	}

	htmxAttrs := widget.HTMXAttrs{
		HxPost: qq.Endpoint(),
		HxTrigger: event.HxTrigger(
			event.DocumentTypeCreated,
			event.DocumentTypeUpdated,
			event.DocumentTypeDeleted,
		),
		// HxVals:    util.JSON(qq.Data()),
		HxTarget: "#innerContent",
		HxSwap:   "innerHTML",
	}

	if len(items) == 0 {
		return &widget.Container{
			Widget: widget.Widget[widget.Container]{
				ID: id,
			},
			Child: &widget.EmptyState{
				Icon:     widget.NewIcon("category"),
				Headline: widget.T("No document types available yet."),
				Actions: []widget.IWidget{
					&widget.Button{
						Icon:  widget.NewIcon("add"),
						Label: widget.T("Create document type"),
						HTMXAttrs: qq.actions.CreateCmd.ModalLinkAttrs(
							qq.actions.CreateCmd.Data(""),
							"",
						),
					},
				},
			},
			HTMXAttrs: htmxAttrs,
		}
	}

	return &widget.List{
		Widget: widget.Widget[widget.List]{
			ID: id,
		},
		HTMXAttrs: htmxAttrs,
		Children:  items,
	}
}

func (qq *DocumentTypesListPartial) ListItem(ctx ctxx.Context, typex *enttenant.DocumentType, isSelected bool) *widget.ListItem {
	icon := "category"
	if typex.Icon != "" {
		icon = typex.Icon
	}
	return &widget.ListItem{
		Widget: widget.Widget[widget.ListItem]{},
		HTMXAttrs: widget.HTMXAttrs{
			HxGet: route.ManageDocumentTypesWithSelection(ctx.TenantCtx().TenantID, ctx.SpaceCtx().SpaceID, typex.ID),
		},
		RadioGroupName: "documentTypes",
		Leading:        widget.NewIcon(icon),
		Headline:       widget.Tu(typex.Name),
		IsSelected:     isSelected,
		ContextMenu:    NewContextMenuWidget(qq.actions).Widget(ctx, typex),
	}
}
