package mcp

type RenameFileInput struct {
	FileID      string `json:"file_id"`
	NewFilename string `json:"new_filename"`
}
