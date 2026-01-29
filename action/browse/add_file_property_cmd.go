package browse

import (
	"entgo.io/ent/dialect/sql"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/uix/route"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type AddFilePropertyCmdData struct {
	FileID string `form_attr_type:"hidden"`
}

type AddFilePropertyCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewAddFilePropertyCmd(infra *common.Infra, actions *Actions) *AddFilePropertyCmd {
	config := actionx.NewConfig(
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

func (qq *AddFilePropertyCmd) ModalLinkAttrs(data *AddFilePropertyCmdData, hxTargetForm string) wx.HTMXAttrs {
	return wx.HTMXAttrs{
		HxPost:        qq.FormEndpointWithParams(actionx.ResponseWrapperDialog, hxTargetForm),
		HxVals:        util.JSON(data),
		LoadInPopover: true,
	}
}

func (qq *AddFilePropertyCmd) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[AddFilePropertyCmdData](rw, req, ctx)
	if err != nil {
		return err
	}
	return qq.infra.Renderer().Render(rw, ctx, qq.listContent(ctx, data))
}

func (qq *AddFilePropertyCmd) FormHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormDataX[AddFilePropertyCmdData](rw, req, ctx, true)
	if err != nil {
		return err
	}

	wrapper := req.URL.Query().Get("wrapper")

	return qq.infra.Renderer().Render(rw, ctx,
		qq.Form(ctx, data, actionx.ResponseWrapper(wrapper)),
	)
}

func (qq *AddFilePropertyCmd) Form(
	ctx ctxx.Context,
	data *AddFilePropertyCmdData,
	wrapper actionx.ResponseWrapper,
) renderable.Renderable {
	return autil.WrapWidgetWithID(
		wx.T("Add field"),
		nil,
		qq.listContent(ctx, data),
		wrapper,
		wx.DialogLayoutStable,
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

	var child wx.IWidget

	if len(properties) == 0 {
		child = &wx.EmptyState{
			Headline: wx.T("No unassigned fields available."),
			Actions: []wx.IWidget{
				&wx.Button{
					Label: wx.T("Manage fields"),
					Icon:  wx.NewIcon("tune"),
					HTMXAttrs: wx.HTMXAttrs{
						HxGet: route.ManageProperties(ctx.SpaceCtx().TenantID, ctx.SpaceCtx().SpaceID),
						HxOn:  event.CloseDialog.HxOn("click"),
					},
				},
			},
		}
	} else {
		var listItems []*wx.ListItem
		for _, propertyx := range properties {
			listItems = append(listItems, &wx.ListItem{
				Headline:       wx.Tu(propertyx.Name),
				SupportingText: wx.Tu(propertyx.Type.String()),
				Leading:        wx.NewIcon("tune"),
				HTMXAttrs: qq.actions.AddFilePropertyValueDialog.ModalLinkAttrs(
					qq.actions.AddFilePropertyValueDialog.Data(data.FileID, propertyx.ID),
					"",
				),
			})
		}

		child = &wx.List{
			Children: listItems,
		}
	}

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.listID(),
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxTrigger: event.HxTrigger(event.FilePropertyUpdated),
			HxPost:    qq.Endpoint(),
			HxVals:    util.JSON(data),
			HxTarget:  "#" + qq.listID(),
			HxSwap:    "outerHTML",
		},
		Children: child,
	}
}
