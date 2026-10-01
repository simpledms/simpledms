package mcp

type InboxData struct {
	Files      []FileSummary `json:"files"`
	HasMore    bool          `json:"has_more"`
	NextOffset *int          `json:"next_offset,omitempty"`
}
