package mcp

type DirectoryData struct {
	DirectoryID string        `json:"directory_id"`
	Name        string        `json:"name"`
	ParentID    string        `json:"parent_id,omitempty"`
	Children    []FileSummary `json:"children"`
	HasMore     bool          `json:"has_more"`
	NextOffset  *int          `json:"next_offset,omitempty"`
}
