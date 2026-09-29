package mcp

type FilePropertyData struct {
	FileID          string  `json:"file_id,omitempty"`
	PropertyID      string  `json:"property_id"`
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Unit            string  `json:"unit,omitempty"`
	TextValue       *string `json:"text_value,omitempty"`
	NumberValue     *int64  `json:"number_value,omitempty"`
	MoneyMinorUnits *int64  `json:"money_minor_units,omitempty"`
	DateValue       *string `json:"date_value,omitempty"`
	CheckboxValue   *bool   `json:"checkbox_value,omitempty"`
}
