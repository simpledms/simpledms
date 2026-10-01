package mcp

type PropertyFilterInput struct {
	PropertyID         string  `json:"property_id"`
	Operator           string  `json:"operator"`
	TextValue          *string `json:"text_value,omitempty"`
	NumberValue        *int64  `json:"number_value,omitempty"`
	MoneyMinorUnits    *int64  `json:"money_minor_units,omitempty"`
	DateValue          *string `json:"date_value,omitempty"`
	CheckboxValue      *bool   `json:"checkbox_value,omitempty"`
	EndNumberValue     *int64  `json:"end_number_value,omitempty"`
	EndMoneyMinorUnits *int64  `json:"end_money_minor_units,omitempty"`
	EndDateValue       *string `json:"end_date_value,omitempty"`
}
