package mcp

type EditDocumentNoteInput struct {
	DocumentNoteInput
	Title string `json:"title"`
	Body  string `json:"body"`
}
