package documenttype

// package action

import (
	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/ui/uix/events"
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/attribute"
	"github.com/simpledms/simpledms/model/tenant/common/attributetype"
	"github.com/simpledms/simpledms/ui/uix/event"
)

type AttributesPartialData struct {
	DocumentTypeID int64
}

type AttributesPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewAttributesPartial(infra *common.Infra, actions *Actions) *AttributesPartial {
	config := actionx.NewConfig(
		actions.Route("attributes-partial"),
		true,
	)
	return &AttributesPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *AttributesPartial) Data(documentTypeID int64) *AttributesPartialData {
	return &AttributesPartialData{
		DocumentTypeID: documentTypeID,
	}
}

func (qq *AttributesPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AttributesPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *AttributesPartial) Widget(ctx ctxx.Context, data *AttributesPartialData) *widget.List {
	attributes := ctx.TenantCtx().TTx.Attribute.
		Query().
		WithTag().
		WithProperty().
		Where(attribute.DocumentTypeID(data.DocumentTypeID)).
		AllX(ctx)
	var items []*widget.ListItem

	items = append(items, &widget.ListItem{
		Headline: widget.T("Add field attribute"),
		Type:     widget.ListItemTypeHelper,
		Leading:  widget.NewIcon("add"),
		HTMXAttrs: qq.actions.AddPropertyAttributeCmd.ModalLinkAttrs(
			qq.actions.AddPropertyAttributeCmd.Data(data.DocumentTypeID),
			"",
		),
	})
	items = append(items, &widget.ListItem{
		Headline: widget.T("Add list attribute (tag group)"),
		Type:     widget.ListItemTypeHelper,
		Leading:  widget.NewIcon("add"),
		HTMXAttrs: qq.actions.CreateAttributeCmd.ModalLinkAttrs(
			qq.actions.CreateAttributeCmd.Data(data.DocumentTypeID),
			"",
		),
	})

	for _, attributex := range attributes {
		if attributex.Type == attributetype.Field {
			supportingText := widget.T(attributex.Edges.Property.Type.String())
			if attributex.IsNameGiving {
				supportingText = widget.Tuf("%s, %s", supportingText.String(ctx), widget.T("name-giving").String(ctx))
			}
			items = append(items, &widget.ListItem{
				Headline:       widget.Tu(attributex.Edges.Property.Name),
				SupportingText: supportingText,
				Leading:        widget.NewIcon("list_alt"), // TODO okay?
				ContextMenu:    NewAttributeContextMenuWidget(qq.actions).Widget(ctx, attributex),
			})
		} else if attributex.Type == attributetype.Tag {
			supportingText := widget.Tu(attributex.Edges.Tag.Name)
			if attributex.IsNameGiving {
				supportingText = widget.Tuf("%s, %s", supportingText.String(ctx), widget.T("name-giving").String(ctx))
			}
			items = append(items, &widget.ListItem{
				Headline:       widget.Tu(attributex.Name),
				SupportingText: supportingText,
				Leading:        widget.NewIcon("list_alt"), // TODO okay?
				ContextMenu:    NewAttributeContextMenuWidget(qq.actions).Widget(ctx, attributex),
			})
		} else {
			// TODO okay?
			panic("unknown attribute type")
		}
	}

	return &widget.List{
		Widget: widget.Widget[widget.List]{
			ID: qq.listID(),
		},
		Children: items,
		HTMXAttrs: widget.HTMXAttrs{
			HxTrigger: events.HxTrigger(
				event.DocumentTypeAttributeCreated,
				event.DocumentTypeAttributeUpdated,
				event.DocumentTypeAttributeDeleted,
			),
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(data),
			HxTarget: "#" + qq.listID(),
			HxSwap:   "outerHTML",
		},
	}
}

func (qq *AttributesPartial) listID() string {
	return "attributesList"
}
