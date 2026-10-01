package mcp

type TagListData struct {
	Tags       []TagData `json:"tags"`
	HasMore    bool      `json:"has_more"`
	NextOffset *int      `json:"next_offset,omitempty"`
}
