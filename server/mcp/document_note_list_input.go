package mcp

type DocumentNoteListInput struct {
	FileID      string `json:"file_id"`
	ShowHistory bool   `json:"show_history,omitempty"`
	Offset      int    `json:"offset,omitempty"`
	Limit       *int   `json:"limit,omitempty"`
}
