package mcp

type CreateTagInput struct {
	Name    string `json:"name"`
	Type    string `json:"type" jsonschema:"Simple, Group, or Super (composed Tag)"`
	GroupID string `json:"group_id,omitempty"`
}
