package filesystem

type uploadMetadata struct {
	originalFilename          string
	temporaryStoragePath      string
	storageFilenameWithoutExt string
	storageFilename           string
}
