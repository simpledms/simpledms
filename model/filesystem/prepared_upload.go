package filesystem

type PreparedUpload struct {
	FileID                    int64
	StoredFileID              int64
	OriginalFilename          string
	StorageFilenameWithoutExt string
	StorageFilename           string
	TemporaryStoragePath      string
	TemporaryStorageFilename  string
}
