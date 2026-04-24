package file

import "github.com/simpledms/simpledms/ctxx"

type FileQueryRepository interface {
	BrowseFilesX(ctx ctxx.Context, filter *BrowseFileQueryFilterDTO) *BrowseFileQueryResultDTO
	InboxFilesX(ctx ctxx.Context, filter *InboxFileQueryFilterDTO) []*FileWithChildrenDTO
	TrashFilesX(ctx ctxx.Context) []*FileDTO
	InboxAssignmentSuggestionDirectoriesX(
		ctx ctxx.Context,
		searchQuery string,
		limit int,
	) []*FileWithChildrenDTO
}
