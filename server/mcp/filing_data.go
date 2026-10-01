package mcp

type FilingData struct {
	FileID    string `json:"file_id"`
	Name      string `json:"name"`
	ParentID  string `json:"parent_id,omitempty"`
	IsInInbox bool   `json:"is_in_inbox"`
}
