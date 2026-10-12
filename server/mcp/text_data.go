package mcp

const (
	TextFormatMarkdown = "markdown"
	TextFormatText     = "text"
)

type TextData struct {
	FileID        string `json:"file_id"`
	VersionNumber int    `json:"version_number"`
	Available     bool   `json:"available"`
	// Format is TextFormatMarkdown or TextFormatText if Available.
	Format     string `json:"format,omitempty"`
	Text       string `json:"text"`
	HasMore    bool   `json:"has_more"`
	NextOffset *int   `json:"next_offset,omitempty"`
}
