package mcp

type GetDocumentNoteInput struct {
	DocumentNoteInput
	Offset int  `json:"offset,omitempty"`
	Length *int `json:"length,omitempty"`
}
