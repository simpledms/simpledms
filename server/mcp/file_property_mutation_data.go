package mcp

type FilePropertyMutationData struct {
	FileID   string           `json:"file_id"`
	Property FilePropertyData `json:"property"`
	Assigned bool             `json:"assigned"`
}
