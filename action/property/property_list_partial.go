package property

import (
	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ui/uix/events"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/ui/uix/event"
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
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
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

func (qq *PropertyListPartial) Widget(ctx ctxx.Context, data *PropertyListPartialData) *widget.List {
	properties := ctx.AppCtx().TTx.Property.Query().AllX(ctx)

	var items []*widget.ListItem

	items = append(items, &widget.ListItem{
		Headline: widget.T("Add field"),
		Type:     widget.ListItemTypeHelper,
		Leading:  widget.NewIcon("add"),
		HTMXAttrs: qq.actions.CreatePropertyCmd.ModalLinkAttrs(
			qq.actions.CreatePropertyCmd.Data(""),
			"",
		),
	})

	for _, propertyx := range properties {
		items = append(items, &widget.ListItem{
			Headline:       widget.Tu(propertyx.Name),
			SupportingText: widget.T(propertyx.Type.String()),
			Leading:        widget.NewIcon("list_alt"),
			ContextMenu:    NewPropertyContextMenuWidget(qq.actions).Widget(ctx, propertyx),
			/*Trailing: &wx.IconButton{
				Icon: "more_vert",
				// Children:  // TODO context menu
			},*/
		})
	}

	return &widget.List{
		Widget: widget.Widget[widget.List]{
			ID: qq.id(),
		},
		Children: items,
		HTMXAttrs: widget.HTMXAttrs{
			HxTrigger: events.HxTrigger(
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
