package mcp

type MoveTagToGroupInput struct {
	TagID   string `json:"tag_id"`
	GroupID string `json:"group_id,omitempty" jsonschema:"Omit to remove the Tag from its group"`
}
