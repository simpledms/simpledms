package mcp

type UploadFileData struct {
	FileID    string `json:"file_id"`
	URL       string `json:"url"`
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	IsInInbox bool   `json:"is_in_inbox"`
}
