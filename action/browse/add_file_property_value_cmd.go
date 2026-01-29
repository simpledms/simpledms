package browse

import (
	"log"
	"net/http"
	"strings"

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
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/httpx"
	"github.com/simpledms/simpledms/util/timex"
)

type AddFilePropertyValueCmdData struct {
	FileID     string `form_attr_type:"hidden" validate:"required"`
	PropertyID int64  `form_attr_type:"hidden" validate:"required"`
}

type AddFilePropertyValueCmdFormData struct {
	AddFilePropertyValueCmdData `structs:",flatten"`
	TextValue                   string
	NumberValue                 int
	MoneyValue                  float64
	CheckboxValue               bool
	DateValue                   timex.Date
}

type AddFilePropertyValueCmd struct {
	infra   *common.Infra
	actions *Actions
	*actionx.Config
}

func NewAddFilePropertyValueCmd(infra *common.Infra, actions *Actions) *AddFilePropertyValueCmd {
	config := actionx.NewConfig(
		actions.Route("add-file-property-value-cmd"),
		false,
	)
	return &AddFilePropertyValueCmd{
		infra:   infra,
		actions: actions,
		Config:  config,
	}
}

func (qq *AddFilePropertyValueCmd) Data(fileID string, propertyID int64) *AddFilePropertyValueCmdData {
	return &AddFilePropertyValueCmdData{
		FileID:     fileID,
		PropertyID: propertyID,
	}
}

func (qq *AddFilePropertyValueCmd) Handler(
	rw httpx.ResponseWriter,
	req *httpx.Request,
	ctx ctxx.Context,
) error {
	data, err := autil.FormData[AddFilePropertyValueCmdFormData](rw, req, ctx)
	if err != nil {
		return err
	}

	filex := qq.infra.FileRepo.GetX(ctx, data.FileID)
	propertyx := ctx.SpaceCtx().Space.QueryProperties().Where(property.ID(data.PropertyID)).OnlyX(ctx)

	if err := qq.validateValue(req, propertyx.Type, data); err != nil {
		return err
	}

	nilableAssignment, err := ctx.SpaceCtx().TTx.FilePropertyAssignment.Query().
		Where(
			filepropertyassignment.PropertyID(data.PropertyID),
			filepropertyassignment.FileID(filex.Data.ID),
		).Only(ctx)
	if err != nil && !enttenant.IsNotFound(err) {
		log.Println(err)
		return err
	}

	if enttenant.IsNotFound(err) {
		query := ctx.SpaceCtx().TTx.FilePropertyAssignment.Create().
			SetSpaceID(ctx.SpaceCtx().Space.ID).
			SetFileID(filex.Data.ID).
			SetPropertyID(data.PropertyID)
		if err := applyPropertyValuesToCreate(query, propertyx.Type, filePropertyValuesFromAdd(data)); err != nil {
			return err
		}

		query.ExecX(ctx)
	} else {
		query := nilableAssignment.Update()
		if err := applyPropertyValuesToUpdate(query, propertyx.Type, filePropertyValuesFromAdd(data)); err != nil {
			return err
		}

		query.SaveX(ctx)
	}

	rw.Header().Set("HX-Reswap", "none")
	rw.Header().Set("HX-Trigger", event.FilePropertyUpdated.String())
	rw.AddRenderables(wx.NewSnackbarf("«%s» saved.", propertyx.Name))

	return nil
}

func (qq *AddFilePropertyValueCmd) validateValue(
	req *httpx.Request,
	propertyType fieldtype.FieldType,
	data *AddFilePropertyValueCmdFormData,
) error {
	valueProvided := func(fieldName string) bool {
		return strings.TrimSpace(req.PostForm.Get(fieldName)) != ""
	}

	switch propertyType {
	case fieldtype.Text:
		if strings.TrimSpace(data.TextValue) == "" {
			return e.NewHTTPErrorf(http.StatusBadRequest, "Value is required.")
		}
	case fieldtype.Number:
		if !valueProvided("NumberValue") {
			return e.NewHTTPErrorf(http.StatusBadRequest, "Value is required.")
		}
	case fieldtype.Money:
		if !valueProvided("MoneyValue") {
			return e.NewHTTPErrorf(http.StatusBadRequest, "Value is required.")
		}
	case fieldtype.Date:
		if !valueProvided("DateValue") || data.DateValue.IsZero() {
			return e.NewHTTPErrorf(http.StatusBadRequest, "Value is required.")
		}
	case fieldtype.Checkbox:
		// allow both checked and unchecked values
	default:
		return e.NewHTTPErrorf(http.StatusBadRequest, "Unsupported field type.")
	}

	return nil
}
