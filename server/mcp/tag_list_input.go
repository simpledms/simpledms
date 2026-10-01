package mcp

type TagListInput struct {
	MetadataListInput
	GroupID string `json:"group_id,omitempty" jsonschema:"Optional public Tag group identifier"`
}
