package filesystem

type PreparedAccountUpload struct {
	TemporaryFileID           int64
	OriginalFilename          string
	StorageFilenameWithoutExt string
	StorageFilename           string
	StoragePath               string
}
