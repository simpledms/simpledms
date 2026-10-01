package mcp

type TextData struct {
	FileID        string `json:"file_id"`
	VersionNumber int    `json:"version_number"`
	Available     bool   `json:"available"`
	Text          string `json:"text"`
	HasMore       bool   `json:"has_more"`
	NextOffset    *int   `json:"next_offset,omitempty"`
}
