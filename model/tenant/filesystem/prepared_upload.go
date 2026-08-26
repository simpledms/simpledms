package filesystem

import (
	"github.com/simpledms/simpledms/model/main/common/filesource"
)

type PreparedUpload struct {
	FileID                    int64
	FilePublicID              string
	IsNewFile                 bool
	StoredFileID              int64
	OriginalFilename          string
	ParentDirFileID           int64
	IsInInbox                 bool
	Source                    filesource.FileSource
	StorageFilenameWithoutExt string
	StorageFilename           string
	TemporaryStoragePath      string
	TemporaryStorageFilename  string
}
