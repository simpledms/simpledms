package mcp

type DocumentNoteDeletionData struct {
	FileID  string `json:"file_id"`
	NoteID  string `json:"note_id"`
	Deleted bool   `json:"deleted"`
}
