package mcp

type DirectoryMutationData struct {
	DirectoryID string `json:"directory_id"`
	Name        string `json:"name"`
	ParentID    string `json:"parent_id"`
}
