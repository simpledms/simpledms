package widget

type FileUpload struct {
	Widget[FileUpload]
	Endpoint    string
	ParentDirID string
	FileID      string
	AddToInbox  bool
}

func (qq *FileUpload) GetEndpoint() string {
	if qq.Endpoint != "" {
		return qq.Endpoint
	}
	return "/-/browse/upload-file-cmd"
}
