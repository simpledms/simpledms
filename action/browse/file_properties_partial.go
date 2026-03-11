package browse

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/model"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type FilePropertiesPartialData struct {
	FileID string
}

type FilePropertiesPartial struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewFilePropertiesPartial(infra *common.Infra, actions *Actions) *FilePropertiesPartial {
	config := actionx.NewConfig(
		actions.Route("file-properties-partial"),
		true,
	)
	return &FilePropertiesPartial{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *FilePropertiesPartial) Data(fileID string) *FilePropertiesPartialData {
	return &FilePropertiesPartialData{
		FileID: fileID,
	}
}

func (qq *FilePropertiesPartial) Handler(rw httpx.ResponseWriter, req *httpx.Request, ctx ctxx.Context) error {
	data, err := autil.FormData[FilePropertiesPartialData](rw, req, ctx)
	if err != nil {
		return err
	}

	return qq.infra.Renderer().Render(
		rw,
		ctx,
		qq.Widget(ctx, data),
	)
}

func (qq *FilePropertiesPartial) Widget(ctx ctxx.Context, data *FilePropertiesPartialData) *wx.ScrollableContent {
	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	assignments := ctx.SpaceCtx().TTx.FilePropertyAssignment.Query().
		Where(filepropertyassignment.FileID(filex.Data.ID)).
		WithProperty(
			func(query *enttenant.PropertyQuery) {
				query.Order(property.ByName())
			},
		).
		AllX(ctx)

	addFieldButton := &wx.Button{
		Label:     wx.T("Add field"),
		Icon:      wx.NewIcon("add"),
		StyleType: wx.ButtonStyleTypeElevated,
		HTMXAttrs: qq.actions.AddFilePropertyCmd.ModalLinkAttrs(
			qq.actions.AddFilePropertyCmd.Data(filex.Data.PublicID.String()),
			"",
		),
	}

	var children []wx.IWidget
	if len(assignments) == 0 {
		children = append(children, &wx.EmptyState{
			Headline: wx.T("No fields assigned yet."),
			Actions: []wx.IWidget{
				addFieldButton,
			},
		})
	} else {
		children = append(children, addFieldButton)
		for _, assignment := range assignments {
			if assignment.Edges.Property == nil {
				continue
			}
			children = append(children, qq.propertyAssignmentBlock(ctx, filex, assignment))
		}
	}

	return &wx.ScrollableContent{
		Widget: wx.Widget[wx.ScrollableContent]{
			ID: qq.ID(),
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxTrigger: event.HxTrigger(
				event.FilePropertyUpdated,
				event.PropertyCreated,
				event.PropertyUpdated,
				event.PropertyDeleted,
			),
			HxPost:   qq.Endpoint(),
			HxVals:   util.JSON(data),
			HxTarget: "#" + qq.ID(),
			HxSwap:   "outerHTML",
		},
		GapY:     true,
		Children: children,
		MarginY:  true,
	}
}

func (qq *FilePropertiesPartial) ID() string {
	return "fileProperties"
}

func (qq *FilePropertiesPartial) propertyAssignmentBlock(
	ctx ctxx.Context,
	filex *model.File,
	assignment *enttenant.FilePropertyAssignment,
) *wx.Column {
	propertyx := assignment.Edges.Property
	if propertyx == nil {
		return &wx.Column{}
	}

	htmxAttrsFn := func(hxTrigger string) wx.HTMXAttrs {
		return wx.HTMXAttrs{
			HxTrigger: hxTrigger,
			HxPost:    qq.actions.SetFilePropertyCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.SetFilePropertyCmd.Data(filex.Data.PublicID.String(), propertyx.ID)),
			HxInclude: "this",
		}
	}

	field, found := fieldByProperty(propertyx, assignment, htmxAttrsFn)
	if !found {
		log.Println("unknown property type: ", propertyx.Type)
		return &wx.Column{}
	}

	removeButton := &wx.Row{
		JustifyEnd: true,
		Children: &wx.Link{
			Child: wx.T("Remove").SetWrap().SetSmall(),
			HTMXAttrs: wx.HTMXAttrs{
				HxPost:    qq.actions.RemoveFilePropertyCmd.Endpoint(),
				HxVals:    util.JSON(qq.actions.RemoveFilePropertyCmd.Data(filex.Data.PublicID.String(), propertyx.ID)),
				HxSwap:    "none",
				HxConfirm: wx.T("Remove this field value?").String(ctx),
			},
		},
	}

	return &wx.Column{
		GapYSize:         wx.Gap2,
		NoOverflowHidden: true,
		AutoHeight:       true,
		Children: []wx.IWidget{
			field,
			removeButton,
		},
	}
}
