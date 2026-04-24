package file

import "github.com/simpledms/simpledms/ctxx"

type FileWriteRepository interface {
	CreateRootDirectory(ctx ctxx.Context, name string) (*FileDTO, error)
	CreateDirectory(ctx ctxx.Context, parentID int64, name string) (*FileDTO, error)
	CreateFile(ctx ctxx.Context, parentID int64, name string, isInInbox bool) (*FileDTO, error)
	MoveFileByIDX(ctx ctxx.Context, fileID int64, parentID int64, nilableName *string) (*FileDTO, error)
	RenameFileByIDX(ctx ctxx.Context, fileID int64, name string) error
	ResetFileOCRStateByIDX(ctx ctxx.Context, fileID int64) error
	SetFileInInboxByIDX(ctx ctxx.Context, fileID int64, isInInbox bool) error
	SetFileDocumentTypeByIDX(ctx ctxx.Context, fileID int64, nilableDocumentTypeID *int64) error
	SoftDeleteFileByIDX(ctx ctxx.Context, fileID int64, deletedBy int64) error
	SoftDeleteFilesByIDs(ctx ctxx.Context, fileIDs []int64, deletedBy int64) error
	RestoreDeletedFile(ctx ctxx.Context, filePublicID string) (*RestoreFileResultDTO, error)
	DeleteFileWithVersionsByIDX(ctx ctxx.Context, fileID int64) error
	MergeInboxFileVersion(ctx ctxx.Context, sourceFile *FileDTO, targetFile *FileDTO) error
}
