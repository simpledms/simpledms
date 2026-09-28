package mcp

type FileSummary struct {
	FileID       string `json:"file_id"`
	URL          string `json:"url"`
	Name         string `json:"name"`
	IsDirectory  bool   `json:"is_directory"`
	IsInInbox    bool   `json:"is_in_inbox"`
	Source       string `json:"source"`
	OCRAvailable bool   `json:"ocr_available"`
}
