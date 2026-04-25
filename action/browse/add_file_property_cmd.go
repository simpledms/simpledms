package browse

import (
	"entgo.io/ent/dialect/sql"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/marcobeierer/go-core/ui/renderable"
	"github.com/marcobeierer/go-core/ui/uix/events"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	actionx2 "github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
)

type AddFilePropertyCmdData struct {
	FileID string `form_attr_type:"hidden"`
}

type AddFilePropertyCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx2.Config
}

func NewAddFilePropertyCmd(infra *common.Infra, actions *Actions) *AddFilePropertyCmd {
	config := actionx2.NewConfig(
		actions.Route("add-file-property-cmd"),
		false,
	)
	return &AddFilePropertyCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *AddFilePropertyCmd) Data(fileID string) *AddFilePropertyCmdData {
	return &AddFilePropertyCmdData{
		FileID: fileID,
	}
}

func (qq *AddFilePropertyCmd) ModalLinkAttrs(data *AddFilePropertyCmdData, hxTargetForm string) widget.HTMXAttrs {
	return widget.HTMXAttrs{
		HxPost:        qq.FormEndpointWithParams(actionx2.ResponseWrapperDialog, hxTargetForm),
		HxVals:        util.JSON(data),
		LoadInPopover: true,
	}
}

func (qq *AddFilePropertyCmd) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AddFilePropertyCmdData](rw, req, ctx)
	if err != nil {
		return err
	}
	return qq.infra.Renderer().Render(rw, ctx, qq.listContent(ctx, data))
}

func (qq *AddFilePropertyCmd) FormHandler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormDataX[AddFilePropertyCmdData](rw, req, ctx, true)
	if err != nil {
		return err
	}

	wrapper := req.URL.Query().Get("wrapper")

	return qq.infra.Renderer().Render(rw, ctx,
		qq.Form(ctx, data, actionx2.ResponseWrapper(wrapper)),
	)
}

func (qq *AddFilePropertyCmd) Form(
	ctx ctxx.Context,
	data *AddFilePropertyCmdData,
	wrapper actionx2.ResponseWrapper,
) renderable.Renderable {
	return autil.WrapWidgetWithID(
		widget.T("Add field"),
		nil,
		qq.listContent(ctx, data),
		wrapper,
		widget.DialogLayoutStable,
		qq.popoverID(),
		"",
	)
}

func (qq *AddFilePropertyCmd) popoverID() string {
	return "addFilePropertyPopover"
}

func (qq *AddFilePropertyCmd) listID() string {
	return "addFilePropertyList"
}

func (qq *AddFilePropertyCmd) listContent(ctx ctxx.Context, data *AddFilePropertyCmdData) renderable.Renderable {
	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	properties := ctx.SpaceCtx().Space.QueryProperties().
		Where(func(qs *sql.Selector) {
			assignmentTable := sql.Table(filepropertyassignment.Table)
			qs.Where(
				sql.NotIn(
					qs.C(property.FieldID),
					sql.
						Select(assignmentTable.C(filepropertyassignment.FieldPropertyID)).
						From(assignmentTable).
						Where(sql.EQ(assignmentTable.C(filepropertyassignment.FieldFileID), filex.Data.ID)),
				),
			)
		}).
		Order(property.ByName()).
		AllX(ctx)

	var child widget.IWidget

	if len(properties) == 0 {
		child = &widget.EmptyState{
			Headline: widget.T("No unassigned fields available."),
			Actions: []widget.IWidget{
				&widget.Button{
					Label: widget.T("Manage fields"),
					Icon:  widget.NewIcon("tune"),
					HTMXAttrs: widget.HTMXAttrs{
						HxGet: route.ManageProperties(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
						HxOn:  events.CloseDialog.HxOn("click"),
					},
				},
			},
		}
	} else {
		var listItems []*widget.ListItem
		for _, propertyx := range properties {
			listItems = append(listItems, &widget.ListItem{
				Headline:       widget.Tu(propertyx.Name),
				SupportingText: widget.Tu(propertyx.Type.String()),
				Leading:        widget.NewIcon("tune"),
				HTMXAttrs: qq.actions.AddFilePropertyValueDialog.ModalLinkAttrs(
					qq.actions.AddFilePropertyValueDialog.Data(data.FileID, propertyx.ID),
					"",
				),
			})
		}

		child = &widget.List{
			Children: listItems,
		}
	}

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: qq.listID(),
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxTrigger: events.HxTrigger(event.FilePropertyUpdated),
			HxPost:    qq.Endpoint(),
			HxVals:    util.JSON(data),
			HxTarget:  "#" + qq.listID(),
			HxSwap:    "outerHTML",
		},
		Children: child,
	}
}
