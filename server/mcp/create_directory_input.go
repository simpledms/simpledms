package mcp

type CreateDirectoryInput struct {
	ParentDirectoryID string `json:"parent_directory_id"`
	Name              string `json:"name"`
}
