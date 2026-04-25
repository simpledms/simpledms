package browse

import (
	"log"

	autil "github.com/marcobeierer/go-core/action/util"
	"github.com/marcobeierer/go-core/common"
	"github.com/marcobeierer/go-core/ctxx"
	"github.com/marcobeierer/go-core/ui/uix/events"
	"github.com/marcobeierer/go-core/ui/util"
	"github.com/marcobeierer/go-core/ui/widget"
	"github.com/marcobeierer/go-core/util/actionx"
	httpx2 "github.com/marcobeierer/go-core/util/httpx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/property"
	filemodel "github.com/simpledms/simpledms/model/tenant/file"
	"github.com/simpledms/simpledms/ui/uix/event"
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

func (qq *FilePropertiesPartial) Handler(rw httpx2.ResponseWriter, req *httpx2.Request, ctx ctxx.Context) error {
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

func (qq *FilePropertiesPartial) Widget(ctx ctxx.Context, data *FilePropertiesPartialData) *widget.ScrollableContent {
	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	assignments := ctx.SpaceCtx().TTx.FilePropertyAssignment.Query().
		Where(filepropertyassignment.FileID(filex.Data.ID)).
		WithProperty(
			func(query *enttenant.PropertyQuery) {
				query.Order(property.ByName())
			},
		).
		AllX(ctx)

	addFieldButton := &widget.Button{
		Label:     widget.T("Add field"),
		Icon:      widget.NewIcon("add"),
		StyleType: widget.ButtonStyleTypeElevated,
		HTMXAttrs: qq.actions.AddFilePropertyCmd.ModalLinkAttrs(
			qq.actions.AddFilePropertyCmd.Data(filex.Data.PublicID.String()),
			"",
		),
	}

	var children []widget.IWidget
	if len(assignments) == 0 {
		children = append(children, &widget.EmptyState{
			Headline: widget.T("No fields assigned yet."),
			Actions: []widget.IWidget{
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

	return &widget.ScrollableContent{
		Widget: widget.Widget[widget.ScrollableContent]{
			ID: qq.ID(),
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxTrigger: events.HxTrigger(
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
	filex *filemodel.File,
	assignment *enttenant.FilePropertyAssignment,
) *widget.Column {
	propertyx := assignment.Edges.Property
	if propertyx == nil {
		return &widget.Column{}
	}

	htmxAttrsFn := func(hxTrigger string) widget.HTMXAttrs {
		return widget.HTMXAttrs{
			HxTrigger: hxTrigger,
			HxPost:    qq.actions.SetFilePropertyCmd.Endpoint(),
			HxVals:    util.JSON(qq.actions.SetFilePropertyCmd.Data(filex.Data.PublicID.String(), propertyx.ID)),
			HxInclude: "this",
		}
	}

	field, found := fieldByProperty(propertyx, assignment, htmxAttrsFn)
	if !found {
		log.Println("unknown property type: ", propertyx.Type)
		return &widget.Column{}
	}

	removeButton := &widget.Row{
		JustifyEnd: true,
		Children: &widget.Link{
			Child: widget.T("Remove").SetWrap().SetSmall(),
			HTMXAttrs: widget.HTMXAttrs{
				HxPost:    qq.actions.RemoveFilePropertyCmd.Endpoint(),
				HxVals:    util.JSON(qq.actions.RemoveFilePropertyCmd.Data(filex.Data.PublicID.String(), propertyx.ID)),
				HxSwap:    "none",
				HxConfirm: widget.T("Remove this field value?").String(ctx),
			},
		},
	}

	return &widget.Column{
		GapYSize:         widget.Gap2,
		NoOverflowHidden: true,
		AutoHeight:       true,
		Children: []widget.IWidget{
			field,
			removeButton,
		},
	}
}
