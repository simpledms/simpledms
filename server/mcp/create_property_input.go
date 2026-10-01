package mcp

type CreatePropertyInput struct {
	Name string `json:"name"`
	Type string `json:"type" jsonschema:"Text, Number, Money, Date, or Checkbox"`
	Unit string `json:"unit,omitempty"`
}
