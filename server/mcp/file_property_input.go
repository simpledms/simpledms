package mcp

type FilePropertyInput struct {
	FileID     string `json:"file_id" jsonschema:"Public file identifier"`
	PropertyID string `json:"property_id" jsonschema:"Public field identifier"`
}
