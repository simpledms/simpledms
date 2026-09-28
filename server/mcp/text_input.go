package mcp

type TextInput struct {
	FileID string `json:"file_id"`
	Offset int    `json:"offset,omitempty"`
	Length *int   `json:"length,omitempty"`
}
