package browse

import (
	"fmt"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/model/common/fieldtype"
	"github.com/simpledms/simpledms/ui/renderable"
	"github.com/simpledms/simpledms/ui/util"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
)

type AddFilePropertyValueDialog struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewAddFilePropertyValueDialog(infra *common.Infra, actions *Actions) *AddFilePropertyValueDialog {
	config := actionx.NewConfig(
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

func (qq *AddFilePropertyValueDialog) ModalLinkAttrs(data *AddFilePropertyValueCmdData, hxTargetForm string) wx.HTMXAttrs {
	return wx.HTMXAttrs{
		HxPost:        qq.FormEndpointWithParams(actionx.ResponseWrapperDialog, hxTargetForm),
		HxVals:        util.JSON(data),
		LoadInPopover: true,
	}
}

func (qq *AddFilePropertyValueDialog) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	return qq.FormHandler(rw, req, ctx)
}

func (qq *AddFilePropertyValueDialog) FormHandler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormDataX[AddFilePropertyValueCmdFormData](rw, req, ctx, true)
	if err != nil {
		return err
	}

	wrapper := req.URL.Query().Get("wrapper")

	return qq.infra.Renderer().Render(rw, ctx,
		qq.Form(ctx, data, actionx.ResponseWrapper(wrapper)),
	)
}

func (qq *AddFilePropertyValueDialog) Form(
	ctx ctxx.Context,
	data *AddFilePropertyValueCmdFormData,
	wrapper actionx.ResponseWrapper,
) renderable.Renderable {
	propertyx := ctx.SpaceCtx().Space.QueryProperties().Where(property.ID(data.PropertyID)).OnlyX(ctx)

	var valueField wx.IWidget

	switch propertyx.Type {
	case fieldtype.Text:
		valueField = &wx.TextField{
			Label: wx.Tu(propertyx.Name),
			Name:  "TextValue",
			Type:  "text",
		}
	case fieldtype.Number:
		valueField = &wx.TextField{
			Label: wx.Tu(propertyx.Name),
			Name:  "NumberValue",
			Type:  "number",
		}
	case fieldtype.Money:
		valueField = &wx.TextField{
			Label: wx.Tu(propertyx.Name),
			Name:  "MoneyValue",
			Type:  "number",
			Step:  "0.01",
		}
	case fieldtype.Date:
		valueField = &wx.TextField{
			Label: wx.Tu(propertyx.Name),
			Name:  "DateValue",
			Type:  "date",
		}
	case fieldtype.Checkbox:
		valueField = &wx.Checkbox{
			Label: wx.Tu(propertyx.Name),
			Name:  "CheckboxValue",
		}
	default:
		valueField = wx.T("Unsupported field type.")
	}

	form := &wx.Form{
		Widget: wx.Widget[wx.Form]{
			ID: qq.formID(),
		},
		HTMXAttrs: wx.HTMXAttrs{
			HxPost: qq.actions.AddFilePropertyValueCmd.Endpoint(),
		},
		Children: []wx.IWidget{
			&wx.Container{
				GapY: true,
				Child: []wx.IWidget{
					&wx.TextField{
						Name:         "FileID",
						Type:         "hidden",
						DefaultValue: data.FileID,
					},
					&wx.TextField{
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
		wx.T("Add field"),
		wx.T("Save"),
		form,
		wrapper,
		wx.DialogLayoutStable,
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
