package filesystem

import "github.com/minio/minio-go/v7"

type PreparedUpload struct {
	FileID                    int64
	StoredFileID              int64
	OriginalFilename          string
	StorageFilenameWithoutExt string
	StorageFilename           string
	TemporaryStoragePath      string
	TemporaryStorageFilename  string
}

type PreparedUploadResult struct {
	FileInfo      *minio.UploadInfo
	FileSize      int64
	ContentSHA256 string
}
