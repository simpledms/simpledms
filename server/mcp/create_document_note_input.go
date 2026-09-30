package mcp

type CreateDocumentNoteInput struct {
	FileID string `json:"file_id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}
