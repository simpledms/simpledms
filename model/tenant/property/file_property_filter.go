package property

import (
	"net/http"

	"github.com/simpledms/simpledms/db/enttenant"
	"github.com/simpledms/simpledms/db/enttenant/file"
	"github.com/simpledms/simpledms/db/enttenant/filepropertyassignment"
	"github.com/simpledms/simpledms/db/enttenant/predicate"
	"github.com/simpledms/simpledms/model/main/common/fieldtype"
	"github.com/simpledms/simpledms/util/e"
)

// FilePropertyFilter is a validated, typed predicate shared by Browse and MCP.
type FilePropertyFilter struct {
	propertyID int64
	operator   string
	value      FilePropertyValue
	end        FilePropertyValue
}

// NewFilePropertyFilter validates the operator and typed bounds before query composition.
func NewFilePropertyFilter(
	propertyID int64,
	operator string,
	value FilePropertyValue,
	end *FilePropertyValue,
) (FilePropertyFilter, error) {
	if operator == "" {
		operator = "equals"
		if value.typex == fieldtype.Text {
			operator = "contains"
		}
	}
	valid := propertyID > 0
	switch value.typex {
	case fieldtype.Text:
		valid = valid && (operator == "equals" || operator == "contains" || operator == "starts_with")
	case fieldtype.Number, fieldtype.Money, fieldtype.Date:
		valid = valid && (operator == "equals" || operator == "greater_than" || operator == "less_than" ||
			operator == "greater_than_or_equal" || operator == "less_than_or_equal" || operator == "between")
	case fieldtype.Checkbox:
		valid = valid && (operator == "equals" || operator == "is_checked")
	default:
		valid = false
	}
	if !valid || operator == "between" && (end == nil || end.typex != value.typex) ||
		operator != "between" && end != nil {
		return FilePropertyFilter{}, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid field filter.")
	}
	if end != nil && ((value.typex == fieldtype.Date && end.date.Before(value.date.Time)) ||
		(value.typex != fieldtype.Date && end.number < value.number)) {
		return FilePropertyFilter{}, e.NewHTTPErrorf(http.StatusBadRequest, "Invalid field filter range.")
	}
	filter := FilePropertyFilter{
		propertyID: propertyID,
		operator:   operator,
		value:      value,
	}
	if end != nil {
		filter.end = *end
	}
	return filter, nil
}

// Apply composes the validated filter with an already scoped query.
func (qq FilePropertyFilter) Apply(query *enttenant.FileQuery) *enttenant.FileQuery {
	conditions := []predicate.FilePropertyAssignment{
		filepropertyassignment.PropertyID(qq.propertyID),
	}
	switch qq.value.typex {
	case fieldtype.Text:
		switch qq.operator {
		case "contains":
			conditions = append(conditions, filepropertyassignment.TextValueContainsFold(qq.value.text))
		case "starts_with":
			conditions = append(conditions, filepropertyassignment.TextValueHasPrefix(qq.value.text))
		default:
			conditions = append(conditions, filepropertyassignment.TextValueEqualFold(qq.value.text))
		}
	case fieldtype.Number, fieldtype.Money:
		switch qq.operator {
		case "greater_than":
			conditions = append(conditions, filepropertyassignment.NumberValueGT(qq.value.number))
		case "less_than":
			conditions = append(conditions, filepropertyassignment.NumberValueLT(qq.value.number))
		case "greater_than_or_equal":
			conditions = append(conditions, filepropertyassignment.NumberValueGTE(qq.value.number))
		case "less_than_or_equal":
			conditions = append(conditions, filepropertyassignment.NumberValueLTE(qq.value.number))
		case "between":
			conditions = append(conditions,
				filepropertyassignment.NumberValueGTE(qq.value.number),
				filepropertyassignment.NumberValueLTE(qq.end.number),
			)
		default:
			conditions = append(conditions, filepropertyassignment.NumberValue(qq.value.number))
		}
	case fieldtype.Date:
		switch qq.operator {
		case "greater_than":
			conditions = append(conditions, filepropertyassignment.DateValueGT(qq.value.date))
		case "less_than":
			conditions = append(conditions, filepropertyassignment.DateValueLT(qq.value.date))
		case "greater_than_or_equal":
			conditions = append(conditions, filepropertyassignment.DateValueGTE(qq.value.date))
		case "less_than_or_equal":
			conditions = append(conditions, filepropertyassignment.DateValueLTE(qq.value.date))
		case "between":
			conditions = append(conditions,
				filepropertyassignment.DateValueGTE(qq.value.date),
				filepropertyassignment.DateValueLTE(qq.end.date),
			)
		default:
			conditions = append(conditions, filepropertyassignment.DateValue(qq.value.date))
		}
	case fieldtype.Checkbox:
		if !qq.value.checkbox {
			return query.Where(file.Or(
				file.HasPropertyAssignmentWith(
					filepropertyassignment.PropertyID(qq.propertyID),
					filepropertyassignment.Or(
						filepropertyassignment.BoolValue(false),
						filepropertyassignment.BoolValueIsNil(),
					),
				),
				file.Not(file.HasPropertyAssignmentWith(filepropertyassignment.PropertyID(qq.propertyID))),
			))
		}
		conditions = append(conditions, filepropertyassignment.BoolValue(true))
	}
	return query.Where(file.HasPropertyAssignmentWith(conditions...))
}
