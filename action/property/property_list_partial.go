package property

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type PropertyListPartialData struct {
}

type PropertyListPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewPropertyListPartial(infra *common.Infra, actions *Actions) *PropertyListPartial {
	config := actionx.NewConfig(
		actions.Route("property-list-partial"),
		true,
	)
	return &PropertyListPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *PropertyListPartial) Data() *PropertyListPartialData {
	return &PropertyListPartialData{}
}

func (qq *PropertyListPartial) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[PropertyListPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw, ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *PropertyListPartial) Widget(ctx ctxx.Context, data *PropertyListPartialData) *wx.List {
	properties := ctx.TenantCtx().TTx.Property.Query().AllX(ctx)

	var items []*wx.ListItem

	items = append(items, &wx.ListItem{
		Headline: wx.T("Add field"),
		Type:     wx.ListItemTypeHelper,
		Leading:  wx.NewIcon("add"),
		HTMXAttrs: qq.actions.CreatePropertyCmd.ModalLinkAttrs(
			qq.actions.CreatePropertyCmd.Data(""),
			"",
		),
	})

	for _, propertyx := range properties {
		items = append(items, &wx.ListItem{
			Headline:       wx.Tu(propertyx.Name),
			SupportingText: wx.T(propertyx.Type.String()),
			Leading:        wx.NewIcon("list_alt"),
			ContextMenu:    NewPropertyContextMenuPartial(qq.actions).Widget(ctx, propertyx),
			/*Trailing: &wx.IconButton{
				Icon: "more_vert",
				// Children:  // TODO context menu
			},*/
		})
	}

	return &wx.List{
		Widget: wx.Widget[wx.List]{
			ID: qq.id(),
		},
		Children: items,
		HTMXAttrs: wx.HTMXAttrs{
			HxTrigger: event.HxTrigger(
				event.PropertyCreated,
				event.PropertyUpdated,
				event.PropertyDeleted,
			),
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(data),
			HxTarget: "#" + qq.id(),
			HxSwap:   "outerHTML",
		},
	}
}

func (qq *PropertyListPartial) id() string {
	return "propertyList"
}
