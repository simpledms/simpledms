package filesystem

type PreparedUpload struct {
	StoredFileID              int64
	OriginalFilename          string
	StorageFilenameWithoutExt string
	StorageFilename           string
	TemporaryStoragePath      string
	TemporaryStorageFilename  string
}
