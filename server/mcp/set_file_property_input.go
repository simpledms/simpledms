package mcp

type SetFilePropertyInput struct {
	FileID          string  `json:"file_id" jsonschema:"Public file identifier"`
	PropertyID      string  `json:"property_id" jsonschema:"Public field identifier"`
	TextValue       *string `json:"text_value,omitempty"`
	NumberValue     *int64  `json:"number_value,omitempty"`
	MoneyMinorUnits *int64  `json:"money_minor_units,omitempty"`
	DateValue       *string `json:"date_value,omitempty" jsonschema:"Date in YYYY-MM-DD format"`
	CheckboxValue   *bool   `json:"checkbox_value,omitempty"`
}
