package mcp

import "time"

type DocumentNoteData struct {
	FileID           string         `json:"file_id"`
	NoteID           string         `json:"note_id"`
	Title            string         `json:"title"`
	Body             string         `json:"body"`
	BodyOffset       int            `json:"body_offset"`
	HasMoreBody      bool           `json:"has_more_body"`
	NextBodyOffset   *int           `json:"next_body_offset,omitempty"`
	Author           *NoteActorData `json:"author,omitempty"`
	AuthoredAt       *time.Time     `json:"authored_at,omitempty"`
	Editor           *NoteActorData `json:"editor,omitempty"`
	EditedAt         *time.Time     `json:"edited_at,omitempty"`
	DeletedAt        *time.Time     `json:"deleted_at,omitempty"`
	ReplacedByNoteID string         `json:"replaced_by_note_id,omitempty"`
	IsLegacy         bool           `json:"is_legacy"`
	CanChange        bool           `json:"can_change"`
}
