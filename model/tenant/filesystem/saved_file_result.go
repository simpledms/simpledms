package filesystem

type savedFileResult struct {
	fileSize      int64
	contentSHA256 string
	storageSize   int64
	storageSHA256 string
	storageCRC32C string
	err           error
}
