package mcp

type FileData struct {
	FileSummary
	ParentID      string `json:"parent_id,omitempty"`
	VersionNumber int    `json:"version_number,omitempty"`
	MIMEType      string `json:"mime_type,omitempty"`
	Size          int64  `json:"size"`
	ContentSHA256 string `json:"content_sha256,omitempty"`
}
