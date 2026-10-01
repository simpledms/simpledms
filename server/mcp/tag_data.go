package mcp

type TagData struct {
	TagID     string   `json:"tag_id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	GroupID   string   `json:"group_id,omitempty"`
	SubTagIDs []string `json:"sub_tag_ids,omitempty"`
}
