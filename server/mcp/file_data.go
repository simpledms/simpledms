package mcp

type FileData struct {
	FileSummary
	ParentID      string               `json:"parent_id,omitempty"`
	VersionNumber int                  `json:"version_number,omitempty"`
	MIMEType      string               `json:"mime_type,omitempty"`
	Size          int64                `json:"size"`
	ContentSHA256 string               `json:"content_sha256,omitempty"`
	DocumentType  *DocumentTypeSummary `json:"document_type,omitempty"`
	DirectTags    []TagData            `json:"direct_tags"`
	ResolvedTags  []TagData            `json:"resolved_tags"`
	Properties    []FilePropertyData   `json:"properties"`
}
