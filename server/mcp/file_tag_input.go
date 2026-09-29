package mcp

type FileTagInput struct {
	FileID string `json:"file_id" jsonschema:"Public file identifier"`
	TagID  string `json:"tag_id" jsonschema:"Public Tag identifier"`
}
