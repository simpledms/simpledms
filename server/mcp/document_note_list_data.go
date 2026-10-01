package mcp

type DocumentNoteListData struct {
	FileID     string             `json:"file_id"`
	Notes      []DocumentNoteData `json:"notes"`
	LegacyNote *DocumentNoteData  `json:"legacy_note,omitempty"`
	HasMore    bool               `json:"has_more"`
	NextOffset *int               `json:"next_offset,omitempty"`
}
