package documenttype

// package action

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/attribute"
	"github.com/simpledms/simpledms/model/common/attributetype"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type AttributesData struct {
	DocumentTypeID int64
}

type Attributes struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewAttributes(infra *common.Infra, actions *Actions) *Attributes {
	config := actionx.NewConfig(
		actions.Route("attributes"),
		true,
	)
	return &Attributes{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *Attributes) Data(documentTypeID int64) *AttributesData {
	return &AttributesData{
		DocumentTypeID: documentTypeID,
	}
}

func (qq *Attributes) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AttributesData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *Attributes) Widget(ctx ctxx.Context, data *AttributesData) *wx.List {
	attributes := ctx.TenantCtx().TTx.Attribute.
		Query().
		WithTag().
		WithProperty().
		Where(attribute.DocumentTypeID(data.DocumentTypeID)).
		AllX(ctx)
	var items []*wx.ListItem

	items = append(items, &wx.ListItem{
		Headline: wx.T("Add field attribute"),
		Type:     wx.ListItemTypeHelper,
		Leading:  wx.NewIcon("add"),
		HTMXAttrs: qq.actions.AddPropertyAttribute.ModalLinkAttrs(
			qq.actions.AddPropertyAttribute.Data(data.DocumentTypeID),
			"",
		),
	})
	items = append(items, &wx.ListItem{
		Headline: wx.T("Add list attribute (tag group)"),
		Type:     wx.ListItemTypeHelper,
		Leading:  wx.NewIcon("add"),
		HTMXAttrs: qq.actions.CreateAttribute.ModalLinkAttrs(
			qq.actions.CreateAttribute.Data(data.DocumentTypeID),
			"",
		),
	})

	for _, attributex := range attributes {
		if attributex.Type == attributetype.Field {
			supportingText := wx.T(attributex.Edges.Property.Type.String())
			if attributex.IsNameGiving {
				supportingText = wx.Tuf("%s, %s", supportingText.String(ctx), wx.T("name-giving").String(ctx))
			}
			items = append(items, &wx.ListItem{
				Headline:       wx.Tu(attributex.Edges.Property.Name),
				SupportingText: supportingText,
				Leading:        wx.NewIcon("list_alt"), // TODO okay?
				ContextMenu:    NewAttributeContextMenu(qq.actions).Widget(ctx, attributex),
			})
		} else if attributex.Type == attributetype.Tag {
			supportingText := wx.Tu(attributex.Edges.Tag.Name)
			if attributex.IsNameGiving {
				supportingText = wx.Tuf("%s, %s", supportingText.String(ctx), wx.T("name-giving").String(ctx))
			}
			items = append(items, &wx.ListItem{
				Headline:       wx.Tu(attributex.Name),
				SupportingText: supportingText,
				Leading:        wx.NewIcon("list_alt"), // TODO okay?
				ContextMenu:    NewAttributeContextMenu(qq.actions).Widget(ctx, attributex),
			})
		} else {
			// TODO okay?
			panic("unknown attribute type")
		}
	}

	return &wx.List{
		Widget: wx.Widget[wx.List]{
			ID: qq.listID(),
		},
		Children: items,
		HTMXAttrs: wx.HTMXAttrs{
			HxTrigger: event.HxTrigger(
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

func (qq *Attributes) listID() string {
	return "attributesList"
}
