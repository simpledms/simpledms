package widget

type FileUpload struct {
	Widget[FileUpload]
	Endpoint           string
	ParentDirID        string
	FileID             string
	AddToInbox         bool
	MaxUploadSizeBytes int64
}

func (qq *FileUpload) GetEndpoint() string {
	if qq.Endpoint != "" {
		return qq.Endpoint
	}
	return "/-/browse/upload-file-cmd"
}
