package mcp

type DownloadFileData struct {
	FileID        string `json:"file_id"`
	Filename      string `json:"filename"`
	VersionNumber int    `json:"version_number"`
	MIMEType      string `json:"mime_type"`
	Size          int64  `json:"size"`
	ContentSHA256 string `json:"content_sha256,omitempty"`
	Offset        int64  `json:"offset"`
	ContentBase64 string `json:"content_base64"`
	HasMore       bool   `json:"has_more"`
	NextOffset    *int64 `json:"next_offset,omitempty"`
}
