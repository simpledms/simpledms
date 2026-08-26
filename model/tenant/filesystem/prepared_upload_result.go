package filesystem

import "github.com/minio/minio-go/v7"

type PreparedUploadResult struct {
	FileInfo      *minio.UploadInfo
	FileSize      int64
	ContentSHA256 string
	StorageCRC32C string
	MimeType      string
}
