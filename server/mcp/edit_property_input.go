package mcp

type EditPropertyInput struct {
	PropertyID string  `json:"property_id"`
	Name       string  `json:"name"`
	Unit       *string `json:"unit,omitempty" jsonschema:"Omit to preserve the current unit; empty clears it"`
}
