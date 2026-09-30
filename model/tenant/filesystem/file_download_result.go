package filesystem

type FileDownloadResult struct {
	FilePublicID  string
	Filename      string
	VersionNumber int
	MIMEType      string
	Size          int64
	ContentSHA256 string
	Offset        int64
	Content       []byte
	HasMore       bool
}
