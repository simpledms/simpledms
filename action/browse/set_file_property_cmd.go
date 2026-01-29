package browse

import (
	"log"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/common"
	"github.com/simpledms/simpledms/ctxx"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/property"
	"github.com/simpledms/simpledms/model/common/fieldtype"
	"github.com/simpledms/simpledms/ui/uix/event"
	wx "github.com/simpledms/simpledms/ui/widget"
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
	)
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

		if err := applyPropertyValuesToCreate(query, propertyx.Type, filePropertyValuesFromSet(data)); err != nil {
			return err
		}

		query.ExecX(ctx)

	} else if propertyx.Type == fieldtype.Date && data.DateValue.IsZero() {
		ctx.SpaceCtx().TTx.FilePropertyAssignment.Delete().Where(
			filepropertyassignment.PropertyID(data.PropertyID),
			filepropertyassignment.FileID(filex.Data.ID),
		).ExecX(ctx)
	} else {
		query := nilableAssignment.Update()

		if err := applyPropertyValuesToUpdate(query, propertyx.Type, filePropertyValuesFromSet(data)); err != nil {
			return err
		}

		query.SaveX(ctx)
	}

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.FilePropertyUpdated.String())

	rw.AddRenderables(wx.NewSnackbarf("«%s» saved.", propertyx.Name))

	return nil
}
