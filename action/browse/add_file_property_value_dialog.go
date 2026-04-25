package browse

import (
	"fmt"

	autil "github.com/simpledms/simpledms/core/action/util"
	"github.com/simpledms/simpledms/core/common"
	"github.com/simpledms/simpledms/core/model/common/fieldtype"
	"github.com/simpledms/simpledms/core/ui/renderable"
	"github.com/simpledms/simpledms/core/ui/util"
	"github.com/simpledms/simpledms/core/ui/widget"
	actionx2 "github.com/simpledms/simpledms/core/util/actionx"
	httpx2 "github.com/simpledms/simpledms/core/util/httpx"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/property"
)

type AddFilePropertyValueDialog struct {
	infra   *common.Infra
	actions *Actions
	*actionx2.Config
}

func NewAddFilePropertyValueDialog(infra *common.Infra, actions *Actions) *AddFilePropertyValueDialog {
	config := actionx2.NewConfig(
		actions.Route("add-file-property-value-dialog"),
		true,
	).SetUsesSeparatedCmd(true)
	return &AddFilePropertyValueDialog{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *AddFilePropertyValueDialog) Data(fileID string, propertyID int64) *AddFilePropertyValueCmdData {
	return &AddFilePropertyValueCmdData{
		FileID:     fileID,
		PropertyID: propertyID,
	}
}

func (qq *AddFilePropertyValueDialog) ModalLinkAttrs(data *AddFilePropertyValueCmdData, hxTargetForm string) widget.HTMXAttrs {
	return widget.HTMXAttrs{
		HxPost:        qq.FormEndpointWithParams(actionx2.ResponseWrapperDialog, hxTargetForm),
		HxVals:        util.JSON(data),
		LoadInPopover: true,
	}
}

func (qq *AddFilePropertyValueDialog) Handler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	return qq.FormHandler(rw, req, ctx)
}

func (qq *AddFilePropertyValueDialog) FormHandler(
	rw httpx2.ResponseWriter,
	req *httpx2.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormDataX[AddFilePropertyValueCmdFormData](rw, req, ctx, true)
	if err != nil {
		return err
	}

	wrapper := req.URL.Query().Get("wrapper")

	return qq.infra.Renderer().Render(rw, ctx,
		qq.Form(ctx, data, actionx2.ResponseWrapper(wrapper)),
	)
}

func (qq *AddFilePropertyValueDialog) Form(
	ctx ctxx.Context,
	data *AddFilePropertyValueCmdFormData,
	wrapper actionx2.ResponseWrapper,
) renderable.Renderable {
	propertyx := ctx.SpaceCtx().Space.QueryProperties().Where(property.ID(data.PropertyID)).OnlyX(ctx)

	var valueField widget.IWidget

	switch propertyx.Type {
	case fieldtype.Text:
		valueField = &widget.TextField{
			Label: widget.Tu(propertyx.Name),
			Name:  "TextValue",
			Type:  "text",
		}
	case fieldtype.Number:
		valueField = &widget.TextField{
			Label: widget.Tu(propertyx.Name),
			Name:  "NumberValue",
			Type:  "number",
		}
	case fieldtype.Money:
		valueField = &widget.TextField{
			Label: widget.Tu(propertyx.Name),
			Name:  "MoneyValue",
			Type:  "number",
			Step:  "0.01",
		}
	case fieldtype.Date:
		valueField = &widget.TextField{
			Label: widget.Tu(propertyx.Name),
			Name:  "DateValue",
			Type:  "date",
		}
	case fieldtype.Checkbox:
		valueField = &widget.Checkbox{
			Label: widget.Tu(propertyx.Name),
			Name:  "CheckboxValue",
		}
	default:
		valueField = widget.T("Unsupported field type.")
	}

	form := &widget.Form{
		Widget: widget.Widget[widget.Form]{
			ID: qq.formID(),
		},
		HTMXAttrs: widget.HTMXAttrs{
			HxPost: qq.actions.AddFilePropertyValueCmd.Endpoint(),
		},
		Children: []widget.IWidget{
			&widget.Container{
				GapY: true,
				Child: []widget.IWidget{
					&widget.TextField{
						Name:         "FileID",
						Type:         "hidden",
						DefaultValue: data.FileID,
					},
					&widget.TextField{
						Name:         "PropertyID",
						Type:         "hidden",
						DefaultValue: fmt.Sprintf("%d", data.PropertyID),
					},
					valueField,
				},
			},
		},
	}

	return autil.WrapWidgetWithID(
		widget.T("Add field"),
		widget.T("Save"),
		form,
		wrapper,
		widget.DialogLayoutStable,
		qq.popoverID(),
		qq.formID(),
	)
}

func (qq *AddFilePropertyValueDialog) popoverID() string {
	return "addFilePropertyValuePopover"
}

func (qq *AddFilePropertyValueDialog) formID() string {
	return "addFilePropertyValueForm"
}
