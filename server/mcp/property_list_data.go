package mcp

type PropertyListData struct {
	Properties []PropertyData `json:"properties"`
	HasMore    bool           `json:"has_more"`
	NextOffset *int           `json:"next_offset,omitempty"`
}
