package browse

import (
	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	wx "github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	propertymodel "github.com/simpledms/simpledms/model/tenant/property"
	"github.com/simpledms/simpledms/ui/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/timex"
)

type SetFilePropertyCmdData struct {
	FileID     string
	PropertyID int64
}

// necessary to make request with hx-include="this" working; if just one struct is used, hx-vals
// contains all empty values and TextValue gets overwritten by empty value from hx-vals
type SetFilePropertyCmdFormData struct {
	SetFilePropertyCmdData `structs:",flatten"`
	// Value      string // TODO parse or multiple values like in db (text_value, number_value, etc.) how to handle Money then?
	TextValue     string
	NumberValue   int
	MoneyValue    float64
	CheckboxValue bool
	DateValue     timex.Date
}

// TODO in browse or property package?
type SetFilePropertyCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewSetFilePropertyCmd(infra *common.Infra, actions *Actions) *SetFilePropertyCmd {
	config := actionx.NewConfig(
		actions.Route("set-file-property-cmd"),
		false,
	).EnableCommittedResponse()
	return &SetFilePropertyCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *SetFilePropertyCmd) Data(fileID string, propertyID int64) *SetFilePropertyCmdData {
	return &SetFilePropertyCmdData{
		FileID:     fileID,
		PropertyID: propertyID,
	}
}

func (qq *SetFilePropertyCmd) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[SetFilePropertyCmdFormData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	propertyx := ctx.SpaceCtx().Space.QueryProperties().Where(property.ID(data.PropertyID)).OnlyX(ctx)
	service := propertymodel.NewFilePropertyAssignmentService()
	if propertyx.Type == fieldtype.Date && data.DateValue.IsZero() {
		if _, _, err := service.Remove(ctx, filex.Data.ID, data.PropertyID); err != nil {
			return err
		}
	} else {
		value, err := filePropertyValue(propertyx.Type, filePropertyValuesFromSet(data))
		if err != nil {
			return err
		}
		if _, _, err := service.Set(ctx, filex.Data.ID, data.PropertyID, value); err != nil {
			return err
		}
	}

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.FilePropertyUpdated.String())

	if propertyx.Type == fieldtype.Date {
		dateSuggestionsWidget := NewDateSuggestionsWidget(filex, filePropertyFieldID(propertyx.ID), propertyx.ID)
		rw.AddRenderables(dateSuggestionsWidget.Widget(
			ctx,
			data.DateValue.IsZero(),
			"outerHTML",
		))
	}

	rw.AddRenderables(wx.NewSnackbarf("«%s» saved.", propertyx.Name))

	return nil
}
