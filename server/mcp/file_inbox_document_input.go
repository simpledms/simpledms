package mcp

type FileInboxDocumentInput struct {
	FileID                 string `json:"file_id"`
	DestinationDirectoryID string `json:"destination_directory_id"`
	Filename               string `json:"filename,omitempty"`
	NewDirectoryName       string `json:"new_directory_name,omitempty"`
}
