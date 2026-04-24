package file

import "github.com/simpledms/simpledms/ctxx"

type FileReadRepository interface {
	FileByPublicID(ctx ctxx.Context, filePublicID string) (*FileDTO, error)
	FileByPublicIDX(ctx ctxx.Context, filePublicID string) *FileDTO
	FileByID(ctx ctxx.Context, fileID int64) (*FileDTO, error)
	FileByParentIDAndName(ctx ctxx.Context, parentID int64, name string) (*FileDTO, error)
	FilesByIDs(ctx ctxx.Context, fileIDs []int64) ([]*FileDTO, error)
	FileExistsByNameAndParentX(ctx ctxx.Context, name string, parentID int64, isInInbox bool) bool
	FileByPublicIDWithParentX(ctx ctxx.Context, filePublicID string) *FileWithParentDTO
	FileByPublicIDWithChildrenX(ctx ctxx.Context, filePublicID string) *FileWithChildrenDTO
	FileByIDWithChildrenX(ctx ctxx.Context, fileID int64) *FileWithChildrenDTO
	FileByIDX(ctx ctxx.Context, fileID int64) *FileDTO
	FileByPublicIDWithDeletedX(ctx ctxx.Context, filePublicID string) *FileDTO
	ParentDirectoryNameByIDX(ctx ctxx.Context, parentID int64) string
}
