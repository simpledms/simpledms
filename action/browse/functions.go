package browse

import (
	"fmt"
	"math"
	"net/http"

	autil "github.com/simpledms/simpledms/action/util"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/common/fieldtype"
	wx "github.com/simpledms/simpledms/ui/widget"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/timex"
)

func fieldByProperty(
	propertyx *enttenant.Property,
	nilableAssignment *enttenant.FilePropertyAssignment,
	htmxAttrsFn func(string) wx.HTMXAttrs,
) (wx.IWidget, bool) {
	fieldID := autil.GenerateID(propertyx.Name) // TODO unique enough?
	defaultValue := ""

	switch propertyx.Type {
	case fieldtype.Text:
		if nilableAssignment != nil {
			defaultValue = nilableAssignment.TextValue
		}
		return &wx.TextField{
			Widget: wx.Widget[wx.TextField]{
				ID: fieldID,
			},
			Label:        wx.Tu(propertyx.Name),
			Name:         "TextValue",
			Type:         "text",
			DefaultValue: defaultValue,
			HTMXAttrs:    htmxAttrsFn("change, input delay:1000ms"),
		}, true
	case fieldtype.Number:
		if nilableAssignment != nil {
			defaultValue = fmt.Sprintf("%d", nilableAssignment.NumberValue)
		}
		return &wx.TextField{
			Widget: wx.Widget[wx.TextField]{
				ID: fieldID,
			},
			Label:        wx.Tu(propertyx.Name),
			Name:         "NumberValue",
			Type:         "number",
			DefaultValue: defaultValue,
			// `change` event doesn't work because a change is triggered all the time a user uses arrow increase/decrease
			HTMXAttrs: htmxAttrsFn("input delay:1000ms"),
		}, true
	case fieldtype.Money:
		if nilableAssignment != nil {
			val := float64(nilableAssignment.NumberValue) / 100.0
			defaultValue = fmt.Sprintf("%.2f", val)
		}
		return &wx.TextField{
			Widget: wx.Widget[wx.TextField]{
				ID: fieldID,
			},
			Label:        wx.Tu(propertyx.Name),
			Name:         "MoneyValue",
			Type:         "number",
			Step:         "0.01",
			DefaultValue: defaultValue,
			// `change` event doesn't work because a change is triggered all the time a user uses arrow increase/decrease
			HTMXAttrs: htmxAttrsFn("input delay:1000ms"),
		}, true
	case fieldtype.Date:
		if nilableAssignment != nil && !nilableAssignment.DateValue.IsZero() {
			defaultValue = nilableAssignment.DateValue.Format("2006-01-02")
		}
		return &wx.TextField{
			Widget: wx.Widget[wx.TextField]{
				ID: fieldID,
			},
			Label:        wx.Tu(propertyx.Name),
			Name:         "DateValue",
			Type:         "date",
			DefaultValue: defaultValue,
			// short delay because going quickly up and down on day or month or year triggers change event
			HTMXAttrs: htmxAttrsFn("change delay:250ms"),
		}, true
	case fieldtype.Checkbox:
		// TODO cannot handle nil value; is this okay?
		isChecked := false
		if nilableAssignment != nil {
			isChecked = nilableAssignment.BoolValue
		}
		return &wx.Checkbox{
			Label:     wx.Tu(propertyx.Name),
			Name:      "CheckboxValue",
			IsChecked: isChecked,
			HTMXAttrs: htmxAttrsFn("change"),
		}, true
	}
	return nil, false
}

type filePropertyValues struct {
	TextValue     string
	NumberValue   int
	MoneyValue    float64
	CheckboxValue bool
	DateValue     timex.Date
}

func filePropertyValuesFromAdd(data *AddFilePropertyValueCmdFormData) filePropertyValues {
	return filePropertyValues{
		TextValue:     data.TextValue,
		NumberValue:   data.NumberValue,
		MoneyValue:    data.MoneyValue,
		CheckboxValue: data.CheckboxValue,
		DateValue:     data.DateValue,
	}
}

func filePropertyValuesFromSet(data *SetFilePropertyCmdFormData) filePropertyValues {
	return filePropertyValues{
		TextValue:     data.TextValue,
		NumberValue:   data.NumberValue,
		MoneyValue:    data.MoneyValue,
		CheckboxValue: data.CheckboxValue,
		DateValue:     data.DateValue,
	}
}

func applyPropertyValuesToCreate(
	query *enttenant.FilePropertyAssignmentCreate,
	propertyType fieldtype.FieldType,
	values filePropertyValues,
) error {
	switch propertyType {
	case fieldtype.Text:
		query.SetTextValue(values.TextValue)
	case fieldtype.Number:
		query.SetNumberValue(values.NumberValue)
	case fieldtype.Money:
		// convert to minor unit // TODO is this good enough?
		val := int(math.Round(values.MoneyValue * 100))
		query.SetNumberValue(val)
	case fieldtype.Date:
		query.SetDateValue(values.DateValue)
	case fieldtype.Checkbox:
		query.SetBoolValue(values.CheckboxValue)
	default:
		return e.NewHTTPErrorf(http.StatusBadRequest, "Unsupported field type.")
	}

	return nil
}

func applyPropertyValuesToUpdate(
	query *enttenant.FilePropertyAssignmentUpdateOne,
	propertyType fieldtype.FieldType,
	values filePropertyValues,
) error {
	switch propertyType {
	case fieldtype.Text:
		query.SetTextValue(values.TextValue)
	case fieldtype.Number:
		query.SetNumberValue(values.NumberValue)
	case fieldtype.Money:
		val := int(math.Round(values.MoneyValue * 100))
		query.SetNumberValue(val)
	case fieldtype.Date:
		query.SetDateValue(values.DateValue)
	case fieldtype.Checkbox:
		query.SetBoolValue(values.CheckboxValue)
	default:
		return e.NewHTTPErrorf(http.StatusBadRequest, "Unsupported field type.")
	}

	return nil
}
