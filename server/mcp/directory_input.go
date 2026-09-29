package mcp

type DirectoryInput struct {
	DirectoryID string `json:"directory_id,omitempty"`
	Offset      int    `json:"offset,omitempty"`
	Limit       *int   `json:"limit,omitempty"`
}
