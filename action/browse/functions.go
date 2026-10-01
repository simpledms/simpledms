package browse

import (
	"fmt"
	"net/http"

	"github.com/simpledms/simpledms/core/ui/widget"
	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	propertymodel "github.com/simpledms/simpledms/model/tenant/property"
	"github.com/simpledms/simpledms/util/e"
	"github.com/simpledms/simpledms/util/timex"
)

func fieldByProperty(
	propertyx *enttenant.Property,
	nilableAssignment *enttenant.FilePropertyAssignment,
	htmxAttrsFn func(string) widget.HTMXAttrs,
) (widget.IWidget, bool) {
	fieldID := filePropertyFieldID(propertyx.ID)
	defaultValue := ""

	switch propertyx.Type {
	case fieldtype.Text:
		if nilableAssignment != nil {
			defaultValue = nilableAssignment.TextValue
		}
		return &widget.TextField{
			Widget: widget.Widget[widget.TextField]{
				ID: fieldID,
			},
			Label:        widget.Tu(propertyx.Name),
			Name:         "TextValue",
			Type:         "text",
			DefaultValue: defaultValue,
			HTMXAttrs:    htmxAttrsFn("change, input delay:1000ms"),
		}, true
	case fieldtype.Number:
		if nilableAssignment != nil {
			defaultValue = fmt.Sprintf("%d", nilableAssignment.NumberValue)
		}
		return &widget.TextField{
			Widget: widget.Widget[widget.TextField]{
				ID: fieldID,
			},
			Label:        widget.Tu(propertyx.Name),
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
		return &widget.TextField{
			Widget: widget.Widget[widget.TextField]{
				ID: fieldID,
			},
			Label:        widget.Tu(propertyx.Name),
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
		return &widget.TextField{
			Widget: widget.Widget[widget.TextField]{
				ID: fieldID,
			},
			Label:        widget.Tu(propertyx.Name),
			Name:         "DateValue",
			Type:         "date",
			DefaultValue: defaultValue,
			// short delay because going quickly up and down on day or month or
			// year triggers change event;
			// 30.01.2026: increased delay because 250ms was to short when
			// year gets modified manually and focus is lost on update
			HTMXAttrs: htmxAttrsFn("change delay:1000ms"),
		}, true
	case fieldtype.Checkbox:
		// TODO cannot handle nil value; is this okay?
		isChecked := false
		if nilableAssignment != nil {
			isChecked = nilableAssignment.BoolValue
		}
		return &widget.Checkbox{
			Label:     widget.Tu(propertyx.Name),
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

func filePropertyValue(
	propertyType fieldtype.FieldType,
	values filePropertyValues,
) (propertymodel.FilePropertyValue, error) {
	switch propertyType {
	case fieldtype.Text:
		return propertymodel.NewTextFilePropertyValue(values.TextValue), nil
	case fieldtype.Number:
		return propertymodel.NewNumberFilePropertyValue(int64(values.NumberValue))
	case fieldtype.Money:
		return propertymodel.NewDecimalMoneyFilePropertyValue(values.MoneyValue)
	case fieldtype.Date:
		return propertymodel.NewDateFilePropertyValue(values.DateValue)
	case fieldtype.Checkbox:
		return propertymodel.NewCheckboxFilePropertyValue(values.CheckboxValue), nil
	default:
		return propertymodel.FilePropertyValue{}, e.NewHTTPErrorf(
			http.StatusBadRequest, "Unsupported field type.",
		)
	}
}
