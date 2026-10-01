package mcp

type CreateAndAssignTagInput struct {
	CreateTagInput
	FileID string `json:"file_id"`
}
