package mcp

type UploadFileInput struct {
	Filename      string `json:"filename"`
	ContentBase64 string `json:"content_base64"`
}
