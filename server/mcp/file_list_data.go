package mcp

type FileListData struct {
	Files      []FileSummary `json:"files"`
	HasMore    bool          `json:"has_more"`
	NextOffset *int          `json:"next_offset,omitempty"`
}
