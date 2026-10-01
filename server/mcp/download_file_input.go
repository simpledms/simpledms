package mcp

type DownloadFileInput struct {
	FileID        string `json:"file_id"`
	VersionNumber *int   `json:"version_number,omitempty"`
	Offset        int64  `json:"offset,omitempty"`
	Length        *int   `json:"length,omitempty"`
}
