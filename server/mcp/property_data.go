package mcp

type PropertyData struct {
	PropertyID string `json:"property_id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Unit       string `json:"unit,omitempty"`
}
