package browse

import (
	"log"
	"math"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/model/common/fieldtype"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/uix/event"
	"github.com/simpledms/simpledms/util/actionx"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/timex"
)

type SetFilePropertyData struct {
	FileID     string
	PropertyID int64
}

// necessary to make request with hx-include="this" working; if just one struct is used, hx-vals
// contains all empty values and TextValue gets overwritten by empty value from hx-vals
type SetFilePropertyFormData struct {
	SetFilePropertyData `structs:",flatten"`
	// Value      string // TODO parse or multiple values like in db (text_value, number_value, etc.) how to handle Money then?
	TextValue     string
	NumberValue   int
	MoneyValue    float64
	CheckboxValue bool
	DateValue     timex.Date
}

// TODO in browse or property package?
type SetFileProperty struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewSetFileProperty(infra *common.Infra, actions *Actions) *SetFileProperty {
	config := actionx.NewConfig(
		actions.Route("set-file-property"),
		false,
	)
	return &SetFileProperty{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *SetFileProperty) Data(fileID string, propertyID int64) *SetFilePropertyData {
	return &SetFilePropertyData{
		FileID:     fileID,
		PropertyID: propertyID,
	}
}

func (qq *SetFileProperty) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[SetFilePropertyFormData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)

	// TODO check if exists and update if so
	nilableAssignment, err := filex.Data.
		QueryPropertyAssignment().Where(filepropertyassignment.PropertyID(data.PropertyID)).
		Only(ctx)
	if err != nil && !enttenant.IsNotFound(err) {
		log.Println(err)
		return err
	}

	propertyx := ctx.SpaceCtx().Space.QueryProperties().Where(property.ID(data.PropertyID)).OnlyX(ctx)

	if enttenant.IsNotFound(err) {
		query := ctx.SpaceCtx().TTx.FilePropertyAssignment.Create().
			SetSpaceID(ctx.SpaceCtx().Space.ID).
			SetFileID(filex.Data.ID).
			SetPropertyID(data.PropertyID)

		// duplicate below
		// TODO move to propertym?
		switch propertyx.Type {
		case fieldtype.Text:
			query.SetTextValue(data.TextValue)
		case fieldtype.Number:
			query.SetNumberValue(data.NumberValue)
		case fieldtype.Money:
			val := int(math.Round(data.MoneyValue * 100)) // convert to minor unit // TODO is this good enough?
			query.SetNumberValue(val)
		case fieldtype.Date:
			query.SetDateValue(data.DateValue)
		case fieldtype.Checkbox:
			query.SetBoolValue(data.CheckboxValue)
		default:
			return e.NewHTTPErrorf(http.StatusBadRequest, "Unsupported field type.")
		}

		query.ExecX(ctx)

	} else {
		query := nilableAssignment.Update()

		// duplicate above
		switch propertyx.Type {
		case fieldtype.Text:
			query.SetTextValue(data.TextValue)
		case fieldtype.Number:
			query.SetNumberValue(data.NumberValue)
		case fieldtype.Money:
			val := int(math.Round(data.MoneyValue * 100)) // convert to minor unit // TODO is this good enough?
			query.SetNumberValue(val)
		case fieldtype.Date:
			query.SetDateValue(data.DateValue)
		case fieldtype.Checkbox:
			query.SetBoolValue(data.CheckboxValue)
		default:
			return e.NewHTTPErrorf(http.StatusBadRequest, "Unsupported field type.")
		}

		query.SaveX(ctx)
	}

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.FilePropertyUpdated.String())

	rw.AddRenderables(wx.NewSnackbarf("«%s» saved.", propertyx.Name))

	return nil
}
